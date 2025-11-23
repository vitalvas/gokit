package wirefilter

import "net"

// ExecutionContext holds the runtime values for fields that are evaluated during filter execution.
type ExecutionContext struct {
	fields map[string]Value
}

// NewExecutionContext creates a new empty execution context.
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		fields: make(map[string]Value),
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

// GetField retrieves a field value from the execution context.
// Returns the value and true if found, or nil and false if not found.
func (ctx *ExecutionContext) GetField(name string) (Value, bool) {
	val, ok := ctx.fields[name]
	return val, ok
}
