# xconfig

A flexible Go configuration library that supports loading from multiple sources with clear precedence rules.

## Features

- **Multiple file formats**: YAML, JSON, YML
- **Environment variables**: with prefix support
- **Multiple files**: load and merge from multiple configuration files
- **Custom defaults**: override struct defaults programmatically
- **Macro expansion**: `${env:VAR_NAME}` syntax for environment variable substitution
- **Data types**: strings, numbers, booleans, slices, maps
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
    Level string `yaml:"level" json:"level"`
}

func (c *LoggerConfig) Default() {
    *c = LoggerConfig{Level: "info"}
}

type HealthConfig struct {
    Address string `yaml:"address" json:"address"`
    Enabled bool   `yaml:"enabled" json:"enabled"`
}

func (c *HealthConfig) Default() {
    *c = HealthConfig{
        Address: ":8080",
        Enabled: true,
    }
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

### 1. Struct Defaults
```go
func (c *Config) Default() {
    *c = Config{
        Logger: LoggerConfig{Level: "info"},
        Health: HealthConfig{Address: ":8080"},
    }
}
```

### 2. Custom Defaults
```go
customDefaults := Config{
    Logger: LoggerConfig{Level: "debug"},
}

err := xconfig.Load(&cfg, xconfig.WithDefault(customDefaults))
```

### 3. Configuration Files

**YAML** (`config.yaml`):
```yaml
logger:
  level: "debug"
health:
  address: ":9090"
  enabled: true
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
  }
}
```

### 4. Environment Variables
```bash
APP_LOGGER_LEVEL=error
APP_HEALTH_ADDRESS=:3000
APP_HEALTH_ENABLED=false
```

### 5. Macro Expansion

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

## Priority Chain

Configuration values are resolved in this order (later sources override earlier ones):

```
1. Struct Defaults → 2. Custom Defaults → 3. Directories → 4. Files → 5. Environment Variables
   (lowest priority)                                                         (highest priority)
```

**Note**: 
- Directories are processed before individual files
- Macro expansion happens after all files/directories are loaded but before environment variables are processed
- Environment variables always have the highest precedence

### Detailed Priority Chain

| Priority | Source | Method | Override Behavior |
|----------|--------|--------|-------------------|
| 1 (Lowest) | **Struct Defaults** | `Default()` methods | Sets initial values |
| 2 | **Custom Defaults** | `WithDefault(config)` | Overrides struct defaults |
| 3 | **Directory Files** | `WithDirs()` | Overrides custom defaults |
| 4 | **Configuration Files** | `WithFiles()` | Overrides directory files |
| 4.5 | **Macro Expansion** | `${env:VAR}` in files | Expands macros in loaded config |
| 5 (Highest) | **Environment Variables** | `WithEnv(prefix)` | Overrides everything |

### Example Priority Resolution

```go
// 1. Struct Default
func (c *LoggerConfig) Default() {
    c.Level = "info"  // Initial value
}

// 2. Custom Default
customDefaults := Config{
    Logger: LoggerConfig{Level: "debug"},  // Overrides "info" → "debug"
}

// 3. Configuration File (config.yaml)
logger:
  level: "warn"  # Overrides "debug" → "warn"

// 4. Environment Variable
APP_LOGGER_LEVEL=error  # Overrides "warn" → "error" (final value)

// Result: cfg.Logger.Level = "error"
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
| `WithEnv(prefix)` | Load environment variables | `WithEnv("APP")` |
| `WithDefault(config)` | Set custom defaults | `WithDefault(myDefaults)` |

## Supported Types

- **Primitives**: `string`, `int`, `bool`, `float64`, etc.
- **Slices**: `[]string`, `[]int`, `[]bool`, `[]float64`
- **Maps**: `map[string]string`, `map[string]int`, etc.
- **Structs**: Nested configuration structures
- **Pointers**: `*Config`, `*string`, etc.

## Environment Variable Key Construction

Environment variable keys are built using this pattern: `PREFIX_FIELD_SUBFIELD_...`

### Key Building Rules

1. **Start with prefix** (converted to uppercase)
2. **Add field names** from struct tags (yaml/json) or field names
3. **Separate with underscores** (`_`)
4. **Convert to uppercase**

### Tag Priority for Field Names

The library checks tags in this order:
1. `yaml` tag (first choice)
2. `json` tag (if no yaml tag)
3. Struct field name converted from camelCase to snake_case (if no tags)

### Examples

```go
type Config struct {
    Logger    LoggerConfig      `yaml:"logger" json:"log"`
    Health    HealthConfig      `yaml:"health"`
    DB        DatabaseConfig   `json:"database"`
    CachePool CacheConfig      // no tags - uses camelCase conversion
}

type LoggerConfig struct {
    Level        string `yaml:"level" json:"lvl"`
    File         string `yaml:"file"`
    TheLongKey   string // no tags - converts to "the_long_key"
}

type HealthConfig struct {
    Address string `yaml:"address"`
    Auth    AuthConfig `yaml:"auth"`
}

type AuthConfig struct {
    Enabled bool   `yaml:"enabled"`
    Secret  string `yaml:"secret"`
}
```

### Environment Variable Keys (with prefix "APP"):

| Field Path | yaml tag used | Environment Key | Example Value |
|------------|---------------|-----------------|---------------|
| `Logger.Level` | ✓ | `APP_LOGGER_LEVEL` | `debug` |
| `Logger.File` | ✓ | `APP_LOGGER_FILE` | `/var/log/app.log` |
| `Logger.TheLongKey` | ✗ (camelCase) | `APP_LOGGER_THE_LONG_KEY` | `my-value` |
| `Health.Address` | ✓ | `APP_HEALTH_ADDRESS` | `:8080` |
| `Health.Auth.Enabled` | ✓ | `APP_HEALTH_AUTH_ENABLED` | `true` |
| `Health.Auth.Secret` | ✓ | `APP_HEALTH_AUTH_SECRET` | `mysecret` |
| `DB` (json tag) | ✗ (uses json) | `APP_DATABASE_*` | |
| `CachePool` (no tags) | ✗ (camelCase) | `APP_CACHE_POOL_*` | |

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
```

#### Maps
```bash
# For map[string]string field with yaml:"labels"
APP_LABELS=env=production,region=us-east,tier=web

# For map[string]int field with yaml:"ports"
APP_PORTS=http=80,https=443,ssh=22

# For map[string]bool field with yaml:"features"
APP_FEATURES=auth=true,cache=false,debug=true
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
- **Slices**: `PREFIX_FIELD=item1,item2,item3`
- **Maps**: `PREFIX_FIELD=key1=value1,key2=value2`

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.