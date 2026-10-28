package xstd

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collectKV[K any, V any](seq func(func(K, V) bool)) ([]K, []V) {
	var ks []K
	var vs []V
	for k, v := range seq {
		ks = append(ks, k)
		vs = append(vs, v)
	}
	return ks, vs
}

func collectK[K any](seq func(func(K) bool)) []K {
	var ks []K
	for k := range seq {
		ks = append(ks, k)
	}
	return ks
}

func filledMap(n int) *BTreeMap[int, int] {
	m := NewBTreeMap[int, int]()
	for i := 0; i < n; i++ {
		m.Insert(i, i*10)
	}
	return m
}

func TestBTreeMapKeysValues(t *testing.T) {
	m := NewBTreeMap[int, int]()
	for _, k := range []int{5, 1, 3, 2, 4} {
		m.Insert(k, k*10)
	}
	assert.Equal(t, []int{1, 2, 3, 4, 5}, collectK(m.Keys()))
	assert.Equal(t, []int{10, 20, 30, 40, 50}, collectK(m.Values()))
}

func TestBTreeMapIntoKeysValues(t *testing.T) {
	m := filledMap(50)
	keys := m.IntoKeys()
	assert.True(t, slices.IsSorted(keys))
	assert.Len(t, keys, 50)
	assert.Equal(t, 0, m.Len(), "map drained")

	m = filledMap(50)
	vals := m.IntoValues()
	assert.Len(t, vals, 50)
	assert.Equal(t, 0, vals[0])
	assert.Equal(t, 490, vals[49])
	assert.Equal(t, 0, m.Len())
}

func TestBTreeMapRange(t *testing.T) {
	m := filledMap(100)

	t.Run("included to excluded", func(t *testing.T) {
		keys, _ := collectKV(m.Range(Included(10), Excluded(15)))
		assert.Equal(t, []int{10, 11, 12, 13, 14}, keys)
	})

	t.Run("included to included", func(t *testing.T) {
		keys, _ := collectKV(m.Range(Included(10), Included(13)))
		assert.Equal(t, []int{10, 11, 12, 13}, keys)
	})

	t.Run("excluded to excluded", func(t *testing.T) {
		keys, _ := collectKV(m.Range(Excluded(10), Excluded(13)))
		assert.Equal(t, []int{11, 12}, keys)
	})

	t.Run("unbounded lower", func(t *testing.T) {
		keys, _ := collectKV(m.Range(Unbounded[int](), Excluded(3)))
		assert.Equal(t, []int{0, 1, 2}, keys)
	})

	t.Run("unbounded upper", func(t *testing.T) {
		keys, _ := collectKV(m.Range(Included(97), Unbounded[int]()))
		assert.Equal(t, []int{97, 98, 99}, keys)
	})

	t.Run("fully unbounded equals iter", func(t *testing.T) {
		all, _ := collectKV(m.Iter())
		rng, _ := collectKV(m.Range(Unbounded[int](), Unbounded[int]()))
		assert.Equal(t, all, rng)
	})

	t.Run("empty range", func(t *testing.T) {
		keys, _ := collectKV(m.Range(Included(20), Excluded(20)))
		assert.Empty(t, keys)
	})

	t.Run("bounds outside data", func(t *testing.T) {
		keys, _ := collectKV(m.Range(Included(-5), Excluded(3)))
		assert.Equal(t, []int{0, 1, 2}, keys)
		keys, _ = collectKV(m.Range(Included(98), Included(500)))
		assert.Equal(t, []int{98, 99}, keys)
	})
}

func TestBTreeMapLowerUpperBound(t *testing.T) {
	m := filledMap(20)

	lower, _ := collectKV(m.LowerBound(Included(17)))
	assert.Equal(t, []int{17, 18, 19}, lower)

	lowerEx, _ := collectKV(m.LowerBound(Excluded(17)))
	assert.Equal(t, []int{18, 19}, lowerEx)

	upper, _ := collectKV(m.UpperBound(Included(2)))
	assert.Equal(t, []int{0, 1, 2}, upper)

	upperEx, _ := collectKV(m.UpperBound(Excluded(2)))
	assert.Equal(t, []int{0, 1}, upperEx)
}

func TestBTreeMapFirstLast(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		m := NewBTreeMap[int, int]()
		_, _, ok := m.FirstKeyValue()
		assert.False(t, ok)
		_, _, ok = m.LastKeyValue()
		assert.False(t, ok)
	})

	t.Run("populated", func(t *testing.T) {
		m := filledMap(100)
		k, v, ok := m.FirstKeyValue()
		assert.True(t, ok)
		assert.Equal(t, 0, k)
		assert.Equal(t, 0, v)

		k, v, ok = m.LastKeyValue()
		assert.True(t, ok)
		assert.Equal(t, 99, k)
		assert.Equal(t, 990, v)
	})
}

func TestBTreeMapPopFirstLast(t *testing.T) {
	m := filledMap(10)

	k, v, ok := m.PopFirst()
	assert.True(t, ok)
	assert.Equal(t, 0, k)
	assert.Equal(t, 0, v)
	assert.False(t, m.ContainsKey(0))
	assert.Equal(t, 9, m.Len())

	k, v, ok = m.PopLast()
	assert.True(t, ok)
	assert.Equal(t, 9, k)
	assert.Equal(t, 90, v)
	assert.Equal(t, 8, m.Len())

	empty := NewBTreeMap[int, int]()
	_, _, ok = empty.PopFirst()
	assert.False(t, ok)
	_, _, ok = empty.PopLast()
	assert.False(t, ok)
}

func TestBTreeMapRetain(t *testing.T) {
	m := filledMap(100)
	m.Retain(func(k, _ int) bool { return k%2 == 0 })
	keys, _ := collectKV(m.Iter())
	assert.Len(t, keys, 50)
	for _, k := range keys {
		assert.Equal(t, 0, k%2)
	}
}

