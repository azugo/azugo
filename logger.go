package azugo

import (
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var canColorStdout = false

func (a *App) initLogger() error {
	if a.logger != nil {
		return nil
	}

	fields := make([]zap.Field, 0, 3)
	if a.AppName != "" {
		fields = append(fields, zap.String("service.name", a.AppName))
	}
	if a.AppVer != "" {
		fields = append(fields, zap.String("service.version", a.AppVer))
	}
	fields = append(fields, zap.String("service.environment", strings.ToLower(string(a.Env()))))

	// TODO: add additional fields for logger

	if canColorStdout && a.Env().IsDevelopment() {
		conf := zap.NewDevelopmentEncoderConfig()
		conf.EncodeLevel = zapcore.CapitalColorLevelEncoder

		a.logger = zap.New(
			zapcore.NewCore(
				zapcore.NewConsoleEncoder(conf),
				zapcore.AddSync(colorable.NewColorableStdout()),
				zap.DebugLevel,
			),
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
		).With(fields...)

		return nil
	}

	encoderConfig := ecszap.NewDefaultEncoderConfig()
	core := ecszap.NewCore(encoderConfig, os.Stdout, zap.InfoLevel)

	a.logger = zap.New(core, zap.AddCaller()).With(fields...)

	return nil
}

func (a *App) Log() *zap.Logger {
	if a.logger == nil {
		if err := a.initLogger(); err != nil {
			panic(err)
		}
	}
	return a.logger
}
