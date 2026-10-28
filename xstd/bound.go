package xstd

// BoundKind mirrors the variants of Rust's std::ops::Bound: a bound on a range
// is either included, excluded, or unbounded.
type BoundKind int

const (
	// BoundUnbounded is an infinite endpoint (Rust Bound::Unbounded).
	BoundUnbounded BoundKind = 0
	// BoundIncluded includes the endpoint value (Rust Bound::Included).
	BoundIncluded BoundKind = 1
	// BoundExcluded excludes the endpoint value (Rust Bound::Excluded).
	BoundExcluded BoundKind = 2
)

// Bound is one endpoint of a range query, mirroring Rust's Bound<T>. The Value
// field is ignored when Kind is BoundUnbounded.
type Bound[T any] struct {
	Kind  BoundKind
	Value T
}

// Included builds an inclusive bound (Rust Bound::Included).
func Included[T any](v T) Bound[T] {
	return Bound[T]{Kind: BoundIncluded, Value: v}
}

// Excluded builds an exclusive bound (Rust Bound::Excluded).
func Excluded[T any](v T) Bound[T] {
	return Bound[T]{Kind: BoundExcluded, Value: v}
}

// Unbounded builds an open-ended bound (Rust Bound::Unbounded).
func Unbounded[T any]() Bound[T] {
	return Bound[T]{Kind: BoundUnbounded}
}
