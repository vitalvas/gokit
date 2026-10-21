package xflags

import "context"

// BindCommand builds a standalone command whose options come from a freshly
// allocated *T, binds that struct's tags, and attaches run as the command's
// handler. The command has no parent until attached with AddCommand, so it can
// be constructed in its own package and wired into a tree elsewhere. Execute
// calls run with its context, the command, the populated *T, and the command's
// trailing positional arguments when the command is selected.
//
// The run handler takes the context first, then the command, mirroring the
// untyped SetRun handler (func(ctx context.Context, cmd *Command, args []string)
// error) with the typed config inserted. A command with no bound config uses
// NewCommand plus SetRun instead.
//
// The command owns its config: the returned *T is the same pointer passed to
// run, so a handler can live in its own function with no shared state. The
// returned *Command accepts further subcommands. T must be a struct type.
func BindCommand[T any](name, short string, run func(ctx context.Context, cmd *Command, cfg *T, args []string) error) (*T, *Command, error) {
	cfg := new(T)
	cmd := NewCommand(name, short)
	if err := cmd.AddGroup("", cfg); err != nil {
		return nil, nil, err
	}
	if run != nil {
		cmd.run = func(ctx context.Context, c *Command, args []string) error { return run(ctx, c, cfg, args) }
	}
	return cfg, cmd, nil
}

// BindRoot binds a freshly allocated *T to the parser's root command and
// attaches run as the root handler, invoked by Execute when no subcommand is
// selected. The run handler takes the context first, then the command, like
// BindCommand. It is the typed counterpart of AddGroup plus SetRun on the root.
// T must be a struct type.
func BindRoot[T any](p *Parser, run func(ctx context.Context, cmd *Command, cfg *T, args []string) error) (*T, error) {
	cfg := new(T)
	if err := p.AddGroup("", cfg); err != nil {
		return nil, err
	}
	if run != nil {
		p.run = func(ctx context.Context, c *Command, args []string) error { return run(ctx, c, cfg, args) }
	}
	return cfg, nil
}
