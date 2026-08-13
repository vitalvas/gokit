package xflags

import (
	"fmt"
	"os"
	"strings"
)

// Parse parses the given arguments, dispatches subcommands, applies precedence
// (command line > environment > default), runs validation callbacks, and
// checks required options. Built-in --help and --version are handled
// internally and do not surface as errors.
func (p *Parser) Parse(args []string) error {
	err := p.parse(args)
	switch err {
	case errHelp:
		fmt.Fprint(os.Stdout, p.helpTarget(args).help())
		return nil
	case errVersion:
		fmt.Fprintf(os.Stdout, "%s version %s\n", p.name, p.version)
		return nil
	default:
		return err
	}
}

// helpTarget resolves which command a help request referred to by re-walking
// the leading subcommand path in args.
func (c *Command) helpTarget(args []string) *Command {
	cur := c
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		next, ok := cur.byCommand[arg]
		if !ok {
			break
		}
		cur = next
	}
	return cur
}

// parse consumes args for this command, delegating to a subcommand when the
// first non-flag token names one.
func (c *Command) parse(args []string) error {
	i := 0
	for i < len(args) {
		arg := args[i]

		switch {
		case arg == "--":
			i++
			// After the terminator flag parsing stops. The next token may name
			// a subcommand; any other leftover token is an unexpected positional
			// (consistent with the pre-terminator path).
			if i < len(args) {
				if sub, ok := c.byCommand[args[i]]; ok {
					if err := c.finalize(); err != nil {
						return err
					}
					return sub.parse(args[i+1:])
				}
				return fmt.Errorf("xflags: unexpected argument %q", args[i])
			}
			return c.finalize()

		case strings.HasPrefix(arg, "--"):
			consumed, err := c.parseLong(arg[2:], args[i+1:])
			if err != nil {
				return err
			}
			i += 1 + consumed

		case len(arg) > 1 && arg[0] == '-':
			consumed, err := c.parseShort(arg[1:], args[i+1:])
			if err != nil {
				return err
			}
			i += 1 + consumed

		default:
			if sub, ok := c.byCommand[arg]; ok {
				if err := c.finalize(); err != nil {
					return err
				}
				return sub.parse(args[i+1:])
			}
			return fmt.Errorf("xflags: unexpected argument %q", arg)
		}
	}

	return c.finalize()
}

// parseLong handles a --name or --name=value token. It returns how many of the
// following args it consumed (0 or 1).
func (c *Command) parseLong(token string, rest []string) (int, error) {
	name, inlineVal, hasInline := strings.Cut(token, "=")

	opt, ok := c.byLong[name]
	if !ok {
		// Fall back to the built-in flags only when the user has not declared
		// an option with the same long name.
		if name == "help" {
			return 0, errHelp
		}
		if name == "version" && c.rootVersion() != "" {
			return 0, errVersion
		}
		return 0, fmt.Errorf("xflags: unknown option --%s", name)
	}

	return c.applyToken(opt, inlineVal, hasInline, rest)
}

// parseShort handles a cluster of short flags such as -v, -vvv, -abc, -ovalue,
// or -o=value.
func (c *Command) parseShort(cluster string, rest []string) (int, error) {
	runes := []rune(cluster)
	for idx, r := range runes {
		name := string(r)

		opt, ok := c.byShort[name]
		if !ok {
			// Fall back to the built-in flags only when the user has not
			// declared an option with the same short name.
			if name == "h" {
				return 0, errHelp
			}
			if name == "v" && c.rootVersion() != "" {
				return 0, errVersion
			}
			return 0, fmt.Errorf("xflags: unknown option -%s", name)
		}

		// The remainder of the cluster after this flag. A leading '=' is an
		// explicit value separator (-o=value); otherwise the remainder is an
		// attached value (-ovalue).
		remainder := string(runes[idx+1:])
		inlineVal, hasInline := "", false
		if after, found := strings.CutPrefix(remainder, "="); found {
			inlineVal, hasInline, remainder = after, true, ""
		}

		if opt.takesNoArg() {
			if hasInline {
				return 0, opt.applyString(inlineVal)
			}
			if err := opt.applyString(""); err != nil {
				return 0, err
			}
			continue
		}

		// Valued option: an inline value, the attached remainder, or the next arg.
		if hasInline {
			return 0, opt.applyString(inlineVal)
		}
		if remainder != "" {
			return 0, opt.applyString(remainder)
		}
		return c.applyToken(opt, "", false, rest)
	}
	return 0, nil
}

// applyToken applies a value to a valued or bool option, taking the next arg
// when a valued option has no inline value.
func (c *Command) applyToken(opt *Option, inlineVal string, hasInline bool, rest []string) (int, error) {
	_ = c
	if opt.takesNoArg() {
		if hasInline {
			return 0, opt.applyString(inlineVal)
		}
		return 0, opt.applyString("")
	}

	if hasInline {
		return 0, opt.applyString(inlineVal)
	}

	// An optional-value option may appear without a value. It only consumes the
	// next token when that token is not itself a flag.
	if opt.optional {
		if len(rest) == 0 || looksLikeFlag(rest[0]) {
			return 0, opt.applyString(opt.optionalValue)
		}
		return 1, opt.applyString(rest[0])
	}

	if len(rest) == 0 {
		return 0, fmt.Errorf("xflags: option %s expects a value", optionName(opt))
	}
	return 1, opt.applyString(rest[0])
}

// looksLikeFlag reports whether a token is a flag rather than a value. A lone
// "-" and negative numbers are treated as values, not flags.
func looksLikeFlag(token string) bool {
	if len(token) < 2 || token[0] != '-' {
		return false
	}
	if token == "--" {
		return true
	}
	// A negative number (-5, -1.2) is a value, not a flag.
	c := token[1]
	if c == '.' || (c >= '0' && c <= '9') {
		return false
	}
	return true
}

// takesNoArg reports whether the option consumes no command-line argument: a
// bool switch, a counter, or a zero-argument callback.
func (opt *Option) takesNoArg() bool {
	if opt.choice == choiceCount {
		return true
	}
	if opt.callback.IsValid() && opt.callback.Type().NumIn() == 0 && !opt.value.IsValid() {
		return true
	}
	return opt.isBool
}

// rootVersion returns the version string set on the root parser.
func (c *Command) rootVersion() string {
	root := c
	for root.parent != nil {
		root = root.parent
	}
	return root.version
}

// finalize applies environment and default fallbacks, runs validation, and
// enforces required options for this command.
func (c *Command) finalize() error {
	for _, opt := range c.options {
		if err := opt.resolve(); err != nil {
			return err
		}
	}
	return nil
}

// resolve applies precedence for a single option: command line beats
// environment, which beats default. It then runs validation and required
// checks.
func (opt *Option) resolve() error {
	if !opt.set {
		if val, ok := opt.envValue(); ok {
			if err := opt.applyEnvMulti(val); err != nil {
				return err
			}
		} else if opt.Default != "" {
			if err := opt.applyString(opt.Default); err != nil {
				return err
			}
			// A default should not, by itself, satisfy required.
			opt.set = false
		}
	}

	if opt.Required && !opt.set {
		return fmt.Errorf("xflags: required option %s is not set", optionName(opt))
	}

	return opt.runValidate()
}

// runValidate invokes the validation callback registered via Validate.
func (opt *Option) runValidate() error {
	if opt.validate == nil || !opt.value.IsValid() {
		return nil
	}
	if err := opt.validate(opt.value.Interface()); err != nil {
		return fmt.Errorf("xflags: option %s: %w", optionName(opt), err)
	}
	return nil
}
