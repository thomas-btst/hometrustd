package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"

	"github.com/fsnotify/fsnotify"
	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
)

const configDir = "hometrust"

func loadConfig() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(
		&config,
		viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(mapstructure.TextUnmarshallerHookFunc())),
	); err != nil {
		return nil, fmt.Errorf("unable to parse configuration file, %w", err)
	}

	if err := config.Validate(); err != nil {
		slog.Warn("Configuration validation failed: ", slog.Any("error", err))
	}

	return &config, nil
}

func initViper() error {
	usrCfgDir, err := os.UserConfigDir()
	if err != nil {
		slog.Warn(
			"failed to get user config directory: ",
			slog.Any("error", err),
		)
		return nil
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(filepath.Join(usrCfgDir, configDir))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := errors.AsType[viper.ConfigFileNotFoundError](err); ok {
			return nil
		} else {
			return fmt.Errorf("failed to read configuration file: %w", err)
		}
	}

	return nil
}

func Load() (*Config, error) {
	if err := initViper(); err != nil {
		return nil, err
	}

	config, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	return config, nil
}

func LoadAndWatch() (*Store, error) {
	config, err := Load()
	if err != nil {
		return nil, err
	}

	store := newStore(config)

	viper.OnConfigChange(func(e fsnotify.Event) {
		config, err = loadConfig()
		if err != nil {
			slog.Error("Failed to reload configuration", slog.Any("error", err))
			return
		}

		current := store.Current()
		if reflect.DeepEqual(current, config) {
			return
		}

		store.Update(config)
		slog.Info("Configuration reloaded")
	})

	viper.WatchConfig()

	return store, nil
}
