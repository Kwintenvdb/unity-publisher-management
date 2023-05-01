package logger

import "go.uber.org/zap"

type Logger interface {
	Debug(args ...interface{})
	Info(args ...interface{})
	Warn(args ...interface{})
	Error(args ...interface{})
	Fatal(args ...interface{})

	Debugw(msg string, keysAndValues ...interface{})
	Infow(msg string, keysAndValues ...interface{})
	Warnw(msg string, keysAndValues ...interface{})
	Errorw(msg string, keysAndValues ...interface{})
	Fatalw(msg string, keysAndValues ...interface{})

	Sync() error
}

// Wrap the zap logger into our own interface for easier drop-in replacement and testing.
func NewLogger() Logger {
	zapLogger, _ := zap.NewDevelopment()
	return zapLogger.Sugar()
}
