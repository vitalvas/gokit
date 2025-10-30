# xconfig

A flexible Go configuration library that supports loading from multiple sources with clear precedence rules.

## Features

- **Multiple file formats**: YAML, JSON, YML
- **Duration parsing**: Human-readable duration strings in both JSON and YAML (`"5m"`, `"30s"`, `"2h30m"`)
- **Environment variables**: with prefix support and custom env variable names
- **Custom env tags**: override default environment variable keys (`env:"CUSTOM_VAR"`)
- **Custom separators**: customize delimiters for slices and maps (`envSeparator:":"`)
- **Multiple files**: load and merge from multiple configuration files
- **Default tags**: set default values using struct tags (`default:"value"`)
- **Custom defaults**: override struct defaults programmatically
- **Macro expansion**: `${env:VAR_NAME}` syntax for environment variable substitution
- **Data types**: strings, numbers, booleans, slices, maps, time.Duration
- **Type safety**: compile-time type checking
- **Zero dependencies**: only uses Go standard library + gopkg.in/yaml.v3

## Installation

```bash
go get github.com/vitalvas/gokit/xconfig
```

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/vitalvas/gokit/xconfig"
)

type Config struct {
    Logger LoggerConfig `yaml:"logger" json:"logger"`
    Health HealthConfig `yaml:"health" json:"health"`
}

type LoggerConfig struct {
    Level string `yaml:"level" json:"level" default:"info"`
}

type HealthConfig struct {
    Address string `yaml:"address" json:"address" default:":8080"`
    Enabled bool   `yaml:"enabled" json:"enabled" default:"true"`
}

