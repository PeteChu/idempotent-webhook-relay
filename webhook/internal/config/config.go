package config

import (
	"fmt"

	"github.com/caarlos0/env/v11"
	"github.com/stripe/stripe-go/v82"
)

type Config struct {
	DBHost     string `env:"DB_HOST" envDefault:"localhost"`
	DBPort     string `env:"DB_PORT" envDefault:"5432"`
	DBName     string `env:"DB_NAME" envDefault:"idempotent-webhook-relay"`
	DBUsername string `env:"DB_USERNAME" envDefault:"postgres"`
	DBPassword string `env:"DB_PASSWORD" envDefault:"postgres"`
	DBOptions  string `env:"DB_OPTIONS" envDefault:"sslmode=disable"`

	StripeSecretKey     string `env:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret string `env:"STRIPE_WEBHOOK_SECRET"`
}

func LoadConfig() (*Config, error) {
	var config Config
	if err := env.Parse(&config); err != nil {
		return nil, err
	}

	stripe.Key = config.StripeSecretKey

	return &config, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?%s", c.DBUsername, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBOptions)
}
