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

func snippet(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

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
		slog.Float64("poll_interval_sec", cfg.PollInterval.Seconds()),
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

	fw := forwarder.New(state, bot, forwarder.NewTextFilter(cfg.SkipPatterns), cfg.HTTPTimeout, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test modes (run before the daemon loop):
	switch {
	case len(os.Args) > 1 && os.Args[1] == "-test-scrape":
		msgs, err := channelScraper.Fetch(ctx)
		if err != nil {
			logger.Error("channel scrape failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("scraped messages", slog.Int("count", len(msgs)))
		for i, m := range msgs {
			if i >= 3 {
				break
			}
			logger.Info("message", slog.Int("id", m.ID), slog.String("text", snippet(m.Text, 80)), slog.String("photo", m.PhotoURL))
		}
		return
	case len(os.Args) > 1 && os.Args[1] == "-test-forward":
		// Simulate an active alert and forward the latest channel post.
		state.SetActive(true)
		msgs, err := channelScraper.Fetch(ctx)
		if err != nil {
			logger.Error("channel scrape failed", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if len(msgs) == 0 {
			logger.Error("no messages found on channel page")
			os.Exit(1)
		}
		latest := msgs[len(msgs)-1]
		logger.Info("forwarding latest channel message (alert simulated as active)",
			slog.Int("id", latest.ID), slog.String("text", snippet(latest.Text, 80)))
		fw.Forward(ctx, latest)
		time.Sleep(1 * time.Second)
		return
	}

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
				forwarded, skipped, filtered := fw.Stats()
				logger.Info("daemon heartbeat",
					slog.Bool("neptun_connected", neptunClient.IsConnected()),
					slog.Bool("kyiv_alert_active", state.IsActive()),
					slog.Int64("ws_frames_processed", processed),
					slog.Int64("alerts_frames", alertFrames),
					slog.Int64("messages_forwarded", forwarded),
					slog.Int64("messages_skipped", skipped),
					slog.Int64("messages_filtered", filtered))
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
