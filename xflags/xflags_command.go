package xflags

import (
	"context"
	"fmt"
)

// NewCommand creates a standalone command with the given name and one-line short
// description. The command has no parent until it is attached to one with
// AddCommand, so it can be constructed in its own package and wired into a tree
// elsewhere. Options are added with AddGroup or the builder, and a handler with
// SetRun.
func NewCommand(name, short string) *Command {
	c := newCommand(name)
	c.description = short
	return c
}

func newCommand(name string) *Command {
	return &Command{
		name:      name,
		byLong:    make(map[string]*Option),
		byShort:   make(map[string]*Option),
		byCommand: make(map[string]*Command),
	}
}

// SetDescription sets the command's help description.
func (c *Command) SetDescription(description string) {
	c.description = description
}

// SetRun attaches a handler that Execute invokes for this command when it is the
// deepest command selected by parsing. The handler receives the context Execute
// was given, the command, and its trailing positional arguments. A command with
// a run handler accepts positional arguments; without one, a positional argument
// is an error. Pass nil to clear a previously set handler.
func (c *Command) SetRun(fn func(ctx context.Context, cmd *Command, args []string) error) {
	c.run = fn
}

// Args returns the trailing positional arguments collected for this command
// during the last parse. It is empty before parsing or when the command took no
// positional arguments.
func (c *Command) Args() []string {
	return c.positionals
}

// AddCommand attaches one or more pre-built subcommands to this command, setting
// each child's parent. Children are typically constructed with NewCommand or
// BindCommand, possibly in another package, and wired into the tree here. It
// returns an error on an invalid or duplicate command name, or when a child is
// nil or already has a parent; on error no child is attached.
func (c *Command) AddCommand(children ...*Command) error {
	for _, child := range children {
		if child == nil {
			return fmt.Errorf("xflags: AddCommand requires a non-nil command")
		}
		if child.parent != nil {
			return fmt.Errorf("xflags: command %q already attached to %q", child.name, child.parent.name)
		}
		if !validName(child.name) {
			return fmt.Errorf("xflags: invalid command name %q", child.name)
		}
		if _, exists := c.byCommand[child.name]; exists {
			return fmt.Errorf("xflags: duplicate command %q", child.name)
		}
	}

	for _, child := range children {
		child.parent = c
		c.commands = append(c.commands, child)
		c.byCommand[child.name] = child
	}
	return nil
}

// newChildCommand builds a command from a config struct and attaches it to c. It
// backs the struct-tag command path, where a command and its options are
// declared together. config may be nil for a grouping command.
func (c *Command) newChildCommand(name string, config any) (*Command, error) {
	sub := newCommand(name)
	if config != nil {
		if err := sub.AddGroup("", config); err != nil {
			return nil, err
		}
	}
	if err := c.AddCommand(sub); err != nil {
		return nil, err
	}
	return sub, nil
}
