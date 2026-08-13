// Package xflags is a command-line flag parser that fills a struct from tags
// or is built programmatically. It supports nested subcommands, option groups,
// namespaces, environment-variable fallback, default values, and per-option
// validation callbacks.
package xflags

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// itoa formats an int for use as a default-value string.
func itoa(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

// ErrHelp is returned internally when the built-in help flag is triggered. It
// is handled inside Parse and never surfaces to the caller.
var errHelp = errors.New("xflags: help requested")

// errVersion is returned internally when the built-in version flag is
// triggered. It is handled inside Parse and never surfaces to the caller.
var errVersion = errors.New("xflags: version requested")

// choicePolicy controls how repeated occurrences of an option are combined.
type choicePolicy int

const (
	choiceLast   choicePolicy = iota // last occurrence wins (default for scalars)
	choiceAppend                     // every occurrence is appended (slices)
	choiceCount                      // each occurrence increments an integer
	choiceMerge                      // key=value pairs are merged (maps)
)

// Option describes a single command-line option.
type Option struct {
	Short       string       // single-character short name without the dash
	Long        string       // long name without the dashes
	Description string       // help text
	Default     string       // default value, applied when no source provides one
	Env         string       // environment variable to fall back to
	Required    bool         // parsing fails if the option is never set
	ValueName   string       // placeholder shown in help, e.g. FILE
	Hidden      bool         // omit the option from the help message
	choice      choicePolicy // repeated-option policy
	choiceSet   bool         // whether choice was set explicitly

	base           int    // radix for integer parsing; 0 auto-detects 0x/0o/0b prefixes
	baseSet        bool   // whether base was set explicitly
	optional       bool   // the option's value may be omitted on the command line
	optionalValue  string // value used when an optional option is given without one
	optionalValSet bool   // whether optionalValue was set explicitly

	value    reflect.Value // destination value (settable)
	callback reflect.Value // optional func(T) error, invalid if none
	validate func(any) error

	isBool bool // true when the option takes no argument
	set    bool // whether any source provided a value
	count  int  // number of command-line occurrences
}

// Command is a node in the command tree. The root parser is itself a command.
type Command struct {
	name        string
	description string
	version     string

	options   []*Option          // flat list of all options across groups
	byLong    map[string]*Option // long-name lookup
	byShort   map[string]*Option // short-name lookup
	groups    []*group           // option groups, in declaration order
	commands  []*Command         // subcommands, in declaration order
	byCommand map[string]*Command

	parent *Command
}

// group is a named set of options for help organization.
type group struct {
	title   string
	options []*Option
}

// Parser is the root of a command tree.
type Parser struct {
	*Command
}

// New creates a new parser with the given program name.
func New(name string) *Parser {
	return &Parser{Command: newCommand(name, nil)}
}

func newCommand(name string, parent *Command) *Command {
	return &Command{
		name:      name,
		byLong:    make(map[string]*Option),
		byShort:   make(map[string]*Option),
		byCommand: make(map[string]*Command),
		parent:    parent,
	}
}

// SetVersion sets the version string. When set, a built-in --version option is
// added. When empty, no version option exists.
func (p *Parser) SetVersion(version string) {
	p.version = version
}

// SetDescription sets the command's help description.
func (c *Command) SetDescription(description string) {
	c.description = description
}

// AddGroup registers all options declared as struct tags on config under a
// named group. config must be a pointer to a struct.
func (c *Command) AddGroup(title string, config any) error {
	v := reflect.ValueOf(config)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("xflags: AddGroup requires a non-nil pointer to a struct, got %T", config)
	}
	elem := v.Elem()
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("xflags: AddGroup requires a pointer to a struct, got %T", config)
	}

	g := &group{title: title}
	if err := c.collectStruct(elem, "", g); err != nil {
		return err
	}
	c.groups = append(c.groups, g)
	return nil
}

// AddCommand registers a subcommand whose options come from config. config may
// be nil for a command that only groups further subcommands.
func (c *Command) AddCommand(name string, config any) (*Command, error) {
	if _, exists := c.byCommand[name]; exists {
		return nil, fmt.Errorf("xflags: duplicate command %q", name)
	}

	sub := newCommand(name, c)
	if config != nil {
		if err := sub.AddGroup("", config); err != nil {
			return nil, err
		}
	}

	c.commands = append(c.commands, sub)
	c.byCommand[name] = sub
	return sub, nil
}

// Validate attaches a validation callback to the option with the given long
// name. The callback runs once with the resolved value; a returned error fails
// parsing.
func (c *Command) Validate(long string, fn func(any) error) error {
	opt, ok := c.byLong[long]
	if !ok {
		return fmt.Errorf("xflags: no option named %q", long)
	}
	opt.validate = fn
	return nil
}

// addOption registers an option, checking for duplicate names.
func (c *Command) addOption(opt *Option, g *group) error {
	if opt.Long == "" && opt.Short == "" {
		return fmt.Errorf("xflags: option must have a short or long name")
	}
	if opt.Long != "" {
		if _, exists := c.byLong[opt.Long]; exists {
			return fmt.Errorf("xflags: duplicate long option %q", opt.Long)
		}
		c.byLong[opt.Long] = opt
	}
	if opt.Short != "" {
		if _, exists := c.byShort[opt.Short]; exists {
			return fmt.Errorf("xflags: duplicate short option %q", opt.Short)
		}
		c.byShort[opt.Short] = opt
	}
	c.options = append(c.options, opt)
	if g != nil {
		g.options = append(g.options, opt)
	}
	return nil
}