func main() {
    var cfg Config
    
    err := xconfig.Load(&cfg,
        xconfig.WithFiles("config.yaml"),
        xconfig.WithEnv("APP"),
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Logger Level: %s\n", cfg.Logger.Level)
    fmt.Printf("Health Address: %s\n", cfg.Health.Address)
}
```

## Configuration Sources

### 1. Default Tags
Set default values directly in struct tags:
```go
type Config struct {
    Host    string `yaml:"host" default:"localhost"`
    Port    int    `yaml:"port" default:"8080"`
    Enabled bool   `yaml:"enabled" default:"true"`
    Timeout float64 `yaml:"timeout" default:"30.5"`
    Debug   bool   `yaml:"debug" default:"false"`
}
```

**Supported types**: `string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `bool`, `float32`, `float64`, `time.Duration`, and pointer types.

### 2. Struct Defaults
```go
func (c *Config) Default() {
    *c = Config{
        Logger: LoggerConfig{Level: "info"},
        Health: HealthConfig{Address: ":8080"},
    }
}
```

**Note**: `Default()` methods take precedence over `default` tags.

### 3. Custom Defaults
```go
customDefaults := Config{
    Logger: LoggerConfig{Level: "debug"},
}

err := xconfig.Load(&cfg, xconfig.WithDefault(customDefaults))
```

### 4. Configuration Files

**YAML** (`config.yaml`):
```yaml
logger:
  level: "debug"
health:
  address: ":9090"
  enabled: true
timeout: "30s"  # time.Duration supported in YAML
```

**JSON** (`config.json`):
```json
{
  "logger": {
    "level": "debug"
  },
  "health": {
    "address": ":9090",
    "enabled": true
  },
  "timeout": "30s"
}
```

### 5. Environment Variables

**With prefix:**
```bash
APP_LOGGER_LEVEL=error
APP_HEALTH_ADDRESS=:3000
APP_HEALTH_ENABLED=false
```

**Without prefix (using `WithEnv("-")`):**
```go
err := xconfig.Load(&cfg, xconfig.WithEnv("-"))
```
```bash
LOGGER_LEVEL=error
HEALTH_ADDRESS=:3000
HEALTH_ENABLED=false
```

### 5.1. Custom Environment Variable Keys

Use the `env` tag to override the default environment variable key construction:

```go
type Config struct {
    DatabaseURL  string `env:"DATABASE_URL" yaml:"database_url"`     // Uses DATABASE_URL instead of APP_DATABASE_URL
    APIKey       string `env:"API_SECRET" yaml:"api_key"`            // Uses API_SECRET instead of APP_API_KEY  
    ServicePort  int    `env:"PORT" yaml:"port"`                     // Uses PORT instead of APP_PORT
    LogLevel     string `yaml:"log_level"`                           // Uses APP_LOG_LEVEL (no env tag)
}
```

**Environment variables (with prefix "APP"):**
```bash
# Custom env tag names (prefix is ignored when env tag is present)
DATABASE_URL=postgres://localhost:5432/mydb
API_SECRET=secret123
PORT=8080

# Standard prefixed name (no env tag)
APP_LOG_LEVEL=debug
```

**Environment variables (with `WithEnv("-")` - no prefix):**
```bash
# Custom env tag names (used as-is)
DATABASE_URL=postgres://localhost:5432/mydb
API_SECRET=secret123
PORT=8080

# Standard field names (no prefix applied)
LOG_LEVEL=debug
```

**Key Benefits:**
- **Override complex naming**: Use simple names like `PORT` instead of `APP_SERVICE_HTTP_PORT`
- **Match existing env vars**: Integrate with existing environment variable conventions
- **Third-party compatibility**: Use standard names like `DATABASE_URL`, `REDIS_URL`, etc.
- **Selective customization**: Mix custom env names with standard prefixed names

**Important Notes:**
- When using `env` tags, the **prefix is NOT applied** to the environment variable name
- The `env` tag value is used exactly as specified (e.g., `env:"PORT"` → `PORT`, not `APP_PORT`)
- To ignore a field from environment loading, use `env:"-"`
- Complex tags like `env:"PORT,omitempty"` only use the first part (`PORT`) for the environment variable name

### 6. Macro Expansion

Configuration files support `${env:VAR_NAME}` macro syntax for environment variable substitution:

**YAML with macros**:
```yaml
database:
  url: "postgres://user:pass@${env:DB_HOST}:${env:DB_PORT}/mydb"
  host: "${env:DB_HOST}"
  port: "${env:DB_PORT}"

api:
  endpoint: "${env:API_PROTOCOL}://${env:API_HOST}/api/v1"
  
servers:
  - "${env:SERVER1}"
  - "${env:SERVER2}"
  - "static.example.com"
```

**Environment variables**:
```bash
DB_HOST=localhost
DB_PORT=5432
API_PROTOCOL=https
API_HOST=api.example.com
SERVER1=web1.example.com
SERVER2=web2.example.com
```

**Result after macro expansion**:
```yaml
database:
  url: "postgres://user:pass@localhost:5432/mydb"
  host: "localhost"
  port: "5432"

api:
  endpoint: "https://api.example.com/api/v1"
  
servers:
  - "web1.example.com"
  - "web2.example.com"
  - "static.example.com"
```

## Default Tag Reference

### Supported Types and Examples

```go
type Config struct {
    // String values
    Name        string `yaml:"name" default:"myapp"`
    Environment string `yaml:"env" default:"production"`
    
    // Integer types
    Port        int    `yaml:"port" default:"8080"`
    Workers     int32  `yaml:"workers" default:"4"`
    MaxConns    int64  `yaml:"max_conns" default:"1000"`
    
    // Unsigned integer types  
    BufferSize  uint   `yaml:"buffer_size" default:"1024"`
    Timeout     uint32 `yaml:"timeout" default:"30"`
    
    // Boolean values
    Enabled     bool   `yaml:"enabled" default:"true"`
    Debug       bool   `yaml:"debug" default:"false"`
    
    // Float types
    Ratio       float32 `yaml:"ratio" default:"0.75"`
    Threshold   float64 `yaml:"threshold" default:"99.5"`
    
    // Duration types
    Timeout     time.Duration `yaml:"timeout" default:"30s"`
    RetryDelay  time.Duration `yaml:"retry_delay" default:"5m"`
    MaxWait     time.Duration `yaml:"max_wait" default:"1h"`
    
    // Pointer types (automatically initialized)
    OptionalHost *string `yaml:"optional_host" default:"localhost"`
    OptionalPort *int    `yaml:"optional_port" default:"3000"`
}
```

### Default Tag Rules

1. **Zero Value Check**: Default tags are only applied to fields with zero values
2. **Type Validation**: Values are parsed and validated according to the field type
3. **Overflow Protection**: Integer values are checked for overflow
4. **Error Handling**: Invalid default values cause load errors with descriptive messages

### Nested Structs with Default Tags

```go
type ServerConfig struct {
    HTTP HTTPConfig `yaml:"http"`
    DB   DBConfig   `yaml:"database"`
}

type HTTPConfig struct {
    Host        string `yaml:"host" default:"0.0.0.0"`
    Port        int    `yaml:"port" default:"8080"`
    ReadTimeout int    `yaml:"read_timeout" default:"30"`
}

type DBConfig struct {
    Host     string `yaml:"host" default:"localhost"`
    Port     int    `yaml:"port" default:"5432"`
    Database string `yaml:"database" default:"myapp"`
    SSL      bool   `yaml:"ssl" default:"true"`
}
```

### Combining Default Tags with Default Methods

```go
type LoggerConfig struct {
    Level  string `yaml:"level" default:"info"`  // Tag default
    Format string `yaml:"format" default:"json"` // Tag default
    File   string `yaml:"file"`                  // No default tag
}

// Default method overrides tag defaults
func (c *LoggerConfig) Default() {
    c.Level = "warn"  // Overrides tag default "info" → "warn"
    c.File = "/var/log/app.log"  // Sets value for field without tag
    // c.Format remains "json" from tag since not overridden
}
```

## Advanced Features

### Multiple Files
```go
err := xconfig.Load(&cfg,
    xconfig.WithFiles("base.yaml", "override.json", "local.yml"),
    xconfig.WithEnv("APP"),
)
```

### Loading from Directories
```go
// Load all config files from a directory
err := xconfig.Load(&cfg,
    xconfig.WithDirs("/etc/myapp/config"),
    xconfig.WithEnv("APP"),
)

// Load from multiple directories
err := xconfig.Load(&cfg,
    xconfig.WithDirs("/etc/myapp/config", "/usr/local/etc/myapp"),
    xconfig.WithEnv("APP"),
)

// Combine directories and specific files
err := xconfig.Load(&cfg,
    xconfig.WithDirs("/etc/myapp/config"),        // Load all config files from directory
    xconfig.WithFiles("/etc/myapp/override.yaml"), // Load specific override file
    xconfig.WithEnv("APP"),
)
```

**Directory Loading Rules:**
- Only files with extensions `.json`, `.yaml`, `.yml` are loaded (case-insensitive)
- Files are loaded in **ascending alphabetical order** within each directory
- Subdirectories are ignored
- Non-existent directories are silently skipped
- Files from later directories override files from earlier directories

**File Loading Order Example:**
```
Directory contents: zebra.yaml, alpha.json, config.yml
Loading order:     1. alpha.json → 2. config.yml → 3. zebra.yaml
```

### Slices and Maps
```go
type Config struct {
    Hosts   []string           `yaml:"hosts"`
    Ports   []int              `yaml:"ports"`
    Labels  map[string]string  `yaml:"labels"`
}

// Environment variables:
// APP_HOSTS=web1,web2,web3
// APP_PORTS=8080,9090,3000
// APP_LABELS=env=prod,region=us-east
```

### Custom Separators for Slices and Maps

By default, environment variables for slices and maps use commas (`,`) as separators. You can customize the separator using the `envSeparator` tag:

```go
type Config struct {
    // Use colon as separator
    Hosts    []string          `yaml:"hosts" envSeparator:":"`

    // Use pipe as separator
    Tags     []string          `yaml:"tags" envSeparator:"|"`

    // Use semicolon for maps
    Labels   map[string]string `yaml:"labels" envSeparator:";"`

    // Default comma separator (no tag needed)
    Ports    []int             `yaml:"ports"`
}
```

**Environment variables:**
```bash
APP_HOSTS=web1.example.com:web2.example.com:web3.example.com
APP_TAGS=tag1|tag2|tag3
APP_LABELS=env=prod;region=us-east;tier=web
APP_PORTS=8080,9090,3000
```

**Use Cases:**
- **Colon separator (`:`)**: When values contain commas (e.g., URLs with query parameters)
- **Pipe separator (`|`)**: For better readability or when values contain commas
- **Semicolon separator (`;`)**: Common in Windows environments or when interfacing with legacy systems
- **Custom separators**: Use any single-character or multi-character separator that suits your needs

**Key Points:**
- The `envSeparator` tag only affects environment variable parsing
- File-based configuration (YAML/JSON) uses standard array/object syntax
- Whitespace around separators is automatically trimmed
- Different fields can use different separators
- Default separator remains comma (`,`) for backward compatibility

## Priority Chain

Configuration values are resolved in this order (later sources override earlier ones):

```
1. Default Tags → 2. Default() Methods → 3. Custom Defaults → 4. Directories → 5. Files → 6. Environment Variables
   (lowest priority)                                                                                (highest priority)
```

**Note**: 
- Default tags are applied first, then `Default()` methods override them
- Directories are processed before individual files
- Macro expansion happens after all files/directories are loaded but before environment variables are processed
- Environment variables always have the highest precedence

### Detailed Priority Chain

| Priority | Source | Method | Override Behavior |
|----------|--------|--------|-------------------|
| 1 (Lowest) | **Default Tags** | `default:"value"` tags | Sets initial values from struct tags |
| 2 | **Default Methods** | `Default()` methods | Overrides default tags |
| 3 | **Custom Defaults** | `WithDefault(config)` | Overrides default methods |
| 4 | **Directory Files** | `WithDirs()` | Overrides custom defaults |
| 5 | **Configuration Files** | `WithFiles()` | Overrides directory files |
| 5.5 | **Macro Expansion** | `${env:VAR}` in files | Expands macros in loaded config |
| 6 (Highest) | **Environment Variables** | `WithEnv(prefix)` | Overrides everything |

### Example Priority Resolution

```go
// 1. Default Tag
type LoggerConfig struct {
    Level string `yaml:"level" default:"info"`  // Initial value
}

// 2. Default Method (overrides tag)
func (c *LoggerConfig) Default() {
    c.Level = "debug"  // Overrides "info" → "debug"
}

// 3. Custom Default
customDefaults := Config{
    Logger: LoggerConfig{Level: "warn"},  // Overrides "debug" → "warn"
}

// 4. Configuration File (config.yaml)
logger:
  level: "error"  # Overrides "warn" → "error"

// 5. Environment Variable
APP_LOGGER_LEVEL=fatal  # Overrides "error" → "fatal" (final value)

// Result: cfg.Logger.Level = "fatal"
```

### Multiple Files Priority

When using multiple files, they are processed in order:

```go
xconfig.Load(&cfg, xconfig.WithFiles("base.yaml", "prod.json", "local.yml"))
//                                   ↑            ↑           ↑
//                              1st (lowest)  2nd (middle)  3rd (highest)
```

Each subsequent file can override values from previous files.

## Options

| Option | Description | Example |
|--------|-------------|---------|
| `WithFiles(files...)` | Load single/multiple files | `WithFiles("config.yaml")` or `WithFiles("base.yaml", "prod.json")` |
| `WithDirs(dirs...)` | Load from single/multiple directories | `WithDirs("/etc/myapp")` or `WithDirs("/etc/myapp", "/usr/local/etc/myapp")` |
| `WithEnv(prefix)` | Load environment variables with prefix | `WithEnv("APP")` |
| `WithEnv("-")` | Load environment variables without prefix | `WithEnv("-")` |
| `WithDefault(config)` | Set custom defaults | `WithDefault(myDefaults)` |

## Supported Types

### For Configuration Fields
- **Primitives**: `string`, `int`, `bool`, `float64`, etc.
- **Slices**: `[]string`, `[]int`, `[]bool`, `[]float64`
- **Maps**: `map[string]string`, `map[string]int`, etc.
- **Structs**: Nested configuration structures
- **Pointers**: `*Config`, `*string`, etc.

### For Default Tags
- **Strings**: `string`
- **Integers**: `int`, `int8`, `int16`, `int32`, `int64`
- **Unsigned Integers**: `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- **Booleans**: `bool` (accepts: `"true"`, `"false"`, `"1"`, `"0"`)
- **Floats**: `float32`, `float64`
- **Durations**: `time.Duration` (accepts formats like `"30s"`, `"5m"`, `"1h30m"`, etc.)
- **Pointers**: `*string`, `*int`, `*time.Duration`, etc. (automatically initialized)

## Environment Variable Key Construction

Environment variable keys are built using this pattern: `PREFIX_FIELD_SUBFIELD_...`

### Key Building Rules

1. **Start with prefix** (converted to uppercase)
2. **Add field names** from struct tags (yaml/json) or field names
3. **Separate with underscores** (`_`)
4. **Convert to uppercase**

### Tag Priority for Field Names

The library checks tags in this order:
1. `env` tag (highest priority - overrides environment variable key)
2. `yaml` tag (second choice)
3. `json` tag (if no yaml tag)
4. Struct field name converted from camelCase to snake_case (if no tags)

### Examples

```go
type Config struct {
    Logger    LoggerConfig      `yaml:"logger" json:"log"`
    Health    HealthConfig      `yaml:"health"`
    DB        DatabaseConfig   `env:"DATABASE_URL" json:"database"`  // Custom env tag
    CachePool CacheConfig      // no tags - uses camelCase conversion
}

type LoggerConfig struct {
    Level        string `env:"LOG_LEVEL" yaml:"level" json:"lvl"`     // Custom env tag
    File         string `yaml:"file"`
    TheLongKey   string // no tags - converts to "the_long_key"
}

type HealthConfig struct {
    Address string `env:"PORT" yaml:"address"`                       // Custom env tag
    Auth    AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
    Enabled bool   `yaml:"enabled"`
    Secret  string `env:"AUTH_SECRET" yaml:"secret"`                 // Custom env tag
}
```

### Environment Variable Keys (with prefix "APP"):

| Field Path | Tag Priority | Environment Key | Example Value |
|------------|--------------|-----------------|---------------|
| `Logger.Level` | env tag | `LOG_LEVEL` | `debug` |
| `Logger.File` | yaml tag | `APP_LOGGER_FILE` | `/var/log/app.log` |
| `Logger.TheLongKey` | camelCase | `APP_LOGGER_THE_LONG_KEY` | `my-value` |
| `Health.Address` | env tag | `PORT` | `8080` |
| `Health.Auth.Enabled` | yaml tag | `APP_HEALTH_AUTH_ENABLED` | `true` |
| `Health.Auth.Secret` | env tag | `AUTH_SECRET` | `mysecret` |
| `DB` (custom env) | env tag | `DATABASE_URL` | `postgres://...` |
| `CachePool` (no tags) | camelCase | `APP_CACHE_POOL_*` | |

### CamelCase to snake_case Conversion

When struct fields have no yaml or json tags, field names are automatically converted from camelCase to snake_case for environment variable names:

| Field Name | Converted Name | Environment Key (prefix "APP") |
|------------|----------------|--------------------------------|
| `TheLongKey` | `the_long_key` | `APP_THE_LONG_KEY` |
| `XMLParser` | `xml_parser` | `APP_XML_PARSER` |
| `HTTPClient` | `http_client` | `APP_HTTP_CLIENT` |
| `UserID` | `user_id` | `APP_USER_ID` |
| `APIKey` | `api_key` | `APP_API_KEY` |

The conversion handles acronyms intelligently, keeping consecutive uppercase letters together as a single word.

### Special Cases

#### Slices
```bash
# For []string field with yaml:"hosts"
APP_HOSTS=web1.example.com,web2.example.com,web3.example.com

# For []int field with yaml:"ports"
APP_PORTS=8080,9090,3000

# For []bool field with yaml:"enabled"
APP_ENABLED=true,false,true

# With custom separator (envSeparator:":")
APP_HOSTS=web1.example.com:web2.example.com:web3.example.com
```

#### Maps
```bash
# For map[string]string field with yaml:"labels"
APP_LABELS=env=production,region=us-east,tier=web

# For map[string]int field with yaml:"ports"
APP_PORTS=http=80,https=443,ssh=22

# For map[string]bool field with yaml:"features"
APP_FEATURES=auth=true,cache=false,debug=true

# With custom separator (envSeparator:";")
APP_LABELS=env=production;region=us-east;tier=web
```

#### Nested Structures
```bash
# Nested fields follow the same pattern
APP_DATABASE_HOST=localhost
APP_DATABASE_PORT=5432
APP_DATABASE_CONFIG_TIMEOUT=30
APP_DATABASE_CONFIG_RETRIES=3
```

### Tag Examples

```go
type ServerConfig struct {
    // Uses yaml tag "bind_address" 
    Address string `yaml:"bind_address" json:"addr"`
    // → Environment key: APP_BIND_ADDRESS

    // Uses json tag "db_port" (no yaml tag)
    Port int `json:"db_port"`
    // → Environment key: APP_DB_PORT

    // Uses struct field name (no tags)
    Timeout int
    // → Environment key: APP_TIMEOUT
}
```

## Macro Expansion Rules

### Syntax
- **Format**: `${env:VARIABLE_NAME}`
- **Case-sensitive**: Environment variable names are case-sensitive
- **No nesting**: Macros cannot reference other macros

### Behavior
- **Undefined variables**: If an environment variable is not set, the macro is left unchanged
- **Empty variables**: If an environment variable is empty, the macro is left unchanged  
- **Multiple macros**: Multiple macros can exist in the same string value
- **Non-string fields**: Macros only work in string values, slices of strings, and string map values

### Examples
```yaml
# Valid macro usage
database_url: "postgres://${env:DB_USER}:${env:DB_PASS}@${env:DB_HOST}/db"
servers: ["${env:SERVER1}", "${env:SERVER2}"]
labels:
  environment: "${env:APP_ENV}"
  version: "${env:APP_VERSION}"

# Invalid - macros don't work in non-string fields
port: ${env:DB_PORT}     # Won't expand - use environment variables instead
enabled: ${env:ENABLED}  # Won't expand - use environment variables instead
```

## Environment Variable Format

- **Primitives**: `PREFIX_FIELD=value`
- **Nested**: `PREFIX_PARENT_CHILD=value`
- **Slices**: `PREFIX_FIELD=item1,item2,item3` (default comma separator)
- **Slices with custom separator**: `PREFIX_FIELD=item1:item2:item3` (use `envSeparator:":"`)
- **Maps**: `PREFIX_FIELD=key1=value1,key2=value2` (default comma separator)
- **Maps with custom separator**: `PREFIX_FIELD=key1=value1;key2=value2` (use `envSeparator:";"`)

**Note**: The `envSeparator` tag allows customizing the delimiter for parsing slices and maps from environment variables. This is useful when values contain commas or when working with legacy systems that use different separators.

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.