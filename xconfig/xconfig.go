package xconfig

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
)

const (
	// EnvSkipPrefix is a special prefix value that indicates environment variables
	// should be loaded without any prefix. Use WithEnv(EnvSkipPrefix) to load
	// environment variables without a prefix.
	EnvSkipPrefix = "-"
)

type Options struct {
	dotenvFiles   []string
	files         []string
	dirs          []string
	envPrefix     string
	customDefault interface{}
	envMacroRegex *regexp.Regexp
	strict        bool
}

type Option func(*Options)

func WithDotenv(filenames ...string) Option {
	return func(o *Options) {
		if len(filenames) == 0 {
			filenames = []string{".env"}
		}
		o.dotenvFiles = append(o.dotenvFiles, filenames...)
	}
}

func WithFiles(filenames ...string) Option {
	return func(o *Options) {
		o.files = append(o.files, filenames...)
	}
}

func WithDirs(dirnames ...string) Option {
	return func(o *Options) {
		o.dirs = append(o.dirs, dirnames...)
	}
}

func WithEnv(prefix string) Option {
	return func(o *Options) {
		o.envPrefix = prefix
	}
}

func WithDefault(defaultConfig interface{}) Option {
	return func(o *Options) {
		o.customDefault = defaultConfig
	}
}

func WithEnvMacroRegex(pattern ...string) Option {
	return func(o *Options) {
		p := `\$\{env:([^}]+)\}`
		if len(pattern) > 0 && pattern[0] != "" {
			p = pattern[0]
		}
		o.envMacroRegex = regexp.MustCompile(p)
	}
}

func WithStrict(strict bool) Option {
	return func(o *Options) {
		o.strict = strict
	}
}

func Load(config interface{}, options ...Option) error {
	opts := &Options{}
	for _, option := range options {
		option(opts)
	}

	// Apply default tags first
	configElem, err := validateConfigPointer(config)
	if err != nil {
		return fmt.Errorf("failed to validate config: %w", err)
	}
	if err := applyDefaultTagsRecursive(configElem); err != nil {
		return fmt.Errorf("failed to apply default tags: %w", err)
	}

	// Then call Default() methods (only if no custom defaults provided)
	if opts.customDefault == nil {
		if err := callDefaultMethodsRecursive(configElem); err != nil {
			return fmt.Errorf("failed to call default methods: %w", err)
		}
	}

	// Then apply custom defaults (which completely override Default() methods)
	if opts.customDefault != nil {
		if err := applyCustomDefaults(config, opts.customDefault); err != nil {
			return fmt.Errorf("failed to apply custom defaults: %w", err)
		}
	}

	// Load dotenv files first (lowest priority for configuration)
	if len(opts.dotenvFiles) > 0 {
		if err := loadDotenvFiles(opts.dotenvFiles); err != nil {
			return fmt.Errorf("failed to load dotenv files: %w", err)
		}
	}

	// Load from directories first, then files
	if len(opts.dirs) > 0 {
		if err := loadFromDirs(config, opts.dirs, opts.strict); err != nil {
			return fmt.Errorf("failed to load from directories: %w", err)
		}
	}

	if len(opts.files) > 0 {
		if err := loadFromFiles(config, opts.files, opts.strict); err != nil {
			return fmt.Errorf("failed to load from files: %w", err)
		}
	}

	// Expand macros in loaded configuration (if any files were loaded)
	if len(opts.dirs) > 0 || len(opts.files) > 0 {
		re := opts.envMacroRegex
		if re == nil {
			re = regexp.MustCompile(`\$\{env:([^}]+)\}`)
		}
		configValue := reflect.ValueOf(config).Elem()
		expandMacrosInValue(configValue, re)
	}

	if opts.envPrefix != "" {
		if err := loadFromEnv(config, opts.envPrefix); err != nil {
			return fmt.Errorf("failed to load from environment: %w", err)
		}
	}

	return nil
}

func expandMacros(value string, re *regexp.Regexp) string {
	return re.ReplaceAllStringFunc(value, func(match string) string {
		// Extract the environment variable name from the macro pattern
		envVar := re.FindStringSubmatch(match)[1]
		if envValue := os.Getenv(envVar); envValue != "" {
			return envValue
		}
		// Return original if env var is not set or empty
		return match
	})
}

func expandMacrosInValue(v reflect.Value, re *regexp.Regexp) {
	if !v.CanSet() {
		return
	}

	switch v.Kind() {
	case reflect.String:
		if v.String() != "" {
			v.SetString(expandMacros(v.String(), re))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if field := v.Field(i); field.CanSet() {
				expandMacrosInValue(field, re)
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if elem := v.Index(i); elem.CanSet() {
				expandMacrosInValue(elem, re)
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			mapValue := v.MapIndex(key)
			if mapValue.Kind() == reflect.String && mapValue.String() != "" {
				v.SetMapIndex(key, reflect.ValueOf(expandMacros(mapValue.String(), re)))
			} else if mapValue.CanInterface() {
				newValue := reflect.New(mapValue.Type()).Elem()
				newValue.Set(mapValue)
				expandMacrosInValue(newValue, re)
				v.SetMapIndex(key, newValue)
			}
		}
	case reflect.Ptr:
		if !v.IsNil() {
			expandMacrosInValue(v.Elem(), re)
		}
	case reflect.Interface:
		if !v.IsNil() && v.Elem().CanSet() {
			expandMacrosInValue(v.Elem(), re)
		}
	}
}
