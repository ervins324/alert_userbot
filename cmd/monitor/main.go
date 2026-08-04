package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"alert-userbot/config"
	"alert-userbot/internal/client"
	"alert-userbot/internal/filter"
	"alert-userbot/internal/notifier"
)

func main() {
	// Initialize structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("Starting Neptun Air Defense Alert Daemon for Kyiv & Kyiv Oblast")

	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Configuration error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully",
		slog.String("neptun_url", cfg.NeptunWSURL),
		slog.String("chat_id", cfg.TelegramChatID),
		slog.Int("worker_count", cfg.WorkerCount),
		slog.Int("queue_capacity", cfg.QueueCapacity))

	// Initialize components
	kyivFilter := filter.NewKyivFilter()

	telegramNotifier := notifier.NewTelegramNotifier(
		cfg.TelegramBotToken,
		cfg.TelegramChatID,
		cfg.WorkerCount,
		cfg.QueueCapacity,
		cfg.HTTPTimeout,
		logger,
	)
	defer telegramNotifier.Close()

	neptunClient := client.NewNeptunClient(
		cfg.NeptunWSURL,
		cfg.MinReconnectInterval,
		cfg.MaxReconnectInterval,
		kyivFilter,
		telegramNotifier,
		logger,
	)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle OS signals (SIGINT, SIGTERM)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", slog.String("signal", sig.String()))
		cancel()
	}()

	// Start Neptun WebSocket client in background
	go neptunClient.Start(ctx)

	// Periodic status heartbeat
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				processed, matched := neptunClient.GetStats()
				connected := neptunClient.IsConnected()
				logger.Info("Daemon heartbeat status",
					slog.Bool("connected", connected),
					slog.Int64("total_processed", processed),
					slog.Int64("total_matched", matched))
			}
		}
	}()

	// Send startup notification
	telegramNotifier.Notify("🟢 <b>Neptun Air Defense Monitor Daemon</b> started successfully for Kyiv & Kyiv Oblast.")

	// Wait for context cancellation
	<-ctx.Done()

	logger.Info("Shutting down daemon gracefully...")

	// Give a small window for final flushes
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()

	<-shutdownCtx.Done()
	logger.Info("Neptun Air Defense Alert Daemon stopped successfully")
}
