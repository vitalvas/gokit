package wirefilter

import "net"

// FuncHandler is the type for user-defined function handlers.
type FuncHandler func(args []Value) (Value, error)

// ExecutionContext holds the runtime values for fields that are evaluated during filter execution.
type ExecutionContext struct {
	fields map[string]Value
	lists  map[string]ArrayValue
	tables map[string]MapValue
	funcs  map[string]FuncHandler
}

// NewExecutionContext creates a new empty execution context.
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		fields: make(map[string]Value),
		lists:  make(map[string]ArrayValue),
		tables: make(map[string]MapValue),
	}
}

// SetField sets a field value in the execution context.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetField(name string, value Value) *ExecutionContext {
	ctx.fields[name] = value
	return ctx
}

// SetStringField sets a string field value in the execution context.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetStringField(name string, value string) *ExecutionContext {
	ctx.fields[name] = StringValue(value)
	return ctx
}

// SetIntField sets an integer field value in the execution context.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetIntField(name string, value int64) *ExecutionContext {
	ctx.fields[name] = IntValue(value)
	return ctx
}

// SetFloatField sets a floating-point field value in the execution context.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetFloatField(name string, value float64) *ExecutionContext {
	ctx.fields[name] = FloatValue(value)
	return ctx
}

// SetBoolField sets a boolean field value in the execution context.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetBoolField(name string, value bool) *ExecutionContext {
	ctx.fields[name] = BoolValue(value)
	return ctx
}

// SetIPField sets an IP address field value in the execution context.
// The value string will be parsed as an IP address.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetIPField(name string, value string) *ExecutionContext {
	ip := net.ParseIP(value)
	if ip != nil {
		ctx.fields[name] = IPValue{IP: ip}
	}
	return ctx
}

// SetBytesField sets a bytes field value in the execution context.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetBytesField(name string, value []byte) *ExecutionContext {
	ctx.fields[name] = BytesValue(value)
	return ctx
}

// SetMapField sets a map field value in the execution context.
// Accepts map[string]string and converts values to StringValue.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetMapField(name string, value map[string]string) *ExecutionContext {
	m := make(MapValue, len(value))
	for k, v := range value {
		m[k] = StringValue(v)
	}
	ctx.fields[name] = m
	return ctx
}

// SetMapFieldValues sets a map field with Value types in the execution context.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetMapFieldValues(name string, value map[string]Value) *ExecutionContext {
	ctx.fields[name] = MapValue(value)
	return ctx
}

// SetMapArrayField sets a map field where each key maps to an array of Values.
// This supports any value types in the arrays (strings, ints, floats, IPs, CIDRs, etc.).
// Useful for HTTP headers, ACL rules, and similar map[string][]T structures.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetMapArrayField(name string, value map[string][]Value) *ExecutionContext {
	m := make(MapValue, len(value))
	for k, values := range value {
		m[k] = ArrayValue(values)
	}
	ctx.fields[name] = m
	return ctx
}

// GetField retrieves a field value from the execution context.
// Returns the value and true if found, or nil and false if not found.
func (ctx *ExecutionContext) GetField(name string) (Value, bool) {
	val, ok := ctx.fields[name]
	return val, ok
}

// SetArrayField sets an array of string values as an ArrayValue field.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetArrayField(name string, values []string) *ExecutionContext {
	arr := make(ArrayValue, len(values))
	for i, v := range values {
		arr[i] = StringValue(v)
	}
	ctx.fields[name] = arr
	return ctx
}

// SetIntArrayField sets an array of integer values as an ArrayValue field.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetIntArrayField(name string, values []int64) *ExecutionContext {
	arr := make(ArrayValue, len(values))
	for i, v := range values {
		arr[i] = IntValue(v)
	}
	ctx.fields[name] = arr
	return ctx
}

// SetList sets a string list in the execution context.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetList(name string, values []string) *ExecutionContext {
	arr := make(ArrayValue, len(values))
	for i, v := range values {
		arr[i] = StringValue(v)
	}
	ctx.lists[name] = arr
	return ctx
}

// SetIPList sets an IP address list in the execution context.
// Values can be plain IPs (e.g., "10.0.0.1") or CIDR ranges (e.g., "10.0.0.0/8").
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetIPList(name string, values []string) *ExecutionContext {
	arr := make(ArrayValue, 0, len(values))
	for _, v := range values {
		if _, ipNet, err := net.ParseCIDR(v); err == nil {
			arr = append(arr, CIDRValue{IPNet: ipNet})
			continue
		}
		if ip := net.ParseIP(v); ip != nil {
			arr = append(arr, IPValue{IP: ip})
		}
	}
	ctx.lists[name] = arr
	return ctx
}

// GetList retrieves a list from the execution context.
// Returns the list and true if found, or nil and false if not found.
func (ctx *ExecutionContext) GetList(name string) (ArrayValue, bool) {
	val, ok := ctx.lists[name]
	return val, ok
}

// SetTable sets a lookup table with string values in the execution context.
// Tables are referenced in expressions with $table_name[field] syntax.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetTable(name string, data map[string]string) *ExecutionContext {
	m := make(MapValue, len(data))
	for k, v := range data {
		m[k] = StringValue(v)
	}
	ctx.tables[name] = m
	return ctx
}

// SetTableValues sets a lookup table with mixed value types.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetTableValues(name string, data map[string]Value) *ExecutionContext {
	ctx.tables[name] = MapValue(data)
	return ctx
}

// SetTableList sets a lookup table where each key maps to a string array.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetTableList(name string, data map[string][]string) *ExecutionContext {
	m := make(MapValue, len(data))
	for k, values := range data {
		arr := make(ArrayValue, len(values))
		for i, v := range values {
			arr[i] = StringValue(v)
		}
		m[k] = arr
	}
	ctx.tables[name] = m
	return ctx
}

// SetTableIPList sets a lookup table where each key maps to an IP/CIDR array.
// Values can be plain IPs or CIDR ranges.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetTableIPList(name string, data map[string][]string) *ExecutionContext {
	m := make(MapValue, len(data))
	for k, values := range data {
		arr := make(ArrayValue, 0, len(values))
		for _, v := range values {
			if _, ipNet, err := net.ParseCIDR(v); err == nil {
				arr = append(arr, CIDRValue{IPNet: ipNet})
				continue
			}
			if ip := net.ParseIP(v); ip != nil {
				arr = append(arr, IPValue{IP: ip})
			}
		}
		m[k] = arr
	}
	ctx.tables[name] = m
	return ctx
}

// GetTable retrieves a lookup table from the execution context.
// Returns the table and true if found, or nil and false if not found.
func (ctx *ExecutionContext) GetTable(name string) (MapValue, bool) {
	val, ok := ctx.tables[name]
	return val, ok
}

// SetFunc registers a user-defined function handler in the execution context.
// The handler will be called when the function is invoked in a filter expression.
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetFunc(name string, handler FuncHandler) *ExecutionContext {
	if ctx.funcs == nil {
		ctx.funcs = make(map[string]FuncHandler)
	}
	ctx.funcs[name] = handler
	return ctx
}

// GetFunc retrieves a user-defined function handler from the execution context.
// Returns the handler and true if found, or nil and false if not found.
func (ctx *ExecutionContext) GetFunc(name string) (FuncHandler, bool) {
	if ctx.funcs == nil {
		return nil, false
	}
	fn, ok := ctx.funcs[name]
	return fn, ok
}
