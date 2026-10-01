package xflags

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// collectStruct walks a struct value and registers its fields as options,
// nested namespaces, or subcommands. prefix is the accumulated namespace path.
func (c *Command) collectStruct(v reflect.Value, prefix string, g *group) error {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			// Unexported field.
			continue
		}
		fieldValue := v.Field(i)

		if cmd, ok := field.Tag.Lookup("command"); ok {
			if err := c.collectCommand(cmd, field, fieldValue); err != nil {
				return err
			}
			continue
		}

		if ns, ok := field.Tag.Lookup("namespace"); ok {
			if err := c.collectNamespace(ns, prefix, fieldValue, g); err != nil {
				return err
			}
			continue
		}

		if !hasOptionTags(field) {
			continue
		}

		if err := c.collectOption(field, fieldValue, prefix, g); err != nil {
			return err
		}
	}

	return nil
}

// hasOptionTags reports whether a field declares an option via short or long.
func hasOptionTags(field reflect.StructField) bool {
	if _, ok := field.Tag.Lookup("short"); ok {
		return true
	}
	if _, ok := field.Tag.Lookup("long"); ok {
		return true
	}
	return false
}

// collectCommand registers a subcommand declared by a struct tag.
func (c *Command) collectCommand(name string, field reflect.StructField, fieldValue reflect.Value) error {
	if fieldValue.Kind() != reflect.Struct {
		return fmt.Errorf("xflags: command field %q must be a struct", field.Name)
	}
	if !fieldValue.CanAddr() {
		return fmt.Errorf("xflags: command field %q is not addressable", field.Name)
	}

	sub, err := c.AddCommand(name, fieldValue.Addr().Interface())
	if err != nil {
		return err
	}
	if desc := field.Tag.Get("description"); desc != "" {
		sub.description = desc
	}
	return nil
}

// collectNamespace descends into a nested struct, prefixing its options.
func (c *Command) collectNamespace(ns, prefix string, fieldValue reflect.Value, g *group) error {
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			if !fieldValue.CanSet() {
				return fmt.Errorf("xflags: namespace %q pointer is not settable", ns)
			}
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		}
		fieldValue = fieldValue.Elem()
	}
	if fieldValue.Kind() != reflect.Struct {
		return fmt.Errorf("xflags: namespace %q must be a struct", ns)
	}

	newPrefix := ns
	if prefix != "" {
		newPrefix = fmt.Sprintf("%s.%s", prefix, ns)
	}
	return c.collectStruct(fieldValue, newPrefix, g)
}

// collectOption builds an Option from a struct field and registers it.
func (c *Command) collectOption(field reflect.StructField, fieldValue reflect.Value, prefix string, g *group) error {
	opt := &Option{
		Short:       field.Tag.Get("short"),
		Long:        field.Tag.Get("long"),
		Description: field.Tag.Get("description"),
		Default:     field.Tag.Get("default"),
		Env:         field.Tag.Get("env"),
		Required:    strings.EqualFold(field.Tag.Get("required"), "true"),
		ValueName:   field.Tag.Get("value-name"),
		Hidden:      strings.EqualFold(field.Tag.Get("hidden"), "true"),
		optional:    strings.EqualFold(field.Tag.Get("optional"), "true"),
	}

	_, opt.defaultSet = field.Tag.Lookup("default")
	if opt.Long != "" && prefix != "" {
		opt.Long = fmt.Sprintf("%s.%s", prefix, opt.Long)
	}

	if ov, ok := field.Tag.Lookup("optional-value"); ok {
		opt.optional = true
		opt.optionalValue = ov
		opt.optionalValSet = true
	}

	if baseTag, ok := field.Tag.Lookup("base"); ok {
		base, err := strconv.Atoi(baseTag)
		if err != nil || (base != 0 && (base < 2 || base > 36)) {
			return fmt.Errorf("xflags: option %q: invalid base %q", optionName(opt), baseTag)
		}
		opt.base = base
		opt.baseSet = true
	}

	if choiceTag, ok := field.Tag.Lookup("choice"); ok {
		policy, err := parseChoice(choiceTag)
		if err != nil {
			return fmt.Errorf("xflags: option %q: %w", optionName(opt), err)
		}
		opt.choice = policy
		opt.choiceSet = true
	}

	// A func-typed field is a callback option rather than a value destination.
	if fieldValue.Kind() == reflect.Func {
		if err := bindCallback(opt, field, fieldValue); err != nil {
			return err
		}
	} else {
		if !fieldValue.CanSet() {
			return fmt.Errorf("xflags: option %q field is not settable", optionName(opt))
		}
		opt.value = fieldValue
		opt.isBool = fieldValue.Kind() == reflect.Bool
		if !opt.choiceSet {
			opt.choice = defaultChoice(fieldValue)
		}
	}

	return c.registerOrMerge(opt, g)
}

// registerOrMerge registers an option, or merges a callback into an existing
// value option that shares the same long name (store-and-validate pattern).
func (c *Command) registerOrMerge(opt *Option, g *group) error {
	if opt.Long != "" {
		if existing, ok := c.byLong[opt.Long]; ok {
			return c.mergeOption(existing, opt)
		}
	}
	return c.addOption(opt, g)
}

