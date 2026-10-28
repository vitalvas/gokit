package xstd

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setOf(vs ...int) *BTreeSet[int] {
	s := NewBTreeSet[int]()
	for _, v := range vs {
		s.Insert(v)
	}
	return s
}

func TestBTreeSetBasic(t *testing.T) {
	t.Run("insert reports newness", func(t *testing.T) {
		s := NewBTreeSet[int]()
		assert.True(t, s.Insert(1))
		assert.False(t, s.Insert(1))
		assert.Equal(t, 1, s.Len())
		assert.False(t, s.IsEmpty())
		assert.True(t, s.Contains(1))
		assert.False(t, s.Contains(2))
	})

	t.Run("empty", func(t *testing.T) {
		s := NewBTreeSet[int]()
		assert.True(t, s.IsEmpty())
		assert.Equal(t, 0, s.Len())
	})

	t.Run("clear", func(t *testing.T) {
		s := setOf(1, 2, 3)
		s.Clear()
		assert.True(t, s.IsEmpty())
	})

	t.Run("get", func(t *testing.T) {
		s := setOf(5)
		v, ok := s.Get(5)
		assert.True(t, ok)
		assert.Equal(t, 5, v)
		_, ok = s.Get(6)
		assert.False(t, ok)
	})

	t.Run("replace", func(t *testing.T) {
		s := setOf(5)
		old, existed := s.Replace(5)
		assert.True(t, existed)
		assert.Equal(t, 5, old)

		_, existed = s.Replace(9)
		assert.False(t, existed)
		assert.True(t, s.Contains(9))
	})
}

func TestBTreeSetFirstLastPop(t *testing.T) {
	s := setOf(3, 1, 4, 1, 5, 9, 2, 6)

	v, ok := s.First()
	assert.True(t, ok)
	assert.Equal(t, 1, v)

	v, ok = s.Last()
	assert.True(t, ok)
	assert.Equal(t, 9, v)

	v, ok = s.PopFirst()
	assert.True(t, ok)
	assert.Equal(t, 1, v)
	assert.False(t, s.Contains(1))

	v, ok = s.PopLast()
	assert.True(t, ok)
	assert.Equal(t, 9, v)
	assert.False(t, s.Contains(9))

	empty := NewBTreeSet[int]()
	_, ok = empty.First()
	assert.False(t, ok)
	_, ok = empty.Last()
	assert.False(t, ok)
	_, ok = empty.PopFirst()
	assert.False(t, ok)
	_, ok = empty.PopLast()
	assert.False(t, ok)
}

func TestBTreeSetRemoveTake(t *testing.T) {
	s := setOf(1, 2, 3)

	assert.True(t, s.Remove(2))
	assert.False(t, s.Remove(2))
	assert.Equal(t, 2, s.Len())

	v, ok := s.Take(1)
	assert.True(t, ok)
	assert.Equal(t, 1, v)
	_, ok = s.Take(1)
	assert.False(t, ok)
}

func TestBTreeSetGetOrInsert(t *testing.T) {
	s := NewBTreeSet[int]()
	assert.Equal(t, 5, s.GetOrInsert(5))
	assert.True(t, s.Contains(5))
	assert.Equal(t, 5, s.GetOrInsert(5))
	assert.Equal(t, 1, s.Len())

	calls := 0
	f := func(v int) int { calls++; return v }
	assert.Equal(t, 7, s.GetOrInsertWith(7, f))
	assert.Equal(t, 1, calls)
	assert.Equal(t, 7, s.GetOrInsertWith(7, f))
	assert.Equal(t, 1, calls, "f not called when present")
}

func TestBTreeSetRetain(t *testing.T) {
	s := setOf(1, 2, 3, 4, 5, 6)
	s.Retain(func(v int) bool { return v%2 == 0 })
	assert.Equal(t, []int{2, 4, 6}, collectK(s.Iter()))
}

