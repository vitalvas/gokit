// Package xflags is a command-line flag parser that fills a struct from tags
// or is built programmatically. It supports nested subcommands, option groups,
// namespaces, environment-variable fallback, default values, repeated options,
// maps, function callbacks, and per-option validation.
//
// # Declaring options
//
// Options are declared with struct tags, the builder API, or both. Each struct
// field maps to one option. The recognized tags are:
//
//	short          single-character short name, e.g. short:"v" -> -v
//	long           long name, e.g. long:"verbose" -> --verbose
//	description    help text
//	default        default value used when no source provides one
//	env            environment variable name to fall back to
//	choice         repeated-option policy: append, count, last, or merge
//	required       required:"true" fails parsing if the option is never set
//	namespace      on a nested struct field, prefixes its options
//	value-name     placeholder shown in help, e.g. value-name:"FILE"
//	hidden         hidden:"true" omits the option from the help message
//	base           integer radix, e.g. base:"16"; base:"0" auto-detects prefixes
//	optional       optional:"true" allows the option to appear without a value
//	optional-value value used when an optional option is given without one
//	command        on a struct field, declares a subcommand from that struct
//
// The builder API mirrors the tags: BoolVar, StringVar, IntVar, CountVar,
// StringSliceVar, StringMapVar, and Func register options bound to a pointer,
// while SetHidden, SetValueName, SetBase, SetOptionalValue, and Validate adjust
// a registered option by its long name. Tag-declared and builder-declared
// options can be mixed on the same parser.
//
// # Supported types
//
// All primitive Go types are supported: bool, string, every sized int and uint,
// float32, and float64, plus time.Duration (parsed with time.ParseDuration).
// Pointers to these types are allocated as needed. Slice fields collect repeated
// occurrences and map fields collect key=value pairs; map keys must be strings.
//
// # Short and long syntax
//
// Both spellings accept a value with a space or an equals sign, and short flags
// may be clustered:
//
//	--name value    --name=value
//	-n value        -n=value        -nvalue
//	-abc            cluster of bool/count switches (-a -b -c)
//	-abo=file       trailing valued short in a cluster
//
// A bool option is a switch: --debug sets true and --debug=false sets false. A
// lone "--" stops flag parsing; a subcommand may still follow it.
//
// # Repeated options
//
// When an option appears more than once the policy is set with the choice tag,
// and the field type must match:
//
//	append   slice    every occurrence is appended
//	count    integer  each occurrence increments the value (e.g. -vvv -> 3)
//	last     scalar   the last occurrence wins
//	merge    map       key=value pairs accumulate
//
// Without a choice tag a scalar defaults to last, a slice to append, and a map
// to merge.
//
// # Default values and environment fallback
//
// The default tag supplies a value used when nothing else sets the option. The
// env tag names an environment variable checked before the default; for slice
// and map options the environment value is split on commas. For each option the
// effective value is resolved as
//
//	command line > environment variable > default
//
// A value supplied only by a default does not satisfy required.
//
// # Validation and callbacks
//
// Each option may carry a single func(value) error. It runs once with the
// resolved value; a returned error fails parsing. The same function covers both
// reacting to an option and validating its value. Attach it with a func-typed
// struct field (func(T) error or func() error) or with the Validate builder
// method. Pairing a value field and a func field that share a long name both
// stores the value and runs the callback.
//
// # Option groups and namespaces
//
// AddGroup registers a struct's options under a titled group for help output. A
// nested struct field tagged with namespace prefixes its options with a dotted
// path (namespaces may nest), for example --server.port.
//
// # Subcommands
//
// Commands form a nested tree. AddCommand registers a subcommand with its own
// options and optional child commands; a struct field tagged command declares
// one from the field's struct. After parsing, SelectedCommand reports the
// deepest command reached.
//
// # Built-in help and version
//
// --help and -h are added to every command and print a formatted help message.
// --version is added when SetVersion is called and prints the program version;
// it takes the -v shorthand only when -v is not already used. Both are handled
// internally: Parse prints them, returns no error, and Handled reports that a
// built-in ran. A user option with the same name overrides the built-in.
//
// # Errors and reuse
//
// Parse returns errors and never calls os.Exit; the caller decides how to react.
// Parse resets option state first, so a parser may be reused across several
// argument slices.
//
// # Testing a parser
//
// Several methods support writing tests for a parser without inspecting the
// destination struct. SetArgs configures the arguments Execute parses, so a
// test can set arguments and call Execute with no parameters; Execute falls back
// to os.Args[1:] when SetArgs was not called. IsSet reports whether an option
// was provided by the command line or environment (a default alone does not
// count), and Count reports how many times a counting option occurred. Find
// walks the command tree to locate a subcommand by name without parsing,
// returning the command and its remaining arguments. Help returns the rendered
// help message so a test can assert on it directly, and SetOutput redirects the
// built-in help and version text to a writer, so a test can capture it in a
// bytes.Buffer instead of standard output.
package xflags