// mergeOption folds a second declaration (typically a func field) into an
// existing option that shares the same long name, combining the value
// destination, callback, and metadata from both declarations.
func (c *Command) mergeOption(existing, incoming *Option) error {
	before := *existing
	shorts := make(map[string]*Option, len(c.byShort))
	for k, v := range c.byShort {
		shorts[k] = v
	}
	err := c.mergeOptionUnchecked(existing, incoming)
	if err == nil {
		err = existing.checkBinding()
	}
	if err != nil {
		*existing = before
		c.byShort = shorts
	}
	return err
}

func (c *Command) mergeOptionUnchecked(existing, incoming *Option) error {
	switch {
	case incoming.callback.IsValid():
		if existing.callback.IsValid() {
			return fmt.Errorf("xflags: option %q already has a callback", optionName(existing))
		}
		existing.callback = incoming.callback
	case incoming.value.IsValid():
		if existing.value.IsValid() {
			return fmt.Errorf("xflags: duplicate long option %q", existing.Long)
		}
		existing.value = incoming.value
		existing.isBool = incoming.isBool
		if incoming.choiceSet {
			existing.choice = incoming.choice
			existing.choiceSet = true
		} else if !existing.choiceSet {
			existing.choice = defaultChoice(incoming.value)
		}
	default:
		return fmt.Errorf("xflags: duplicate long option %q", existing.Long)
	}

	return c.mergeMetadata(existing, incoming)
}

// mergeMetadata copies option attributes declared on the incoming field into an
// existing option, preserving values the existing declaration already set.
func (c *Command) mergeMetadata(existing, incoming *Option) error {
	if incoming.Short != "" {
		if !validName(incoming.Short) || len([]rune(incoming.Short)) != 1 {
			return fmt.Errorf("xflags: invalid short option %q", incoming.Short)
		}
		if existing.Short == "" {
			if _, dup := c.byShort[incoming.Short]; dup {
				return fmt.Errorf("xflags: duplicate short option %q", incoming.Short)
			}
			existing.Short = incoming.Short
			c.byShort[incoming.Short] = existing
		} else if existing.Short != incoming.Short {
			return fmt.Errorf("xflags: option %q declared with conflicting short names %q and %q",
				existing.Long, existing.Short, incoming.Short)
		}
	}
	if existing.Description == "" {
		existing.Description = incoming.Description
	}
	if !existing.defaultSet {
		existing.Default = incoming.Default
		existing.defaultSet = incoming.defaultSet
	}
	if existing.Env == "" {
		existing.Env = incoming.Env
	}
	if incoming.Required {
		existing.Required = true
	}
	if !existing.baseSet && incoming.baseSet {
		existing.base, existing.baseSet = incoming.base, true
	}
	if !existing.choiceSet && incoming.choiceSet {
		existing.choice, existing.choiceSet = incoming.choice, true
	}
	if !existing.optionalValSet && incoming.optionalValSet {
		existing.optionalValue, existing.optionalValSet = incoming.optionalValue, true
	}
	existing.optional = existing.optional || incoming.optional
	existing.Hidden = existing.Hidden || incoming.Hidden
	if existing.ValueName == "" {
		existing.ValueName = incoming.ValueName
	}
	return nil
}

// bindCallback validates and stores a func-typed option field. Accepted
// signatures are func(T) error and func() error.
func bindCallback(opt *Option, field reflect.StructField, fieldValue reflect.Value) error {
	ft := fieldValue.Type()
	if ft.NumOut() != 1 || ft.Out(0) != reflect.TypeFor[error]() {
		return fmt.Errorf("xflags: callback option %q must return error", optionName(opt))
	}
	if ft.NumIn() > 1 || ft.IsVariadic() {
		return fmt.Errorf("xflags: callback option %q must take zero or one argument", optionName(opt))
	}
	opt.callback = fieldValue
	opt.isBool = ft.NumIn() == 0
	_ = field
	return nil
}

// parseChoice converts a choice tag value to a policy.
func parseChoice(s string) (choicePolicy, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "append":
		return choiceAppend, nil
	case "count":
		return choiceCount, nil
	case "last":
		return choiceLast, nil
	case "merge":
		return choiceMerge, nil
	default:
		return choiceLast, fmt.Errorf("unknown choice %q (want append, count, last, or merge)", s)
	}
}

// defaultChoice picks a policy based on the destination type.
func defaultChoice(v reflect.Value) choicePolicy {
	switch v.Kind() {
	case reflect.Slice:
		return choiceAppend
	case reflect.Map:
		return choiceMerge
	default:
		return choiceLast
	}
}

// optionName returns a human-readable name for error messages.
func optionName(opt *Option) string {
	if opt.Long != "" {
		return fmt.Sprintf("--%s", opt.Long)
	}
	if opt.Short != "" {
		return fmt.Sprintf("-%s", opt.Short)
	}
	return "?"
}
