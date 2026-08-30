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
	"alert-userbot/internal/command"
	"alert-userbot/internal/filter"
	"alert-userbot/internal/geomap"
	"alert-userbot/internal/geoparse"
	"alert-userbot/internal/notifier"
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

	bot := notifier.NewTelegramBot(
		cfg.TelegramBotToken,
		cfg.DestinationChatID,
		cfg.HTTPTimeout,
		logger,
	)

	ub := telegram.NewUserBot(
		cfg.TelegramAPIID,
		cfg.TelegramAPIHash,
		cfg.TelegramPhone,
		cfg.TelegramPassword,
		cfg.TelegramAuthCode,
		cfg.SessionFile,
		cfg.SourceChannel,
		state,
		textFilter,
		bot,
		cfg.QueueCapacity,
		cfg.ForceAlert,
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

	if len(os.Args) > 1 && os.Args[1] == "-test-map" {
		sampleText := "БпЛА курсом на Дарницький район (Позняки)"
		if len(os.Args) > 2 {
			sampleText = os.Args[2]
		}
		logger.Info("Generating test map", slog.String("query", sampleText))
		loc := geoparse.ExtractLocation(sampleText)
		if loc == nil {
			logger.Error("Could not extract location from test query", slog.String("query", sampleText))
			os.Exit(1)
		}
		imgData, err := geomap.RenderKyivMap(loc, cfg.GoogleMapsAPIKey)
		if err != nil {
			logger.Error("Failed to render map", slog.String("error", err.Error()))
			os.Exit(1)
		}
		if err := bot.SendPhoto(imgData, "🗺️ Тестова карта: "+loc.Description); err != nil {
			logger.Error("Failed to send test map photo", slog.String("error", err.Error()))
			os.Exit(1)
		}
		logger.Info("Test map successfully sent to destination chat!", slog.String("location", loc.Description))
		return
	}

	mode := telegram.ModeDaemon
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-test-notify":
			mode = telegram.ModeTestNotify
		case "-test-forward":
			mode = telegram.ModeTestForward
		case "-test-alert":
			mode = telegram.ModeTestAlert
		}
	}

	cmdHandler := command.NewHandler(bot, cfg.GoogleMapsAPIKey, logger)

	runErr := make(chan error, 1)
	go func() {
		runErr <- ub.Run(ctx, mode)
	}()

	// NEPTUN & Bot Command Listener only matter for daemon mode.
	if mode == telegram.ModeDaemon {
		go neptunClient.Start(ctx)
		go cmdHandler.Start(ctx)
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
					forwarded, skipped, filtered := ub.Stats()
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
