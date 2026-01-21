package xlogger

import (
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
)

type Config struct {
	Level     string
	LogType   string
	AddSource bool
}

func New(conf Config) *slog.Logger {
	sourcePath := detectSourcePath()

	opts := &slog.HandlerOptions{
		AddSource:   conf.AddSource,
		Level:       getLogLevel(conf.Level),
		ReplaceAttr: newReplaceAttr(sourcePath),
	}

	handler := getHandler(conf.LogType, opts)

	return slog.New(handler)
}

// detectSourcePath extracts the module path from build info
func detectSourcePath() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Path == "" {
		return ""
	}

	return info.Main.Path
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

func newReplaceAttr(sourcePath string) func([]string, slog.Attr) slog.Attr {
	return func(_ []string, attr slog.Attr) slog.Attr {
		if attr.Key == slog.SourceKey {
			if source, ok := attr.Value.Any().(*slog.Source); ok && source != nil {
				sourceFile := fmt.Sprintf("%s:%d", source.File, source.Line)

				if len(sourcePath) > 0 {
					if index := strings.Index(source.File, sourcePath); index >= 0 {
						sourceFile = fmt.Sprintf("%s:%d", source.File[index+len(sourcePath)+1:], source.Line)
					}
				}

				return slog.String(slog.SourceKey, sourceFile)
			}
		}

		return attr
	}
}
