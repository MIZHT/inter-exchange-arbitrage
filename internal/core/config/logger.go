package core_config

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

type Logger struct {
	Level  string `envconfig:"LEVEL"`
	Folder string `envconfig:"FOLDER"`
}

func loggerConfig(logger *Logger) error {
	if err := envconfig.Process("LOGGER", logger); err != nil {
		return fmt.Errorf("load logger config: %w", err)
	}

	return nil
}
