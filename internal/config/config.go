package config

import (
	"fmt"
	"os"

	"github.com/EternalQ/gtnh-hub/internal/discord"
	"github.com/EternalQ/gtnh-hub/internal/game"
	"github.com/ilyakaznacheev/cleanenv"
)

const defaultConfigPath = "config.yaml"

type Config struct {
	Port    string          `yaml:"port" env:"PORT" env-default:"5665"`
	Debug   bool            `yaml:"debug" env:"DEBUG" env-default:"false"`
	Discord discord.Config  `yaml:"discord"`
	Game    game.GameConfig `yaml:"game"`
}

// Env vars always override the file, so secrets never need to live in it.
func Load() (*Config, error) {
	path, explicit := os.LookupEnv("CONFIG_PATH")
	if !explicit {
		path = defaultConfigPath
	}

	var cfg Config
	var err error

	if _, statErr := os.Stat(path); statErr == nil {
		err = cleanenv.ReadConfig(path, &cfg)
	} else if explicit {
		return nil, fmt.Errorf("config file: %w", statErr)
	} else {
		err = cleanenv.ReadEnv(&cfg)
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if cfg.Discord.BotToken == "" || cfg.Discord.WebhookID == "" {
		return nil, fmt.Errorf("check discord credentials: DISCORD_BOT_TOKEN and DISCORD_WEBHOOK_ID are required")
	}

	return &cfg, nil
}
