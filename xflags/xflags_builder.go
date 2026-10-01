package xflags

import (
	"fmt"
	"reflect"
	"strconv"
)

// builderGroup returns the command's default builder group, creating it on
// first use. Options registered through the builder API land here.
func (c *Command) builderGroup() *group {
	for _, g := range c.groups {
		if g.title == "Options" {
			return g
		}
	}
	g := &group{title: "Options"}
	c.groups = append(c.groups, g)
	return g
}

// optionSpec describes a builder-registered value option.
type optionSpec struct {
	ptr          any
	short        string
	long         string
	def          string
	description  string
	choice       choicePolicy
	typedDefault any
}

// register wires a destination pointer into a new option and adds it.
func (c *Command) register(spec optionSpec) error {
	v := reflect.ValueOf(spec.ptr)
	if v.Kind() != reflect.Pointer || v.IsNil() {
		return fmt.Errorf("xflags: option %q requires a non-nil pointer destination", spec.long)
	}
	elem := v.Elem()

	opt := &Option{
		Short:       spec.short,
		Long:        spec.long,
		Default:     spec.def,
		Description: spec.description,
		value:       elem,
		isBool:      elem.Kind() == reflect.Bool,
		choice:      spec.choice,
		choiceSet:   true,
	}
	if spec.typedDefault != nil {
		opt.typedDefault = reflect.ValueOf(spec.typedDefault)
		opt.defaultSet = true
	}
	return c.addBuilderOption(opt)
}

func (c *Command) addBuilderOption(opt *Option) error {
	if err := c.addOption(opt, nil); err != nil {
		return err
	}
	g := c.builderGroup()
	g.options = append(g.options, opt)
	return nil
}

// BoolVar registers a boolean switch bound to ptr.
func (c *Command) BoolVar(ptr *bool, long, short string, def bool, description string) error {
	defStr := ""
	if def {
		defStr = "true"
	}
	return c.register(optionSpec{
		ptr:          ptr,
		short:        short,
		long:         long,
		def:          defStr,
		description:  description,
		choice:       choiceLast,
		typedDefault: def,
	})
}

// StringVar registers a string option bound to ptr.
func (c *Command) StringVar(ptr *string, long, short, def, description string) error {
	return c.register(optionSpec{
		ptr:          ptr,
		short:        short,
		long:         long,
		def:          def,
		description:  description,
		choice:       choiceLast,
		typedDefault: def,
	})
}

// IntVar registers an integer option bound to ptr.
func (c *Command) IntVar(ptr *int, long, short string, def int, description string) error {
	return c.register(optionSpec{
		ptr:          ptr,
		short:        short,
		long:         long,
		def:          strconv.Itoa(def),
		description:  description,
		choice:       choiceLast,
		typedDefault: def,
	})
}

// CountVar registers a counting option that increments ptr on each occurrence.
func (c *Command) CountVar(ptr *int, long, short, description string) error {
	return c.register(optionSpec{ptr: ptr, short: short, long: long, description: description, choice: choiceCount})
}

// StringSliceVar registers a slice option that appends each occurrence to ptr.
func (c *Command) StringSliceVar(ptr *[]string, long, short, description string) error {
	return c.register(optionSpec{ptr: ptr, short: short, long: long, description: description, choice: choiceAppend})
}

// StringMapVar registers a map option that merges key=value pairs into ptr.
func (c *Command) StringMapVar(ptr *map[string]string, long, short, description string) error {
	return c.register(optionSpec{ptr: ptr, short: short, long: long, description: description, choice: choiceMerge})
}

// Func registers a callback option that invokes fn for each occurrence.
func (c *Command) Func(long, short, description string, fn func(string) error) error {
	opt := &Option{
		Short:       short,
		Long:        long,
		Description: description,
		callback:    reflect.ValueOf(fn),
		choice:      choiceLast,
		choiceSet:   true,
	}
	return c.addBuilderOption(opt)
}

// SetHidden hides the named option from the help message.
func (c *Command) SetHidden(long string) error {
	opt, ok := c.byLong[long]
	if !ok {
		return fmt.Errorf("xflags: no option named %q", long)
	}
	opt.Hidden = true
	return nil
}

// SetValueName sets the help placeholder for the named valued option.
func (c *Command) SetValueName(long, name string) error {
	opt, ok := c.byLong[long]
	if !ok {
		return fmt.Errorf("xflags: no option named %q", long)
	}
	opt.ValueName = name
	return nil
}

// SetBase sets the integer radix used when parsing the named option.
func (c *Command) SetBase(long string, base int) error {
	opt, ok := c.byLong[long]
	if !ok {
		return fmt.Errorf("xflags: no option named %q", long)
	}
	if base != 0 && (base < 2 || base > 36) {
		return fmt.Errorf("xflags: invalid base %d", base)
	}
	opt.base = base
	opt.baseSet = true
	return nil
}

// SetOptionalValue marks the named option's argument optional, using value when
// the option appears on the command line without one.
func (c *Command) SetOptionalValue(long, value string) error {
	opt, ok := c.byLong[long]
	if !ok {
		return fmt.Errorf("xflags: no option named %q", long)
	}
	opt.optional = true
	opt.optionalValue = value
	opt.optionalValSet = true
	return nil
}
