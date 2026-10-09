package core_logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	core_config "github.com/MIZHT/inter-exchange-arbitrage/internal/core/config"
	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*slog.Logger

	//zapLogger *zap.Logger
	file *os.File
}

func NewLogger(config core_config.Logger) (*Logger, error) {
	zapLvl := zap.NewAtomicLevel()
	if err := zapLvl.UnmarshalText([]byte(config.Level)); err != nil {
		return nil, fmt.Errorf("unmarshal log level: %w", err)
	}

	if err := os.MkdirAll(config.Folder, 0755); err != nil {
		return nil, fmt.Errorf("mkdir log folder: %w", err)
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15-04-05.000000")
	logFilePath := filepath.Join(
		config.Folder,
		fmt.Sprintf("%s.log", timestamp),
	)

	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	zapConfig := zap.NewDevelopmentEncoderConfig()
	zapConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02T15:04-05.000000")

	zapEncoder := zapcore.NewConsoleEncoder(zapConfig)

	core := zapcore.NewTee(
		zapcore.NewCore(zapEncoder, zapcore.AddSync(os.Stdout), zapLvl),
		zapcore.NewCore(zapEncoder, zapcore.AddSync(logFile), zapLvl),
	)
	zapLogger := zap.New(core, zap.AddCaller())

	slogLogger := slog.New(
		zapslog.NewHandler(
			zapLogger.Core(),
			zapslog.WithCaller(true),
		),
	)

	return &Logger{
		Logger: slogLogger,
		//zapLogger: zapLogger,
		file: logFile,
	}, nil
}

func NewLoggerMust(config core_config.Logger) *Logger {
	logger, err := NewLogger(config)
	if err != nil {
		err = fmt.Errorf("get core_logger: %w", err)
		panic(err)
	}
	return logger
}

func (l *Logger) Close() {
	//_ = l.zapLogger.Sync()
	if err := l.file.Close(); err != nil {
		fmt.Println("failed ot close application logger:", err)
	}
}
