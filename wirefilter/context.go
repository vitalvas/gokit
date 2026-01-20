package wirefilter

import "net"

// ExecutionContext holds the runtime values for fields that are evaluated during filter execution.
type ExecutionContext struct {
	fields map[string]Value
	lists  map[string]ArrayValue
}

// NewExecutionContext creates a new empty execution context.
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		fields: make(map[string]Value),
		lists:  make(map[string]ArrayValue),
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
// Returns the context to allow method chaining.
func (ctx *ExecutionContext) SetIPList(name string, ips []string) *ExecutionContext {
	arr := make(ArrayValue, 0, len(ips))
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip != nil {
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