func TestBTreeSetIterRange(t *testing.T) {
	s := NewBTreeSet[int]()
	for _, v := range []int{5, 1, 9, 3, 7, 2, 8, 4, 6, 0} {
		s.Insert(v)
	}
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, collectK(s.Iter()))
	assert.Equal(t, []int{3, 4, 5, 6}, collectK(s.Range(Included(3), Included(6))))
	assert.Equal(t, []int{3, 4, 5}, collectK(s.Range(Included(3), Excluded(6))))
}

func TestBTreeSetExtractIf(t *testing.T) {
	s := setOf(1, 2, 3, 4, 5)
	out := collectK(s.ExtractIf(func(v int) bool { return v > 3 }))
	assert.Equal(t, []int{4, 5}, out)
	assert.Equal(t, []int{1, 2, 3}, collectK(s.Iter()))
}

func TestBTreeSetAppendSplitOff(t *testing.T) {
	a := setOf(1, 2, 3)
	b := setOf(3, 4, 5)
	a.Append(b)
	assert.Equal(t, 0, b.Len())
	assert.Equal(t, []int{1, 2, 3, 4, 5}, collectK(a.Iter()))

	hi := a.SplitOff(3)
	assert.Equal(t, []int{1, 2}, collectK(a.Iter()))
	assert.Equal(t, []int{3, 4, 5}, collectK(hi.Iter()))
}

func TestBTreeSetPredicates(t *testing.T) {
	a := setOf(1, 2, 3)
	b := setOf(1, 2, 3, 4, 5)
	c := setOf(10, 20)

	assert.True(t, a.IsSubset(b))
	assert.False(t, b.IsSubset(a))
	assert.True(t, b.IsSuperset(a))
	assert.False(t, a.IsSuperset(b))

	assert.True(t, a.IsDisjoint(c))
	assert.False(t, a.IsDisjoint(b))

	// subset check short-circuits on size
	assert.False(t, b.IsSubset(a))
}

func TestBTreeSetOperations(t *testing.T) {
	a := setOf(1, 2, 3, 4)
	b := setOf(3, 4, 5, 6)

	assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, collectK(a.Union(b)))
	assert.Equal(t, []int{3, 4}, collectK(a.Intersection(b)))
	assert.Equal(t, []int{1, 2}, collectK(a.Difference(b)))
	assert.Equal(t, []int{1, 2, 5, 6}, collectK(a.SymmetricDifference(b)))

	t.Run("disjoint union", func(t *testing.T) {
		x := setOf(1, 3, 5)
		y := setOf(2, 4, 6)
		assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, collectK(x.Union(y)))
		assert.Empty(t, collectK(x.Intersection(y)))
		assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, collectK(x.SymmetricDifference(y)))
	})

	t.Run("empty operands", func(t *testing.T) {
		empty := NewBTreeSet[int]()
		full := setOf(1, 2)
		assert.Equal(t, []int{1, 2}, collectK(empty.Union(full)))
		assert.Equal(t, []int{1, 2}, collectK(full.Union(empty)))
		assert.Empty(t, collectK(empty.Intersection(full)))
		assert.Equal(t, []int{1, 2}, collectK(full.Difference(empty)))
		assert.Equal(t, []int{1, 2}, collectK(full.SymmetricDifference(empty)))
	})
}

func TestBTreeSetOperationsEarlyStop(t *testing.T) {
	a := setOf(1, 2, 3, 4, 5, 6)
	b := setOf(4, 5, 6, 7, 8, 9)

	stopAt := func(seq func(func(int) bool), at int) []int {
		var out []int
		for v := range seq {
			out = append(out, v)
			if v == at {
				break
			}
		}
		return out
	}
	assert.Equal(t, []int{1, 2}, stopAt(a.Union(b), 2))
	assert.Equal(t, []int{4}, stopAt(a.Intersection(b), 4))
	assert.Equal(t, []int{1, 2}, stopAt(a.Difference(b), 2))
	assert.Equal(t, []int{1, 2}, stopAt(a.SymmetricDifference(b), 2))
	// stop inside the drain-the-remainder tail of symmetric difference
	assert.Equal(t, []int{1, 2, 3, 7}, stopAt(a.SymmetricDifference(b), 7))
	// stop inside the drain-the-remainder tail of union
	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7}, stopAt(a.Union(b), 7))
}

