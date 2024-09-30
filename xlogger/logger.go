package xlogger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Level      string
	LogType    string
	AddSource  bool
	SourcePath string
}

func New(conf Config) *slog.Logger {
	opts := &slog.HandlerOptions{
		AddSource:   conf.AddSource,
		Level:       getLogLevel(conf.Level),
		ReplaceAttr: replaceAttr(conf),
	}

	handler := getHandler(conf.LogType, opts)

	return slog.New(handler)

}

func getLogLevel(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func getHandler(logType string, opts *slog.HandlerOptions) slog.Handler {
	switch strings.ToLower(logType) {
	case "json":
		return slog.NewJSONHandler(os.Stdout, opts)

	default:
		return slog.NewTextHandler(os.Stdout, opts)
	}
}

func replaceAttr(conf Config) func(groups []string, a slog.Attr) slog.Attr {
	return func(_ []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.SourceKey {
			if source, ok := attr.Value.Any().(*slog.Source); ok && source != nil {
				sourceFile := fmt.Sprintf("%s:%d", source.File, source.Line)

				if len(conf.SourcePath) > 0 {
					if strings.HasPrefix(source.File, conf.SourcePath) {
						sourceFile = fmt.Sprintf("%s:%d", strings.TrimPrefix(source.File, conf.SourcePath), source.Line)

					} else if index := strings.Index(source.File, conf.SourcePath); index > 0 {
						sourceFileSuffix := source.File[index+len(conf.SourcePath):]
						sourceFile = fmt.Sprintf("%s:%d", sourceFileSuffix, source.Line)
					}
				}

				return slog.String(slog.SourceKey, sourceFile)
			}
		}

		return attr
	}
}
