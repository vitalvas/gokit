package xflags_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.vitalvas.com/gokit/xflags"
)

// Parsing root options declared as struct tags across the supported kinds, then
// reading the populated struct after Parse. Defaults apply when a flag is
// absent.
func Example() {
	var opts struct {
		Name    string        `long:"name" default:"world"`
		Verbose bool          `short:"v" long:"verbose"`
		Port    int           `long:"port" default:"8080"`
		Timeout time.Duration `long:"timeout" default:"30s"`
	}

	p := xflags.New("greet")
	if err := p.AddGroup("", &opts); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := p.Parse("--name=gopher", "-v", "--port=9090"); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("name=%s verbose=%t port=%d timeout=%s\n", opts.Name, opts.Verbose, opts.Port, opts.Timeout)
	// Output: name=gopher verbose=true port=9090 timeout=30s
}

// Repeated options: a slice appends every occurrence, a counter increments, and
// a map collects key=value pairs.
func Example_repeated() {
	var opts struct {
		Tags    []string          `long:"tag" choice:"append"`
		Verbose int               `short:"v" choice:"count"`
		Labels  map[string]string `long:"label"`
	}

	p := xflags.New("app")
	if err := p.AddGroup("", &opts); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := p.Parse("--tag=a", "--tag=b", "-vvv", "--label=env=prod"); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("tags=%v verbose=%d env=%s\n", opts.Tags, opts.Verbose, opts.Labels["env"])
	// Output: tags=[a b] verbose=3 env=prod
}

// Namespaces prefix a nested struct's options with a dotted path.
func Example_namespaces() {
	var opts struct {
		Server struct {
			Port int    `long:"port" default:"8080"`
			Host string `long:"host" default:"0.0.0.0"`
		} `namespace:"server"`
	}

	p := xflags.New("app")
	if err := p.AddGroup("", &opts); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := p.Parse("--server.port=9090"); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("%s:%d\n", opts.Server.Host, opts.Server.Port)
	// Output: 0.0.0.0:9090
}

// An option with an env tag falls back to the named environment variable when
// no command-line argument is given.
func Example_environmentFallback() {
	os.Setenv("EXAMPLE_API_TOKEN", "from-env")
	defer os.Unsetenv("EXAMPLE_API_TOKEN")

	var opts struct {
		Token string `long:"token" env:"EXAMPLE_API_TOKEN"`
	}

	p := xflags.New("app")
	if err := p.AddGroup("", &opts); err != nil {
		fmt.Println("error:", err)
		return
	}
	// No --token argument, so the environment value is used.
	if err := p.Parse([]string{}...); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(opts.Token)
	// Output: from-env
}

// A required option fails parsing when it is never set.
func Example_required() {
	var opts struct {
		User string `long:"user" required:"true"`
	}

	p := xflags.New("app")
	if err := p.AddGroup("", &opts); err != nil {
		fmt.Println("error:", err)
		return
	}
	// Parse an explicitly empty argument list, so --user is never set.
	if err := p.Parse([]string{}...); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(opts.User)
	// Output: error: required option --user is not set
}

// A validation callback registered with Validate runs once with the resolved
// value; a returned error fails parsing.
func Example_validation() {
	var opts struct {
		Port int `long:"port"`
	}

	p := xflags.New("app")
	if err := p.AddGroup("", &opts); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := p.Validate("port", func(v any) error {
		if port := v.(int); port < 1 || port > 65535 {
			return fmt.Errorf("port %d out of range", port)
		}
		return nil
	}); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := p.Parse("--port=99999"); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(opts.Port)
	// Output: error: option --port: port 99999 out of range
}

// The builder API registers options programmatically instead of via struct tags.
func Example_builder() {
	var (
		name    string
		verbose int
	)

	p := xflags.New("app")
	if err := p.StringVar(&name, "name", "n", "world", "name to greet"); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := p.CountVar(&verbose, "verbose", "v", "increase verbosity"); err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := p.Parse("-n", "gopher", "-vv"); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("name=%s verbose=%d\n", name, verbose)
	// Output: name=gopher verbose=2
}

