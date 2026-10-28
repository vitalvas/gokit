package xstd

import (
	"cmp"
	"iter"
)

// BTreeSet is an ordered set backed by a B-tree. Iteration and range queries
// visit elements in ascending order. It is not safe for concurrent use. A
// BTreeSet is a BTreeMap with no values, matching Rust's definition.
type BTreeSet[T cmp.Ordered] struct {
	m *BTreeMap[T, struct{}]
}

// NewBTreeSet creates an empty ordered set.
func NewBTreeSet[T cmp.Ordered]() *BTreeSet[T] {
	return &BTreeSet[T]{m: NewBTreeMap[T, struct{}]()}
}

// Len returns the number of elements in the set.
func (s *BTreeSet[T]) Len() int {
	return s.m.Len()
}

// IsEmpty reports whether the set contains no elements.
func (s *BTreeSet[T]) IsEmpty() bool {
	return s.m.IsEmpty()
}

// Clear removes all elements from the set.
func (s *BTreeSet[T]) Clear() {
	s.m.Clear()
}

// Contains reports whether value is in the set.
func (s *BTreeSet[T]) Contains(value T) bool {
	return s.m.ContainsKey(value)
}

// Get returns the stored element equal to value and whether it was present,
// mirroring Rust's BTreeSet::get.
func (s *BTreeSet[T]) Get(value T) (T, bool) {
	k, _, ok := s.m.GetKeyValue(value)
	return k, ok
}

// First returns the smallest element and whether the set is non-empty
// (Rust BTreeSet::first).
func (s *BTreeSet[T]) First() (T, bool) {
	k, _, ok := s.m.FirstKeyValue()
	return k, ok
}

// Last returns the largest element and whether the set is non-empty
// (Rust BTreeSet::last).
func (s *BTreeSet[T]) Last() (T, bool) {
	k, _, ok := s.m.LastKeyValue()
	return k, ok
}

// Insert adds value and reports whether it was newly added (false if already
// present), matching Rust's BTreeSet::insert.
func (s *BTreeSet[T]) Insert(value T) bool {
	_, existed := s.m.Insert(value, struct{}{})
	return !existed
}

// Replace inserts value and returns the element it replaced, if any, mirroring
// Rust's BTreeSet::replace. For cmp.Ordered elements the replaced element is
// equal to value, so the returned value matters only for element identity.
func (s *BTreeSet[T]) Replace(value T) (T, bool) {
	old, existed := s.Take(value)
	s.m.Insert(value, struct{}{})
	return old, existed
}

// Remove removes value and reports whether it was present (Rust
// BTreeSet::remove).
func (s *BTreeSet[T]) Remove(value T) bool {
	_, existed := s.m.Remove(value)
	return existed
}

// Take removes value and returns the stored element, mirroring Rust's
// BTreeSet::take.
func (s *BTreeSet[T]) Take(value T) (T, bool) {
	k, _, ok := s.m.RemoveEntry(value)
	return k, ok
}

// PopFirst removes and returns the smallest element (Rust BTreeSet::pop_first).
func (s *BTreeSet[T]) PopFirst() (T, bool) {
	k, _, ok := s.m.PopFirst()
	return k, ok
}

// PopLast removes and returns the largest element (Rust BTreeSet::pop_last).
func (s *BTreeSet[T]) PopLast() (T, bool) {
	k, _, ok := s.m.PopLast()
	return k, ok
}

// GetOrInsert returns the stored element equal to value, inserting value when
// absent, mirroring Rust's BTreeSet::get_or_insert.
func (s *BTreeSet[T]) GetOrInsert(value T) T {
	if k, ok := s.Get(value); ok {
		return k
	}
	s.Insert(value)
	return value
}

// GetOrInsertWith returns the stored element equal to value, inserting the
// result of f when absent, mirroring Rust's BTreeSet::get_or_insert_with.
func (s *BTreeSet[T]) GetOrInsertWith(value T, f func(T) T) T {
	if k, ok := s.Get(value); ok {
		return k
	}
	v := f(value)
	s.Insert(v)
	return v
}

// Retain keeps only the elements for which f returns true (Rust
// BTreeSet::retain).
func (s *BTreeSet[T]) Retain(f func(T) bool) {
	s.m.Retain(func(k T, _ struct{}) bool { return f(k) })
}

// Iter iterates over every element in ascending order (Rust BTreeSet::iter).
func (s *BTreeSet[T]) Iter() iter.Seq[T] {
	return s.m.Keys()
}

