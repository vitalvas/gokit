# xflags

A command-line flag parser that fills a struct from tags or is built programmatically. It supports nested subcommands, option groups, namespaces, environment-variable fallback, default values, and per-option validation callbacks.

## Features

- **Short and long names** -- `-v` and `--verbose`
- **Bool and valued options** -- flags with and without arguments
- **Option groups** -- multiple named groups, each with its own set of options
- **Namespaces** -- nested option groups with dotted prefixes (`--server.port`)
- **Subcommands** -- nested command tree (`app server start`)
- **All primitive Go types** -- `bool`, `string`, all sized `int`/`uint`, `float32`/`float64`, plus `time.Duration`
- **Slices and maps** -- repeated values and `key=value` pairs
- **Repeated options** -- explicit per-flag policy via `choice` (`append`, `count`, `last`)
- **Default values** -- via the `default` tag or builder
- **Environment fallback** -- via the `env` tag
- **Validation callbacks** -- one `func(value) error` per option, covering both reaction and validation
- **Required options** -- `required:"true"`
- **Struct parsing** -- fill a struct directly from tags
- **Builder API** -- register flags and commands programmatically
- **Built-in help and version** -- auto-generated `--help`/`-h` and `--version`
- **Zero dependencies** beyond the Go standard library

## Quick Start

```go
package main

import (
	"fmt"
	"os"

	"github.com/vitalvas/gokit/xflags"
)

type Options struct {
	Verbose int    `short:"v" long:"verbose" choice:"count" description:"increase verbosity"`
	Name    string `long:"name" env:"NAME" default:"app" description:"service name"`
}

func main() {
	var opts Options

	p := xflags.New("app")
	p.SetVersion("1.2.3")
	p.AddGroup("Application Options", &opts)

	if err := p.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("name=%s verbose=%d\n", opts.Name, opts.Verbose)
}
```

## Declaring Options

Options can be declared with struct tags, the builder API, or both.

### Struct tags

Every field maps to one option. Recognized tags:

| Tag | Meaning |
|---|---|
| `short` | single-character short name, e.g. `short:"v"` -> `-v` |
| `long` | long name, e.g. `long:"verbose"` -> `--verbose` |
| `description` | help text |
| `default` | default value used when no source provides one |
| `env` | environment variable name to fall back to |
| `choice` | repeated-option policy: `append`, `count`, or `last` |
| `required` | `required:"true"` fails parsing if the option is never set |
| `namespace` | on a nested struct field, prefixes its options |
| `value-name` | placeholder shown in help, e.g. `value-name:"FILE"` |
| `hidden` | `hidden:"true"` omits the option from the help message |
| `base` | integer radix, e.g. `base:"16"`; `base:"0"` auto-detects `0x`/`0o`/`0b` |
| `optional` | `optional:"true"` allows the option to appear without a value |
| `optional-value` | value used when an optional option is given without one |

Note: the `choice` tag here controls the repeated-option policy, not a set of allowed values.

```go
type Options struct {
	Verbose int               `short:"v" long:"verbose" choice:"count"`
	Name    string            `long:"name" env:"NAME" default:"app" required:"true"`
	Tags    []string          `long:"tag" choice:"append"`
	Labels  map[string]string `long:"label"`
}
```

### Builder API

```go
p := xflags.New("app")

var verbose int
var name string

p.CountVar(&verbose, "verbose", "v", "increase verbosity")
p.StringVar(&name, "name", "", "app", "service name")
```

The builder is useful for dynamically registered flags. Struct tags and the builder can be mixed on the same parser.

## Supported Types

All primitive Go types are supported, plus `time.Duration`:

```go
type Options struct {
	Flag     bool          `long:"flag"`
	Text     string        `long:"text"`
	Count    int           `long:"count"`
	Big      int64         `long:"big"`
	Unsigned uint          `long:"unsigned"`
	Ratio    float64       `long:"ratio"`
	Timeout  time.Duration `long:"timeout"`
}
```

`time.Duration` values are parsed with `time.ParseDuration` (`--timeout=30s`).

## Bool vs Valued Options

A `bool` field is a switch and takes no argument:

```go
type Options struct {
	Debug bool `long:"debug"`
}
// --debug        -> Debug = true
// --debug=false  -> Debug = false
```

All other types require an argument:

```
--name=value
--name value
```

## Repeated Options

When the same option appears more than once, the policy is set explicitly per flag with the `choice` tag. The field type must be compatible with the chosen policy.

| `choice` | Field type | Behavior |
|---|---|---|
| `append` | slice | every occurrence is appended |
| `count` | integer | each occurrence increments the value |
| `last` | scalar | the last occurrence wins |

```go
type Options struct {
	Tags    []string `long:"tag" choice:"append"`  // --tag=a --tag=b -> [a b]
	Verbose int      `short:"v" choice:"count"`     // -vvv           -> 3
	Name    string   `long:"name" choice:"last"`    // --name=a --name=b -> b
}
```

Without a `choice` tag, a scalar defaults to last-wins and a repeated slice/map appends/merges.

## Maps

Map options collect `key=value` pairs, one per occurrence:

```go
type Options struct {
	Labels map[string]string `long:"label"`
}
// --label env=prod --label tier=web -> {env: prod, tier: web}
```

Map keys must be strings; values may be any supported primitive type.

## Default Values

