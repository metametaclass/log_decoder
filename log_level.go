package main

type logLevel int

const (
	logLevelDebug logLevel = iota
	logLevelInfo
	logLevelWarn
	logLevelError
)

func (ll logLevel) IsErrorOrWarn() bool {
	return ll > logLevelInfo
}

func parseLogLevel(level string) logLevel {
	switch level {
	case "debug":
		return logLevelDebug
	case "info":
		return logLevelInfo
	case "warn":
		return logLevelWarn
	case "error":
		return logLevelError
	default:
		// return warn for unknown levels
		return logLevelWarn
	}
}
