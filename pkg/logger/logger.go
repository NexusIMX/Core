package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

// Init initializes the global logger
func Init(level string, encoding string, outputPaths []string) error {
	// Parse log level
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create encoder
	var encoder zapcore.Encoder
	if encoding == "json" {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// Create output syncer
	var writers []zapcore.WriteSyncer
	for _, path := range outputPaths {
		if path == "stdout" {
			writers = append(writers, zapcore.AddSync(os.Stdout))
		} else if path == "stderr" {
			writers = append(writers, zapcore.AddSync(os.Stderr))
		} else {
			file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			writers = append(writers, zapcore.AddSync(file))
		}
	}

	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writers...), zapLevel)
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return nil
}

// Sync flushes any buffered log entries
func Sync() {
	if Log != nil {
		_ = Log.Sync()
	}
}