// Range iterates over elements within [lower, upper] as described by the two
// bounds, in ascending order (Rust BTreeSet::range).
func (s *BTreeSet[T]) Range(lower, upper Bound[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for k := range s.m.Range(lower, upper) {
			if !yield(k) {
				return
			}
		}
	}
}

// ExtractIf removes and returns every element for which pred returns true,
// in ascending order (Rust BTreeSet::extract_if).
func (s *BTreeSet[T]) ExtractIf(pred func(T) bool) iter.Seq[T] {
	seq := s.m.ExtractIf(func(k T, _ struct{}) bool { return pred(k) })
	return func(yield func(T) bool) {
		for k := range seq {
			if !yield(k) {
				return
			}
		}
	}
}

// Append moves all elements from other into s, leaving other empty (Rust
// BTreeSet::append). Appending a set to itself has no effect.
func (s *BTreeSet[T]) Append(other *BTreeSet[T]) {
	s.m.Append(other.m)
}

// SplitOff removes every element >= value from s and returns them as a new
// set (Rust BTreeSet::split_off).
func (s *BTreeSet[T]) SplitOff(value T) *BTreeSet[T] {
	return &BTreeSet[T]{m: s.m.SplitOff(value)}
}

// IsDisjoint reports whether s and other share no elements (Rust
// BTreeSet::is_disjoint).
func (s *BTreeSet[T]) IsDisjoint(other *BTreeSet[T]) bool {
	small, large := s, other
	if large.Len() < small.Len() {
		small, large = large, small
	}
	for v := range small.Iter() {
		if large.Contains(v) {
			return false
		}
	}
	return true
}

// IsSubset reports whether every element of s is in other (Rust
// BTreeSet::is_subset).
func (s *BTreeSet[T]) IsSubset(other *BTreeSet[T]) bool {
	if s.Len() > other.Len() {
		return false
	}
	for v := range s.Iter() {
		if !other.Contains(v) {
			return false
		}
	}
	return true
}

// IsSuperset reports whether every element of other is in s (Rust
// BTreeSet::is_superset).
func (s *BTreeSet[T]) IsSuperset(other *BTreeSet[T]) bool {
	return other.IsSubset(s)
}

// Union iterates over every element in either set, ascending and deduplicated
// (Rust BTreeSet::union).
func (s *BTreeSet[T]) Union(other *BTreeSet[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		next, stop := iter.Pull(other.Iter())
		defer stop()
		ov, ok := next()
		for v := range s.Iter() {
			for ok && cmp.Less(ov, v) {
				if !yield(ov) {
					return
				}
				ov, ok = next()
			}
			if ok && cmp.Compare(ov, v) == 0 {
				ov, ok = next()
			}
			if !yield(v) {
				return
			}
		}
		for ok {
			if !yield(ov) {
				return
			}
			ov, ok = next()
		}
	}
}

// Intersection iterates over every element present in both sets, ascending
// (Rust BTreeSet::intersection).
func (s *BTreeSet[T]) Intersection(other *BTreeSet[T]) iter.Seq[T] {
	small, large := s, other
	if large.Len() < small.Len() {
		small, large = large, small
	}
	return func(yield func(T) bool) {
		for v := range small.Iter() {
			if large.Contains(v) && !yield(v) {
				return
			}
		}
	}
}

// Difference iterates over every element in s that is not in other, ascending
// (Rust BTreeSet::difference).
func (s *BTreeSet[T]) Difference(other *BTreeSet[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		for v := range s.Iter() {
			if !other.Contains(v) && !yield(v) {
				return
			}
		}
	}
}

// SymmetricDifference iterates over every element in exactly one of the sets,
// ascending (Rust BTreeSet::symmetric_difference).
func (s *BTreeSet[T]) SymmetricDifference(other *BTreeSet[T]) iter.Seq[T] {
	return func(yield func(T) bool) {
		next, stop := iter.Pull(other.Iter())
		defer stop()
		ov, ok := next()
		for v := range s.Iter() {
			for ok && cmp.Less(ov, v) {
				if !yield(ov) {
					return
				}
				ov, ok = next()
			}
			if ok && cmp.Compare(ov, v) == 0 {
				ov, ok = next()
				continue
			}
			if !yield(v) {
				return
			}
		}
		for ok {
			if !yield(ov) {
				return
			}
			ov, ok = next()
		}
	}
}
