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
	"alert-userbot/internal/filter"
	"alert-userbot/internal/telegram"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Neptun → Telegram MTProto forwarder for Kyiv city")

	cfg, err := config.Load()
	if err != nil {
		logger.Error("Configuration error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Configuration loaded",
		slog.String("neptun_url", cfg.NeptunWSURL),
		slog.Int("tg_api_id", cfg.TelegramAPIID),
		slog.String("source_channel", cfg.SourceChannel),
		slog.String("destination_chat_id", cfg.DestinationChatID),
		slog.String("session_file", cfg.SessionFile),
		slog.Int("queue_capacity", cfg.QueueCapacity))

	state := alert.NewKyivAlertState(logger)
	textFilter := filter.NewTextFilter(cfg.SkipPatterns)

	bot := telegram.NewUserBot(
		cfg.TelegramAPIID,
		cfg.TelegramAPIHash,
		cfg.TelegramPhone,
		cfg.TelegramPassword,
		cfg.TelegramAuthCode,
		cfg.SessionFile,
		cfg.SourceChannel,
		cfg.DestinationChatID,
		state,
		textFilter,
		cfg.QueueCapacity,
		logger,
	)

	neptunClient := client.NewNeptunClient(
		cfg.NeptunWSURL,
		cfg.MinReconnectInterval,
		cfg.MaxReconnectInterval,
		state,
		logger,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mode := telegram.ModeDaemon
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-test-notify":
			mode = telegram.ModeTestNotify
		case "-test-forward":
			mode = telegram.ModeTestForward
		}
	}

	runErr := make(chan error, 1)
	go func() {
		runErr <- bot.Run(ctx, mode)
	}()

	// NEPTUN only matters for daemon mode.
	if mode == telegram.ModeDaemon {
		go neptunClient.Start(ctx)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", slog.String("signal", sig.String()))
		cancel()
	}()

	if mode == telegram.ModeDaemon {
		go func() {
			ticker := time.NewTicker(60 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					processed, alertFrames := neptunClient.GetStats()
					forwarded, skipped, filtered := bot.Stats()
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
	}

	err = <-runErr
	if err != nil {
		logger.Error("userbot failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("Shutting down forwarder...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	<-shutdownCtx.Done()
	logger.Info("Forwarder stopped")
}
