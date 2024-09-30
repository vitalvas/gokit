package xlogger

import (
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		conf     Config
		expected slog.Handler
	}{
		{
			name: "Create logger with JSON handler and debug level",
			conf: Config{
				Level:     "debug",
				LogType:   "json",
				AddSource: true,
			},
			expected: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelDebug,
			}),
		},
		{
			name: "Create logger with text handler and info level",
			conf: Config{
				Level:     "info",
				LogType:   "text",
				AddSource: false,
			},
			expected: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: false,
				Level:     slog.LevelInfo,
			}),
		},
		{
			name: "Create logger with default handler for unknown log type",
			conf: Config{
				Level:     "warn",
				LogType:   "unknown",
				AddSource: true,
			},
			expected: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelWarn,
			}),
		},
		{
			name: "Create logger with error level and source path replacement",
			conf: Config{
				Level:      "error",
				LogType:    "json",
				AddSource:  true,
				SourcePath: "/Users/vitalvas/workspace/go/src/github.com/vitalvas/gokit/",
			},
			expected: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelError,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.conf)
			assert.NotNil(t, logger)

			handler := logger.Handler()
			assert.NotNil(t, handler)
			assert.IsType(t, tt.expected, handler)
		})
	}
}

func TestGetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		logLevel string
		expected slog.Level
	}{
		{
			name:     "Debug level",
			logLevel: "debug",
			expected: slog.LevelDebug,
		},
		{
			name:     "Info level",
			logLevel: "info",
			expected: slog.LevelInfo,
		},
		{
			name:     "Warn level",
			logLevel: "warn",
			expected: slog.LevelWarn,
		},
		{
			name:     "Error level",
			logLevel: "error",
			expected: slog.LevelError,
		},
		{
			name:     "Default level for unknown input",
			logLevel: "unknown",
			expected: slog.LevelInfo,
		},
		{
			name:     "Case insensitive level",
			logLevel: "DEBUG",
			expected: slog.LevelDebug,
		},
		{
			name:     "Empty level string",
			logLevel: "",
			expected: slog.LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLogLevel(tt.logLevel)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetHandler(t *testing.T) {
	tests := []struct {
		name     string
		logType  string
		opts     *slog.HandlerOptions
		expected slog.Handler
	}{
		{
			name:    "Get JSON handler",
			logType: "json",
			opts: &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			},
			expected: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			}),
		},
		{
			name:    "Get text handler",
			logType: "text",
			opts: &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			},
			expected: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			}),
		},
		{
			name:    "Get default handler for unknown type",
			logType: "unknown",
			opts: &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			},
			expected: slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				AddSource: true,
				Level:     slog.LevelInfo,
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := getHandler(tt.logType, tt.opts)
			assert.NotNil(t, handler)
			assert.IsType(t, tt.expected, handler)
		})
	}
}

func TestReplaceAttr(t *testing.T) {
	tests := []struct {
		name     string
		conf     Config
		attr     slog.Attr
		expected slog.Attr
	}{
		{
			name: "Replace source attribute with source path",
			conf: Config{
				SourcePath: "github.com/vitalvas/gokit/",
			},
			attr: slog.Attr{
				Key: slog.SourceKey,
				Value: slog.AnyValue(&slog.Source{
					File: "/Users/vitalvas/workspace/go/src/github.com/vitalvas/gokit/xlogger/logger.go",
					Line: 42,
				}),
			},
			expected: slog.String("source", "xlogger/logger.go:42"),
		},
		{
			name: "Replace source attribute with full source path",
			conf: Config{
				SourcePath: "/Users/vitalvas/workspace/go/src/github.com/vitalvas/gokit/",
			},
			attr: slog.Attr{
				Key: slog.SourceKey,
				Value: slog.AnyValue(&slog.Source{
					File: "/Users/vitalvas/workspace/go/src/github.com/vitalvas/gokit/xlogger/logger.go",
					Line: 42,
				}),
			},
			expected: slog.String("source", "xlogger/logger.go:42"),
		},
		{
			name: "Replace source attribute without source path",
			conf: Config{},
			attr: slog.Attr{
				Key: slog.SourceKey,
				Value: slog.AnyValue(&slog.Source{
					File: "/Users/vitalvas/workspace/go/src/github.com/vitalvas/gokit/xlogger/logger.go",
					Line: 42,
				}),
			},
			expected: slog.String("source", "/Users/vitalvas/workspace/go/src/github.com/vitalvas/gokit/xlogger/logger.go:42"),
		},
		{
			name: "Non-source attribute remains unchanged",
			conf: Config{},
			attr: slog.Attr{
				Key:   "non-source",
				Value: slog.StringValue("test"),
			},
			expected: slog.Attr{
				Key:   "non-source",
				Value: slog.StringValue("test"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			replaceFunc := replaceAttr(tt.conf)
			assert.NotNil(t, replaceFunc)

			result := replaceFunc(nil, tt.attr)
			assert.Equal(t, tt.expected, result)
		})
	}
}