Defaults are applied when no command-line argument or environment variable provides a value.

```go
type Options struct {
	Port int `long:"port" default:"8080"`
}
```

## Environment Fallback

An option with an `env` tag falls back to the named environment variable when no command-line argument is given.

```go
type Options struct {
	Token string `long:"token" env:"API_TOKEN"`
}
```

### Precedence

For each option the effective value is resolved in this order:

```
command-line argument > environment variable > default
```

## Validation and Callbacks

Each option may carry a single `func(value) error`. The parser calls it once with the resolved value (after precedence is applied). If it returns an error, parsing fails with that error. This one function covers both use cases:

- **Reaction** -- run code when the option is seen.
- **Validation** -- reject an unacceptable value.

### Via a struct func-field

A func-typed field is itself an option, linked by `long`:

```go
type Options struct {
	OnConfig func(string) error `long:"config" description:"load config file"`
}
```

To both store a value and validate it, pair a value field with a func field that shares the same `long` name:

```go
type Options struct {
	Port   int             `long:"port" default:"8080"`
	PortFn func(int) error `long:"port"`
}
```

### Via the builder

```go
p.Validate("port", func(v any) error {
	port := v.(int)
	if port < 1 || port > 65535 {
		return fmt.Errorf("port out of range: %d", port)
	}
	return nil
})
```

## Required Options

`required:"true"` makes parsing fail if the option is never set by any source (command line, environment, or default). It is independent of the value callback.

```go
type Options struct {
	Name string `long:"name" required:"true"`
}
```

## Optional Values

An option marked `optional` may appear on the command line without a value. When it does, its `optional-value` is used (or the zero value if none is given). Setting `optional-value` implies `optional`.

```go
type Options struct {
	Log string `long:"log" optional-value:"info"`
}
// --log         -> "info"
// --log=debug   -> "debug"
// --log trace   -> "trace"  (next token consumed when it is not a flag)
```

A following flag is never consumed as the value: `--log --debug` leaves `Log` at `info`. Negative numbers are treated as values, so `--offset -5` sets `-5`.

## Integer Base

Integer options parse in base 10 by default. The `base` tag selects a radix; `base:"0"` auto-detects `0x` (hex), `0o` (octal), and `0b` (binary) prefixes.

```go
type Options struct {
	Mask  int `long:"mask" base:"16"` // --mask=ff  -> 255
	Value int `long:"value" base:"0"` // --value=0x10 -> 16
}
```

## Hidden Options and Value Names

`hidden:"true"` keeps an option fully functional but omits it from the help message. `value-name` sets the placeholder shown after a valued option in help.

```go
type Options struct {
	Secret string `long:"secret" hidden:"true"`
	File   string `long:"file" value-name:"PATH"`
}
// help shows: --file PATH
// help omits: --secret
```

The builder API exposes `SetHidden`, `SetValueName`, `SetBase`, and `SetOptionalValue` by long name.

## Option Groups

Groups organize options for the help message. Each group has a title and its own struct:

```go
type AppOptions struct {
	Verbose int `short:"v" long:"verbose" choice:"count"`
}

type LogOptions struct {
	Level string `long:"level" default:"info"`
}

p := xflags.New("app")
p.AddGroup("Application Options", &appOpts)
p.AddGroup("Logging Options", &logOpts)
```

## Namespaces

A nested struct field with a `namespace` tag prefixes all of its options with a dotted path. Namespaces can nest.

```go
type Options struct {
	Server struct {
		Port int    `long:"port" default:"8080"`
		Host string `long:"host" default:"0.0.0.0"`
	} `namespace:"server"`
}
// --server.port=9090 --server.host=127.0.0.1
```

## Subcommands

Subcommands form a nested tree. Each command has its own options and may have child commands. Declare them with the builder or a struct tag.

### Via the builder

```go
type ServerOptions struct {
	Port int `long:"port" default:"8080"`
}

p := xflags.New("app")
server := p.AddCommand("server", &serverOpts)
server.SetDescription("run the server")
```

### Via a struct tag

A field tagged `command` declares a subcommand whose options come from the field's struct:

```go
type Root struct {
	Server ServerOptions `command:"server" description:"run the server"`
	Client ClientOptions `command:"client" description:"run the client"`
}
```

Invocation:

```
app server --port 9090
app client connect --addr 127.0.0.1
```

## Help

`--help` and `-h` are added automatically to the root command and every subcommand. When present, the parser prints a formatted help message and stops. Help is handled internally; the caller does not need to check for it.

```
Usage:
  app [options] <command>

Application Options:
  -v, --verbose        increase verbosity
      --name           service name (default: app)

Server Options (--server.*):
      --server.port    (default: 8080)

Commands:
  server    run the server
  client    run the client

  -h, --help            show this help message
      --version         show version information
```

## Version

`--version` is added automatically when a version string is set with `SetVersion`. If no version is set, the flag is not created. When present, it prints the application name and version and stops.

`--version` receives the `-v` shorthand only when `-v` is not already claimed by another option.

```go
p := xflags.New("app")
p.SetVersion("1.2.3")
// app --version -> "app version 1.2.3"
```

## Error Handling

`Parse` returns errors; it never calls `os.Exit`. The caller decides how to react. Built-in `--help` and `--version` are handled internally and do not surface as errors.

```go
if err := p.Parse(os.Args[1:]); err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
```
