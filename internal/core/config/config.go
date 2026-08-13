package core_config

import (
	"fmt"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Logger    Logger
	Exchanges map[ExchangeName]Exchange
}

func NewConfig() (*Config, error) {
	cfg := &Config{
		Exchanges: make(map[ExchangeName]Exchange),
	}

	if err := loggerConfig(&cfg.Logger); err != nil {
		return nil, err
	}

	exchangesString := strings.TrimSpace(strings.ToUpper(os.Getenv("EXCHANGES")))
	if exchangesString == "" {
		return nil, fmt.Errorf("EXCHANGES env is required")
	}
	exchanges := strings.Split(exchangesString, ",")

	for _, exchangeName := range exchanges {
		var exchange Exchange
		if err := envconfig.Process(exchangeName, &exchange); err != nil {
			return nil, fmt.Errorf("get env of %s: %w", exchangeName, err)
		}
		cfg.Exchanges[ExchangeName(strings.ToLower(exchangeName))] = exchange
	}

	return cfg, nil
}

func NewConfigMust() *Config {
	config, err := NewConfig()
	if err != nil {
		err = fmt.Errorf("get core_config: %w", err)
		panic(err)
	}
	return config
}
