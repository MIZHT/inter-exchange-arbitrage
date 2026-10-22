package core_config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Logger    Logger
	Exchanges map[ExchangeName]Exchange
	Proxies   []string
}

func NewConfig() (*Config, error) {
	cfg := &Config{
		Exchanges: make(map[ExchangeName]Exchange),
	}

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("load .env file: %w", err)
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

	var err error
	cfg.Proxies, err = readProxies()
	if err != nil {
		return nil, fmt.Errorf("get proxy list: %w", err)
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

func readProxies() ([]string, error) {
	file, err := os.OpenFile("proxies.txt", os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var proxies []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		proxies = append(proxies, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return proxies, nil
}