func TestBTreeSetIterEarlyStop(t *testing.T) {
	s := setOf(0, 1, 2, 3, 4, 5, 6, 7, 8, 9)

	stopAt := func(seq func(func(int) bool), at int) []int {
		var out []int
		for v := range seq {
			out = append(out, v)
			if v == at {
				break
			}
		}
		return out
	}

	assert.Equal(t, []int{0, 1}, stopAt(s.Iter(), 1))
	assert.Equal(t, []int{2, 3}, stopAt(s.Range(Included(2), Included(8)), 3))

	other := setOf(3, 4, 5)
	assert.Equal(t, []int{3}, stopAt(s.Intersection(other), 3))

	// ExtractIf removes lazily, so early stop leaves unyielded matches.
	drain := setOf(0, 1, 2, 3, 4, 5)
	assert.Equal(t, []int{0, 1}, stopAt(drain.ExtractIf(func(v int) bool { return v < 5 }), 1))
	assert.True(t, drain.Contains(2), "unyielded match stays")
	assert.False(t, drain.Contains(0))
}

func TestBTreeSetLargeSorted(t *testing.T) {
	const n int = 4000
	s := NewBTreeSet[int]()
	for i := 0; i < n; i++ {
		s.Insert((i * 2654435761) % n)
	}
	assert.Equal(t, n, s.Len())
	out := collectK(s.Iter())
	assert.True(t, slices.IsSorted(out))
	assert.Len(t, out, n)
	require.Equal(t, 0, out[0])
	require.Equal(t, n-1, out[n-1])
}

func BenchmarkBTreeSetInsert(b *testing.B) {
	s := NewBTreeSet[int]()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.Insert(i % 100000)
	}
}

func FuzzBTreeSet(f *testing.F) {
	f.Add([]byte{3, 1, 4, 1, 5, 9, 2, 6})
	f.Fuzz(func(t *testing.T, data []byte) {
		s := NewBTreeSet[byte]()
		ref := make(map[byte]struct{})
		for i, b := range data {
			if i%3 == 2 {
				_, want := ref[b]
				assert.Equal(t, want, s.Remove(b))
				delete(ref, b)
			} else {
				_, exists := ref[b]
				assert.Equal(t, !exists, s.Insert(b))
				ref[b] = struct{}{}
			}
			require.Equal(t, len(ref), s.Len())
		}
		out := collectK(s.Iter())
		require.True(t, slices.IsSorted(out))
		require.Equal(t, len(ref), len(out))
	})
}

func FuzzBTreeSetOps(f *testing.F) {
	f.Add([]byte{1, 2, 3}, []byte{2, 3, 4})
	f.Fuzz(func(t *testing.T, ad, bd []byte) {
		a := NewBTreeSet[byte]()
		b := NewBTreeSet[byte]()
		ra := make(map[byte]struct{})
		rb := make(map[byte]struct{})
		for _, x := range ad {
			a.Insert(x)
			ra[x] = struct{}{}
		}
		for _, x := range bd {
			b.Insert(x)
			rb[x] = struct{}{}
		}

		union := collectK(a.Union(b))
		require.True(t, slices.IsSorted(union))
		wantUnion := len(ra)
		for k := range rb {
			if _, ok := ra[k]; !ok {
				wantUnion++
			}
		}
		require.Len(t, union, wantUnion)

		inter := collectK(a.Intersection(b))
		require.True(t, slices.IsSorted(inter))
		for _, k := range inter {
			_, inA := ra[k]
			_, inB := rb[k]
			require.True(t, inA && inB)
		}

		diff := collectK(a.Difference(b))
		for _, k := range diff {
			_, inB := rb[k]
			require.False(t, inB)
		}

		sym := collectK(a.SymmetricDifference(b))
		require.True(t, slices.IsSorted(sym))
		require.Len(t, sym, len(union)-len(inter))
	})
}
