package config

import (
	"github.com/caarlos0/env/v11"
	"github.com/stripe/stripe-go/v82"
)

type Config struct {
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