func TestBTreeMapExtractIf(t *testing.T) {
	t.Run("lazy predicate and repeated iteration", func(t *testing.T) {
		m := filledMap(100)
		var visited []int
		seq := m.ExtractIf(func(k, v int) bool {
			visited = append(visited, k)
			return v%20 == 0
		})
		require.Empty(t, visited, "creating an iterator must not call the predicate")
		m.Insert(0, 1)
		for k, v := range seq {
			assert.Equal(t, 2, k)
			assert.Equal(t, 20, v)
			break
		}
		assert.Equal(t, []int{0, 1, 2}, visited, "stop evaluating after early termination")
		assert.True(t, m.ContainsKey(0))
		assert.True(t, m.ContainsKey(4))
		assert.False(t, m.ContainsKey(2))
		keys, _ := collectKV(seq)
		require.Len(t, keys, 48)
		assert.Equal(t, 4, keys[0], "a second iteration must not yield removed entries")
	})

	t.Run("predicate panic preserves unvisited entries", func(t *testing.T) {
		m := filledMap(100)
		seq := m.ExtractIf(func(k, _ int) bool {
			if k == 1 {
				panic("predicate")
			}
			return true
		})
		assert.Panics(t, func() { collectKV(seq) })
		assert.False(t, m.ContainsKey(0))
		assert.True(t, m.ContainsKey(1))
		assert.True(t, m.ContainsKey(2))
	})

	t.Run("full drain", func(t *testing.T) {
		m := filledMap(10)
		extracted, _ := collectKV(m.ExtractIf(func(k, _ int) bool { return k < 3 }))
		assert.Equal(t, []int{0, 1, 2}, extracted)
		assert.Equal(t, 7, m.Len())
		assert.False(t, m.ContainsKey(1))
		assert.True(t, m.ContainsKey(3))
	})

	t.Run("lazy removal on early stop", func(t *testing.T) {
		m := filledMap(10)
		var seen []int
		for k := range m.ExtractIf(func(k, _ int) bool { return k < 5 }) {
			seen = append(seen, k)
			if k == 1 {
				break
			}
		}
		assert.Equal(t, []int{0, 1}, seen)
		// Only yielded matches are removed; 2,3,4 remain (Rust semantics).
		assert.False(t, m.ContainsKey(0))
		assert.False(t, m.ContainsKey(1))
		assert.True(t, m.ContainsKey(2))
		assert.True(t, m.ContainsKey(4))
		assert.Equal(t, 8, m.Len())
	})
}

func TestBTreeMapAppend(t *testing.T) {
	a := NewBTreeMap[int, int]()
	a.Insert(1, 1)
	a.Insert(2, 2)
	b := NewBTreeMap[int, int]()
	b.Insert(2, 20) // collision: b wins
	b.Insert(3, 3)

	a.Append(b)
	assert.Equal(t, 0, b.Len(), "source drained")
	assert.Equal(t, 3, a.Len())
	v, _ := a.Get(2)
	assert.Equal(t, 20, v)
	keys, _ := collectKV(a.Iter())
	assert.Equal(t, []int{1, 2, 3}, keys)
}

func TestBTreeMapAppendSelf(t *testing.T) {
	m := filledMap(100)
	m.Append(m)
	require.Equal(t, 100, m.Len())
	for i := 0; i < 100; i++ {
		v, ok := m.Get(i)
		assert.True(t, ok)
		assert.Equal(t, i*10, v)
	}
}

func TestBTreeMapSplitOff(t *testing.T) {
	m := filledMap(10)
	hi := m.SplitOff(5)

	loKeys, _ := collectKV(m.Iter())
	hiKeys, _ := collectKV(hi.Iter())
	assert.Equal(t, []int{0, 1, 2, 3, 4}, loKeys)
	assert.Equal(t, []int{5, 6, 7, 8, 9}, hiKeys)
}

func TestBTreeMapIterEarlyStop(t *testing.T) {
	m := filledMap(200)

	var seen []int
	for k := range m.Iter() {
		seen = append(seen, k)
		if k == 5 {
			break
		}
	}
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, seen)

	seen = nil
	for k := range m.Range(Included(10), Excluded(100)) {
		seen = append(seen, k)
		if k == 12 {
			break
		}
	}
	assert.Equal(t, []int{10, 11, 12}, seen)

	var keysSeen []int
	for k := range m.Keys() {
		keysSeen = append(keysSeen, k)
		if k == 1 {
			break
		}
	}
	assert.Equal(t, []int{0, 1}, keysSeen)

	var valsSeen []int
	for v := range m.Values() {
		valsSeen = append(valsSeen, v)
		if v == 10 {
			break
		}
	}
	assert.Equal(t, []int{0, 10}, valsSeen)
}

func BenchmarkBTreeMapRange(b *testing.B) {
	m := filledMap(100000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		for range m.Range(Included(1000), Excluded(2000)) {
			count++
		}
	}
}

func FuzzBTreeMapRange(f *testing.F) {
	f.Add(10, 50, 100)
	f.Fuzz(func(t *testing.T, lo, hi, n int) {
		if n < 0 || n > 2000 {
			n %= 2001
			if n < 0 {
				n = -n
			}
		}
		m := NewBTreeMap[int, int]()
		for i := 0; i < n; i++ {
			m.Insert(i, i)
		}
		keys, _ := collectKV(m.Range(Included(lo), Excluded(hi)))
		require.True(t, slices.IsSorted(keys))
		for _, k := range keys {
			require.GreaterOrEqual(t, k, lo)
			require.Less(t, k, hi)
		}
	})
}
