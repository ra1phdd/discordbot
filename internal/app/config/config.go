package config

import (
	"github.com/caarlos0/env"
	"github.com/joho/godotenv"
	"log"
)

var cfg Config

type Config struct {
	BotToken    string `env:"BOT_TOKEN,required"`
	ChannelID   uint64 `env:"CHANNEL_ID"`
	LoggerLevel string `env:"LOGGER_LEVEL" envDefault:"info"`
}

func NewConfig(files ...string) (*Config, error) {
	err := godotenv.Load(files...)
	if err != nil {
		log.Fatal("Файл .env не найден", err.Error())
	}

	err = env.Parse(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Get() *Config {
	return &cfg
}
