# xflags

A command-line flag parser that fills a struct from tags or is built programmatically. It supports nested subcommands, option groups, namespaces, environment-variable fallback, default values, and per-option validation callbacks.

## Features

- **Short and long names** -- `-v` and `--verbose`
- **Bool and valued options** -- flags with and without arguments
- **Option groups** -- multiple named groups, each with its own set of options
- **Namespaces** -- nested option groups with dotted prefixes (`--server.port`)
- **Subcommands** -- nested command tree (`app server start`)
- **Command handlers** -- a per-command `Run` handler dispatched by `Execute`
- **Typed commands** -- `BindCommand`/`BindRoot` pair a command with its own typed config and handler
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

	"go.vitalvas.com/gokit/xflags"
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

	if err := p.Parse(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if p.Handled() {
		return
	}

	fmt.Printf("name=%s verbose=%d\n", opts.Name, opts.Verbose)
}
```

## Examples

The examples below are complete programs. Runnable versions of each live in
[`example_test.go`](example_test.go) and render on
[pkg.go.dev](https://pkg.go.dev/go.vitalvas.com/gokit/xflags#pkg-examples).

### Subcommands with handlers

Each command owns its logic in a `Run` handler. `Execute` parses the arguments
and dispatches to the deepest selected command, passing a context the caller
controls.

```go
package main

import (
	"context"
	"fmt"
	"os"

	"go.vitalvas.com/gokit/xflags"
)

type serveOptions struct {
	Port int    `long:"port" default:"8080"`
	Host string `long:"host" default:"localhost"`
}

