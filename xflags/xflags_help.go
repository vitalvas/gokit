package xflags

import (
	"fmt"
	"strings"
)

// help renders the formatted help message for the command.
func (c *Command) help() string {
	var b strings.Builder

	b.WriteString("Usage:\n")
	fmt.Fprintf(&b, "  %s\n", c.usageLine())
	if c.description != "" {
		fmt.Fprintf(&b, "\n%s\n", c.description)
	}

	for _, g := range c.groups {
		visible := visibleOptions(g.options)
		if len(visible) == 0 {
			continue
		}
		title := g.title
		if title == "" {
			title = "Options"
		}
		fmt.Fprintf(&b, "\n%s:\n", title)
		writeOptions(&b, visible)
	}

	if len(c.commands) > 0 {
		b.WriteString("\nCommands:\n")
		width := 0
		for _, sub := range c.commands {
			if len(sub.name) > width {
				width = len(sub.name)
			}
		}
		for _, sub := range c.commands {
			fmt.Fprintf(&b, "  %-*s  %s\n", width, sub.name, sub.description)
		}
	}

	b.WriteString("\n")
	c.writeBuiltins(&b)

	return b.String()
}

// usageLine builds the one-line usage summary.
func (c *Command) usageLine() string {
	parts := []string{c.commandPath()}
	if len(c.options) > 0 {
		parts = append(parts, "[options]")
	}
	if len(c.commands) > 0 {
		parts = append(parts, "<command>")
	}
	return strings.Join(parts, " ")
}

// commandPath returns the space-joined path from the root to this command.
func (c *Command) commandPath() string {
	var names []string
	for cur := c; cur != nil; cur = cur.parent {
		names = append([]string{cur.name}, names...)
	}
	return strings.Join(names, " ")
}

// shortTaken reports whether a short name is already used by an option.
func (c *Command) shortTaken(name string) bool {
	_, ok := c.byShort[name]
	return ok
}

// writeOptions renders an aligned list of options.
func writeOptions(b *strings.Builder, options []*Option) {
	names := make([]string, len(options))
	width := 0
	for i, opt := range options {
		names[i] = optionColumn(opt)
		if len(names[i]) > width {
			width = len(names[i])
		}
	}
	for i, opt := range options {
		desc := opt.Description
		if opt.Default != "" {
			if desc != "" {
				desc += " "
			}
			desc += fmt.Sprintf("(default: %s)", opt.Default)
		}
		fmt.Fprintf(b, "  %-*s  %s\n", width, names[i], strings.TrimRight(desc, " "))
	}
}

// visibleOptions filters out hidden options for help rendering.
func visibleOptions(options []*Option) []*Option {
	visible := make([]*Option, 0, len(options))
	for _, opt := range options {
		if !opt.Hidden {
			visible = append(visible, opt)
		}
	}
	return visible
}

// optionColumn renders the short/long name column for an option, including a
// value-name placeholder for valued options.
func optionColumn(opt *Option) string {
	var names string
	switch {
	case opt.Short != "" && opt.Long != "":
		names = fmt.Sprintf("-%s, --%s", opt.Short, opt.Long)
	case opt.Short != "":
		names = fmt.Sprintf("-%s", opt.Short)
	default:
		names = fmt.Sprintf("    --%s", opt.Long)
	}
	if placeholder := opt.valuePlaceholder(); placeholder != "" {
		return fmt.Sprintf("%s %s", names, placeholder)
	}
	return names
}

// valuePlaceholder returns the help placeholder for a valued option, or an
// empty string for switches and callbacks that take no argument.
func (opt *Option) valuePlaceholder() string {
	if opt.takesNoArg() {
		return ""
	}
	if opt.ValueName != "" {
		return opt.ValueName
	}
	return ""
}

// writeBuiltins renders the built-in help and version entries.
func (c *Command) writeBuiltins(b *strings.Builder) {
	write := func(short, long, description string) {
		opt := &Option{Description: description, isBool: true}
		if !c.shortTaken(short) {
			opt.Short = short
		}
		if _, taken := c.byLong[long]; !taken {
			opt.Long = long
		}
		if opt.Short != "" || opt.Long != "" {
			writeOptions(b, []*Option{opt})
		}
	}
	write("h", "help", "show this help message")
	if c.rootVersion() != "" {
		write("v", "version", "show version information")
	}
}
