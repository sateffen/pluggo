package main

import (
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/sateffen/pluggo/backends"
	"github.com/sateffen/pluggo/config"
	"github.com/sateffen/pluggo/frontends"
)

func main() {
	globalLogger := slog.New(slog.NewTextHandler(os.Stdout, nil))
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

	err = backends.InitBackends(conf.Backends)
	if err != nil {
		slog.Error("could not create frontends", slog.Any("error", err))
		os.Exit(1)
	}

	err = frontends.InitFrontends(conf.Frontends)
	if err != nil {
		slog.Error("could not create frontends", slog.Any("error", err))
		os.Exit(1)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	slog.Info("started successfully")

	<-signalChan

	slog.Info("recieved exit signal, stopping...")
	frontends.CloseFrontends()
	os.Exit(0)
}