func main() {
	var serve serveOptions

	server := xflags.NewCommand("serve", "serve traffic")
	if err := server.AddGroup("", &serve); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	server.SetRun(func(ctx context.Context, cmd *xflags.Command, args []string) error {
		fmt.Printf("serving on %s:%d (files: %v)\n", serve.Host, serve.Port, args)
		return nil
	})

	p := xflags.New("app")
	if err := p.AddCommand(server); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := p.Execute(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// app serve --port 9090 a.txt -> serving on localhost:9090 (files: [a.txt])
```

### Root action plus subcommands

`./app` runs the root handler; `./app <name>` runs the matching subcommand. The
typed `BindRoot` and `BindCommand` hand each handler its own populated config.

```go
type rootConfig struct {
	Verbose bool `short:"v" long:"verbose"`
}

type pingConfig struct {
	Host string `long:"host" default:"localhost"`
}

func main() {
	p := xflags.New("app")

	if _, err := xflags.BindRoot(p, func(_ context.Context, _ *xflags.Command, cfg *rootConfig, _ []string) error {
		fmt.Printf("root, verbose=%t\n", cfg.Verbose)
		return nil
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, ping, err := xflags.BindCommand("ping", "ping a host",
		func(_ context.Context, _ *xflags.Command, cfg *pingConfig, _ []string) error {
			fmt.Printf("pinging %s\n", cfg.Host)
			return nil
		})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := p.AddCommand(ping); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := p.Execute(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// app -v            -> root, verbose=true
// app ping          -> pinging localhost
// app ping --host x -> pinging x
```

### Commands split across packages

A command carries no parent until it is attached, so each can be built in its own
package and exported as a `*xflags.Command`. The root package imports the value
and wires it in with `AddCommand` — the cobra-style layout.

```go
// package user
var Cmd = xflags.NewCommand("user", "manage users")

func init() {
	create := xflags.NewCommand("create", "create a user")
	create.SetRun(runCreate)
	Cmd.AddCommand(create)
}
```

```go
// package main
func main() {
	p := xflags.New("app")
	if err := p.AddCommand(user.Cmd, group.Cmd); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := p.Execute(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// app user create ...
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

Standalone func fields and builder `Func` callbacks run for each occurrence (or the environment/default fallback when absent). A callback paired with a value field, and callbacks registered through `Validate`, run once with the resolved value after precedence is applied. Paired callbacks receive the stored counter, slice, or map, including its accumulated values. The argument type must accept the value field's type. A returned error fails parsing.

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

`required:"true"` makes parsing fail unless the option is set by the command line or environment. A default does not satisfy this requirement. It is independent of the value callback.

```go
type Options struct {
	Name string `long:"name" required:"true"`
}
```

## Optional Values

An option marked `optional` may appear on the command line without a value. When it does, its `optional-value` is used (or the zero value if none is given). Setting `optional-value` implies `optional`.

For slices, omission appends a zero element. Optional maps need an explicit `optional-value` containing a `key=value` pair.

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

Subcommands form a nested tree. Each command has its own options and may have child commands. Build them as standalone commands and attach them, or declare them with a struct tag.

### Via the builder

A command is built on its own with `NewCommand`, given options and a handler, and then attached to a parent with `AddCommand`. Because a command carries no parent until it is attached, it can be constructed in its own package and wired into the tree elsewhere:

```go
// package server
var Cmd = xflags.NewCommand("server", "run the server")

func init() {
	Cmd.AddGroup("", &opts)
	Cmd.SetRun(run)
}
```

```go
// package main
p := xflags.New("app")
if err := p.AddCommand(server.Cmd, client.Cmd); err != nil {
	return err
}
```

`AddCommand` takes one or more pre-built commands and attaches nothing when any is nil, already attached, or has an invalid or duplicate name. Nest deeper by attaching to the returned command's children the same way.

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

After parsing successfully, `p.SelectedCommand()` returns the deepest selected `*Command` (the root when no subcommand was selected). Use its `Name()` method, or compare it with a command you built, to dispatch application logic manually when you do not use handlers.

### Command Handlers

Attach a handler to any command with `SetRun` to keep each command's logic in its own function. `Execute` takes a `context.Context` from the caller, parses the arguments, and then invokes the handler of the deepest selected command with that context, so no manual dispatch on `SelectedCommand()` is needed. The handler receives the context as its first argument, for cancellation and deadlines.

```go
server := xflags.NewCommand("server", "run the server")
server.AddGroup("", &serverOpts)
server.SetRun(func(ctx context.Context, cmd *xflags.Command, args []string) error {
	// serverOpts is already populated; args holds trailing positionals.
	return runServer(ctx, serverOpts, args)
})

p := xflags.New("app")
p.AddCommand(server)

if err := p.Execute(context.Background()); err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
```

A command with a handler accepts positional arguments: any token that is not a flag and does not name a subcommand, together with everything after it, becomes a positional argument and is passed to the handler as `args` (also available via `cmd.Args()`). Flags that appear after the first positional are treated as positionals, not parsed. Arguments after a `--` terminator are always positional. A command without a handler rejects positional arguments with an `unexpected argument` error.

`Execute` only dispatches; the lower-level `Parse` never calls a handler. When built-in `--help` or `--version` was handled, `Execute` returns without dispatching. When the selected command has no handler, `Execute` prints that command's help and returns nil. A handler's returned error is propagated unchanged.

### Typed Commands

`BindCommand` builds a standalone command bound to its own typed config and handler in one call, so each command can live in a fully isolated function -- in its own package -- with no closures over shared state and no type assertions. It allocates the config struct, binds its tags, and attaches a handler that receives that same `*T`. The command carries no parent until attached with `AddCommand`.

The handler takes the context first, then the command, mirroring the untyped `SetRun` handler (`func(ctx context.Context, cmd *xflags.Command, args []string) error`) with the typed config inserted:

```go
type ServerConfig struct {
	Port int    `long:"port" default:"8080"`
	Host string `long:"host"`
}

// An isolated handler -- a free function in its own file:
func runServer(ctx context.Context, cmd *xflags.Command, cfg *ServerConfig, args []string) error {
	return serve(ctx, cfg.Host, cfg.Port, args)
}

cfg, server, err := xflags.BindCommand("server", "run the server", runServer)
if err != nil {
	return err
}
_ = cfg // same *ServerConfig the handler receives; read it after Execute if needed

p := xflags.New("app")
p.AddCommand(server)

if err := p.Execute(context.Background()); err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
```

Child commands nest by attaching to the returned `*Command`:

```go
_, server, _ := xflags.BindCommand("server", "run the server", runServer)
_, start, _ := xflags.BindCommand("start", "start the server", runServerStart)
server.AddCommand(start)
```

A command with no bound config uses `NewCommand` plus `SetRun` instead, whose handler is `func(ctx context.Context, cmd *xflags.Command, args []string) error` -- the same shape without the config argument.

`BindRoot` is the typed counterpart for the root command, invoked by `Execute` when no subcommand is selected:

```go
cfg, err := xflags.BindRoot(p, func(ctx context.Context, cmd *xflags.Command, cfg *RootConfig, args []string) error {
	return nil
})
```

Both take `T any`; a non-struct `T` fails at bind time with the same error as `AddGroup`. Pass a nil handler to bind only the config (for a command that just groups subcommands). The dispatch rules, positional-argument handling, and error propagation are identical to `SetRun`.

## Help

`--help` and `-h` are added automatically to the root command and every subcommand unless overridden by user options. When present, the parser prints a formatted help message and stops parsing. `Parse` returns nil and `p.Handled()` returns true; the caller should then return without running application logic. Help and version skip required checks and final validation, including those on parent commands. Standalone callbacks that appeared earlier have already run.

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

The arguments to `Parse` are optional. `p.Parse()` parses `os.Args[1:]`; `p.Parse("--name", "x")` parses an explicit list. To parse a `[]string` slice, spread it with `p.Parse(args...)`, and to parse an explicitly empty list use `p.Parse([]string{}...)`.

Each `Parse` starts from destination values captured at the first parse, then resolves the current command line, environment, and defaults again. Failed parses may leave partial destination values and callback side effects; the next parse resets destinations and occurrence state. Parsers are not safe for concurrent use. `Handled()` resets on every parse. After an error, `SelectedCommand()` may describe a partially parsed path.

```go
if err := p.Parse(); err != nil {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
if p.Handled() {
	return
}
```
