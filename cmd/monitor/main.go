package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"alert-userbot/config"
	"alert-userbot/internal/alert"
	"alert-userbot/internal/client"
	"alert-userbot/internal/forwarder"
	"alert-userbot/internal/notifier"
	"alert-userbot/internal/scraper"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	testNotify := len(os.Args) > 1 && os.Args[1] == "-test-notify"

	logger.Info("Starting Neptun → Telegram channel forwarder for Kyiv city")

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Configuration error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Configuration loaded",
		slog.String("neptun_url", cfg.NeptunWSURL),
		slog.String("source_channel", cfg.SourceChannel),
		slog.String("destination_chat_id", cfg.DestinationChatID),
		slog.Duration("poll_interval", cfg.PollInterval),
		slog.Int("worker_count", cfg.WorkerCount),
		slog.Int("queue_capacity", cfg.QueueCapacity))

	bot := notifier.NewTelegramBot(
		cfg.TelegramBotToken,
		cfg.DestinationChatID,
		cfg.WorkerCount,
		cfg.QueueCapacity,
		cfg.HTTPTimeout,
		logger,
	)
	defer bot.Close()

	// Test mode: verify Telegram delivery end-to-end, then exit.
	if testNotify {
		if bot.SendMessage("🔔 <b>Test notification</b> — Neptun forwarder is working. If you receive this, your bot + chat are configured correctly.") {
			logger.Info("Test notification enqueued. Check your Telegram chat.")
			time.Sleep(1 * time.Second)
		} else {
			logger.Error("Test notification dropped (queue full)")
			os.Exit(1)
		}
		return
	}

	state := alert.NewKyivAlertState(logger)

	neptunClient := client.NewNeptunClient(
		cfg.NeptunWSURL,
		cfg.MinReconnectInterval,
		cfg.MaxReconnectInterval,
		state,
		logger,
	)

	channelScraper := scraper.NewChannelScraper(
		cfg.SourceChannel,
		cfg.PollInterval,
		cfg.HTTPTimeout,
		logger,
	)

	fw := forwarder.New(state, bot, cfg.HTTPTimeout, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", slog.String("signal", sig.String()))
		cancel()
	}()

	msgCh := make(chan scraper.Message, 128)

	go neptunClient.Start(ctx)
	go channelScraper.Start(ctx, msgCh)
	go fw.Run(ctx, msgCh)

	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				processed, alertFrames := neptunClient.GetStats()
				forwarded, skipped := fw.Stats()
				logger.Info("daemon heartbeat",
					slog.Bool("neptun_connected", neptunClient.IsConnected()),
					slog.Bool("kyiv_alert_active", state.IsActive()),
					slog.Int64("ws_frames_processed", processed),
					slog.Int64("alerts_frames", alertFrames),
					slog.Int64("messages_forwarded", forwarded),
					slog.Int64("messages_skipped", skipped))
			}
		}
	}()

	<-ctx.Done()
	logger.Info("Shutting down forwarder...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	<-shutdownCtx.Done()
	logger.Info("Forwarder stopped")
}
