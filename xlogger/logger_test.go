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
			name: "Create logger with error level",
			conf: Config{
				Level:     "error",
				LogType:   "json",
				AddSource: true,
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

func TestDetectSourcePath(t *testing.T) {
	path := detectSourcePath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "gokit")
}

func TestNewReplaceAttr(t *testing.T) {
	t.Run("with source path", func(t *testing.T) {
		replaceFunc := newReplaceAttr("github.com/vitalvas/gokit")

		attr := slog.Attr{
			Key: slog.SourceKey,
			Value: slog.AnyValue(&slog.Source{
				File: "/home/user/go/pkg/mod/github.com/vitalvas/gokit/xlogger/logger.go",
				Line: 42,
			}),
		}

		result := replaceFunc(nil, attr)
		assert.Equal(t, slog.SourceKey, result.Key)
		assert.Equal(t, "xlogger/logger.go:42", result.Value.String())
	})

	t.Run("without source path", func(t *testing.T) {
		replaceFunc := newReplaceAttr("")

		attr := slog.Attr{
			Key: slog.SourceKey,
			Value: slog.AnyValue(&slog.Source{
				File: "/full/path/to/file.go",
				Line: 10,
			}),
		}

		result := replaceFunc(nil, attr)
		assert.Equal(t, slog.SourceKey, result.Key)
		assert.Equal(t, "/full/path/to/file.go:10", result.Value.String())
	})

	t.Run("non-source attribute unchanged", func(t *testing.T) {
		replaceFunc := newReplaceAttr("github.com/vitalvas/gokit")

		attr := slog.Attr{
			Key:   "message",
			Value: slog.StringValue("test"),
		}

		result := replaceFunc(nil, attr)
		assert.Equal(t, attr, result)
	})

	t.Run("source path not in file path", func(t *testing.T) {
		replaceFunc := newReplaceAttr("github.com/other/module")

		attr := slog.Attr{
			Key: slog.SourceKey,
			Value: slog.AnyValue(&slog.Source{
				File: "/home/user/go/src/myproject/main.go",
				Line: 5,
			}),
		}

		result := replaceFunc(nil, attr)
		assert.Equal(t, slog.SourceKey, result.Key)
		assert.Equal(t, "/home/user/go/src/myproject/main.go:5", result.Value.String())
	})
}
