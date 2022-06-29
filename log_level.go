package main

type logLevel int

const (
	logLevelTrace logLevel = iota
	logLevelDebug
	logLevelInfo
	logLevelWarn
	logLevelError
)

func (ll logLevel) IsErrorOrWarn() bool {
	return ll > logLevelInfo
}

func (ll logLevel) IsInfoOrHigher() bool {
	return ll >= logLevelInfo
}

func parseLogLevel(level string) logLevel {
	switch level {
	case "trace":
		return logLevelTrace
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

func levelToColor(level logLevel) string {
	switch level {
	case logLevelTrace, logLevelDebug:
		return "\u001b[32m" // green
	case logLevelInfo:
		return "" // no color
	case logLevelWarn:
		return "\u001b[33m" // yellow
	case logLevelError:
		return "\u001b[31m" // red
	default:
		return "\u001b[36m" // cyan
	}
}