// SetVersion adds a built-in --version flag. Execute handles it internally,
// prints the version, and reports Handled so the caller can return early.
func Example_version() {
	p := xflags.New("app")
	p.SetVersion("1.2.3")
	p.SetArgs([]string{"--version"})

	if err := p.Execute(context.Background()); err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("handled:", p.Handled())
	// Output:
	// app version 1.2.3
	// handled: true
}

// A root handler attached with SetRun is dispatched by Execute, receiving the
// context, the command, and its trailing positional arguments.
func ExampleParser_Execute() {
	p := xflags.New("app")
	p.SetRun(func(_ context.Context, cmd *xflags.Command, args []string) error {
		fmt.Printf("ran %s with %v\n", cmd.Name(), args)
		return nil
	})

	p.SetArgs([]string{"a", "b"})
	if err := p.Execute(context.Background()); err != nil {
		fmt.Println("error:", err)
		return
	}
	// Output: ran app with [a b]
}

// Building a command tree: standalone commands are created with NewCommand and
// wired in with AddCommand. Execute dispatches to the deepest selected command.
func ExampleCommand_AddCommand() {
	start := xflags.NewCommand("start", "start the server")
	start.SetRun(func(_ context.Context, _ *xflags.Command, _ []string) error {
		fmt.Println("server started")
		return nil
	})

	server := xflags.NewCommand("server", "manage the server")
	if err := server.AddCommand(start); err != nil {
		fmt.Println("error:", err)
		return
	}

	p := xflags.New("app")
	if err := p.AddCommand(server); err != nil {
		fmt.Println("error:", err)
		return
	}

	p.SetArgs([]string{"server", "start"})
	if err := p.Execute(context.Background()); err != nil {
		fmt.Println("error:", err)
		return
	}
	// Output: server started
}

// BindCommand pairs a command with its own typed config and an isolated
// handler. The handler receives the populated config, so each command can live
// in its own function with no shared state.
func ExampleBindCommand() {
	type serveConfig struct {
		Port int    `long:"port" default:"8080"`
		Host string `long:"host" default:"localhost"`
	}

	_, serve, err := xflags.BindCommand("serve", "serve traffic",
		func(_ context.Context, _ *xflags.Command, cfg *serveConfig, _ []string) error {
			fmt.Printf("serving on %s:%d\n", cfg.Host, cfg.Port)
			return nil
		})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	p := xflags.New("app")
	if err := p.AddCommand(serve); err != nil {
		fmt.Println("error:", err)
		return
	}

	p.SetArgs([]string{"serve", "--port=9090"})
	if err := p.Execute(context.Background()); err != nil {
		fmt.Println("error:", err)
		return
	}
	// Output: serving on localhost:9090
}

// BindRoot binds the root command to a typed config and handler. The root
// handler runs when no subcommand is selected; a named subcommand takes over
// instead.
func ExampleBindRoot() {
	type rootConfig struct {
		Verbose bool `short:"v" long:"verbose"`
	}

	p := xflags.New("app")
	if _, err := xflags.BindRoot(p, func(_ context.Context, _ *xflags.Command, cfg *rootConfig, _ []string) error {
		fmt.Printf("root verbose=%t\n", cfg.Verbose)
		return nil
	}); err != nil {
		fmt.Println("error:", err)
		return
	}

	hello := xflags.NewCommand("hello", "say hello")
	hello.SetRun(func(_ context.Context, _ *xflags.Command, _ []string) error {
		fmt.Println("hello")
		return nil
	})
	if err := p.AddCommand(hello); err != nil {
		fmt.Println("error:", err)
		return
	}

	// No subcommand: the root handler runs.
	p.SetArgs([]string{"-v"})
	if err := p.Execute(context.Background()); err != nil {
		fmt.Println("error:", err)
		return
	}

	// A named subcommand runs instead of the root.
	p.SetArgs([]string{"hello"})
	if err := p.Execute(context.Background()); err != nil {
		fmt.Println("error:", err)
		return
	}
	// Output:
	// root verbose=true
	// hello
}
