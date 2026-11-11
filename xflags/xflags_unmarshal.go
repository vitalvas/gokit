package xflags

import (
	"encoding"
	"reflect"
)

var textUnmarshalerType = reflect.TypeFor[encoding.TextUnmarshaler]()

// implementsTextUnmarshaler reports whether t, or a pointer to t, implements
// encoding.TextUnmarshaler. UnmarshalText has a pointer receiver, so a value
// type T satisfies the interface only through *T; this mirrors how
// encoding/json detects the interface.
func implementsTextUnmarshaler(t reflect.Type) bool {
	if t.Implements(textUnmarshalerType) {
		return true
	}
	if t.Kind() != reflect.Pointer {
		return reflect.PointerTo(t).Implements(textUnmarshalerType)
	}
	return false
}

// textUnmarshaler returns dst as an encoding.TextUnmarshaler when its type, or
// a pointer to it, implements the interface. A value type is addressed so its
// pointer-receiver UnmarshalText can mutate it in place.
func textUnmarshaler(dst reflect.Value) (encoding.TextUnmarshaler, bool) {
	if u, ok := dst.Interface().(encoding.TextUnmarshaler); ok {
		return u, true
	}
	if dst.CanAddr() {
		if u, ok := dst.Addr().Interface().(encoding.TextUnmarshaler); ok {
			return u, true
		}
	}
	return nil, false
}
