package main

import (
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sateffen/pluggo/backends"
	"github.com/sateffen/pluggo/config"
	"github.com/sateffen/pluggo/frontends"
)

func getLogLevel() slog.Level {
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	globalLogger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: getLogLevel(),
	}))
	slog.SetDefault(globalLogger)

	if len(os.Args) == 1 {
		slog.Error("No arguments given, please pass a path to a config file")
		os.Exit(1)
	}

	configFilePath, err := filepath.Abs(os.Args[1])
	if err != nil {
		slog.Error("could not normalize config file path", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Loading config", slog.String("configFilePath", configFilePath))

	conf, err := config.LoadConfig(configFilePath)
	if err != nil {
		slog.Error("could not load config", slog.Any("error", err))
		os.Exit(1)
	}

	backendList, err := backends.NewBackendList(conf.Backends)
	if err != nil {
		slog.Error("could not create backends", slog.Any("error", err))
		os.Exit(1)
	}

	frontendList, err := frontends.NewFrontendList(conf.Frontends, backendList)
	if err != nil {
		slog.Error("could not create frontends", slog.Any("error", err))
		os.Exit(1)
	}

	frontendList.ListenAll()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("started successfully")

	<-signalChan

	slog.Info("received exit signal, stopping...")
	frontendList.CloseAll()
	os.Exit(0)
}
