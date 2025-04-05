package main

import (
	"os"
	"time"

	"github.com/sabouaram/data2vid/cmd"
	"github.com/sabouaram/data2vid/internal/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	logger *zap.Logger
	err    error

	cfg *viper.Viper
)

func customLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var coloredLevel string

	switch l {
	case zapcore.InfoLevel:
		coloredLevel = "\x1b[34mINFO\x1b[0m"
	case zapcore.ErrorLevel:
		coloredLevel = "\x1b[31mERROR\x1b[0m"
	default:
		coloredLevel = l.String()
	}

	enc.AppendString(coloredLevel)
}

func init() {

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    customLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout(time.RFC3339),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if os.Getenv("DEBUG") == "1" {
		encoderConfig.StacktraceKey = "stacktrace"
	}

	loggerConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      false,
		Sampling:         nil,
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	if logger, err = loggerConfig.Build(); err != nil {
		panic(err)
	}
}

func main() {

	defer logger.Sync()

	if cfg, err = config.ReadConfig(); err != nil {
		logger.Fatal("config error: ", zap.Any("error =>", err))
	}

	cmd.Execute(logger,cfg)
}
