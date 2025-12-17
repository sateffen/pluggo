package config

import (
	"log/slog"

	"github.com/BurntSushi/toml"
)

type TCPFrontendConfig struct {
	Name       string `toml:"name"`
	ListenAddr string `toml:"listenAddr"`
	Target     string `toml:"target"`
}

type FrontendConfigs struct {
	TCP []TCPFrontendConfig `toml:"tcp"`
}

type EchoBackendConfig struct {
	Name string `toml:"name"`
}

type TCPForwarderBackendConfig struct {
	Name       string `toml:"name"`
	TargetAddr string `toml:"targetAddr"`
}

type WoLForwarderBackendConfig struct {
	Name             string `toml:"name"`
	TargetAddr       string `toml:"targetAddr"`
	WoLMACAddr       string `toml:"wolMACAddr"`
	WoLBroadcastAddr string `toml:"wolBroadcastAddr"`
}

type BackendConfigs struct {
	Echo         []EchoBackendConfig         `toml:"echo"`
	TCPForwarder []TCPForwarderBackendConfig `toml:"tcpForwarder"`
	WoLForwarder []WoLForwarderBackendConfig `toml:"wolForwarder"`
}

type Config struct {
	Frontends FrontendConfigs `toml:"frontends"`
	Backends  BackendConfigs  `toml:"backends"`
}

// LoadConfig loads the file from given path and parses it as toml file, decoding it
// to an usable Config.
func LoadConfig(filePath string) (*Config, error) {
	var conf Config

	metaData, err := toml.DecodeFile(filePath, &conf)
	if err != nil {
		return nil, err
	}

	undecodedKeys := metaData.Undecoded()
	if len(undecodedKeys) > 0 {
		slog.Warn(
			"found unknown keys in config",
			slog.String("configFilePath", filePath),
			slog.Any("undecodedKeys", undecodedKeys),
		)
	}

	return &conf, nil
}
