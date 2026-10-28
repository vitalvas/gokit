package xstd

import (
	"maps"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBTreeMapInsertGet(t *testing.T) {
	t.Run("insert and retrieve", func(t *testing.T) {
		m := NewBTreeMap[string, int]()

		old, existed := m.Insert("a", 1)
		assert.False(t, existed)
		assert.Zero(t, old)

		v, ok := m.Get("a")
		assert.True(t, ok)
		assert.Equal(t, 1, v)
		assert.True(t, m.ContainsKey("a"))
		assert.Equal(t, 1, m.Len())
		assert.False(t, m.IsEmpty())
	})

	t.Run("update existing returns old", func(t *testing.T) {
		m := NewBTreeMap[string, int]()
		m.Insert("a", 1)

		old, existed := m.Insert("a", 2)
		assert.True(t, existed)
		assert.Equal(t, 1, old)

		v, _ := m.Get("a")
		assert.Equal(t, 2, v)
		assert.Equal(t, 1, m.Len())
	})

	t.Run("missing key", func(t *testing.T) {
		m := NewBTreeMap[string, int]()
		v, ok := m.Get("missing")
		assert.False(t, ok)
		assert.Zero(t, v)
		assert.False(t, m.ContainsKey("missing"))
		assert.True(t, m.IsEmpty())
	})

	t.Run("get key value", func(t *testing.T) {
		m := NewBTreeMap[int, string]()
		m.Insert(5, "five")

		k, v, ok := m.GetKeyValue(5)
		assert.True(t, ok)
		assert.Equal(t, 5, k)
		assert.Equal(t, "five", v)

		k, v, ok = m.GetKeyValue(9)
		assert.False(t, ok)
		assert.Zero(t, k)
		assert.Zero(t, v)
	})
}

func TestBTreeMapTryInsert(t *testing.T) {
	m := NewBTreeMap[int, int]()

	v, ok := m.TryInsert(1, 10)
	assert.True(t, ok)
	assert.Equal(t, 10, v)

	v, ok = m.TryInsert(1, 20)
	assert.False(t, ok)
	assert.Equal(t, 10, v, "existing value is kept")

	got, _ := m.Get(1)
	assert.Equal(t, 10, got)
}

func TestBTreeMapGetOrInsertWith(t *testing.T) {
	m := NewBTreeMap[string, int]()
	calls := 0
	f := func() int { calls++; return 42 }

	assert.Equal(t, 42, m.GetOrInsertWith("a", f))
	assert.Equal(t, 1, calls)
	assert.Equal(t, 42, m.GetOrInsertWith("a", f))
	assert.Equal(t, 1, calls, "f not called when present")
}

func TestBTreeMapClear(t *testing.T) {
	m := NewBTreeMap[int, int]()
	for i := 0; i < 100; i++ {
		m.Insert(i, i)
	}
	m.Clear()
	assert.Equal(t, 0, m.Len())
	assert.True(t, m.IsEmpty())
	_, ok := m.Get(5)
	assert.False(t, ok)
}

func TestBTreeMapOrderedAgainstReference(t *testing.T) {
	// Enough keys to force several root splits (2t-1 = 63 per node).
	const n int = 5000
	m := NewBTreeMap[int, int]()
	ref := make(map[int]int)

	for i := 0; i < n; i++ {
		k := (i * 2654435761) % n
		m.Insert(k, k*10)
		ref[k] = k * 10
	}
	require.Equal(t, len(ref), m.Len())

	want := slices.Sorted(maps.Keys(ref))
	gotKeys, gotVals := collectKV(m.Iter())
	assert.Equal(t, want, gotKeys)
	for i, k := range gotKeys {
		assert.Equal(t, k*10, gotVals[i])
	}

	for k, v := range ref {
		got, ok := m.Get(k)
		require.True(t, ok)
		require.Equal(t, v, got)
	}
}

func TestBTreeMapRemove(t *testing.T) {
	t.Run("remove from empty", func(t *testing.T) {
		m := NewBTreeMap[int, int]()
		v, ok := m.Remove(1)
		assert.False(t, ok)
		assert.Zero(t, v)
	})

	t.Run("remove missing", func(t *testing.T) {
		m := NewBTreeMap[int, int]()
		m.Insert(1, 10)
		v, ok := m.Remove(2)
		assert.False(t, ok)
		assert.Zero(t, v)
		assert.Equal(t, 1, m.Len())
	})

	t.Run("remove returns old value", func(t *testing.T) {
		m := NewBTreeMap[int, int]()
		m.Insert(1, 10)
		v, ok := m.Remove(1)
		assert.True(t, ok)
		assert.Equal(t, 10, v)
		assert.Equal(t, 0, m.Len())
		assert.False(t, m.ContainsKey(1))
	})

	t.Run("remove entry", func(t *testing.T) {
		m := NewBTreeMap[int, string]()
		m.Insert(3, "three")

		k, v, ok := m.RemoveEntry(3)
		assert.True(t, ok)
		assert.Equal(t, 3, k)
		assert.Equal(t, "three", v)

		k, v, ok = m.RemoveEntry(3)
		assert.False(t, ok)
		assert.Zero(t, k)
		assert.Zero(t, v)
	})
}

func TestBTreeMapRemoveOrders(t *testing.T) {
	// Each order exercises different rebalancing: ascending triggers
	// borrow-from-next/merge, descending triggers borrow-from-prev and
	// predecessor replacement for internal separators.
	orders := map[string]func(n int) []int{
		"ascending": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = i
			}
			return s
		},
		"descending": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = n - 1 - i
			}
			return s
		},
		"interleaved": func(n int) []int {
			s := make([]int, n)
			for i := range s {
				s[i] = (i * 2654435761) % n
			}
			return s
		},
	}
	const n int = 2000
	for name, order := range orders {
		t.Run(name, func(t *testing.T) {
			m := NewBTreeMap[int, int]()
			ref := make(map[int]int)
			for i := 0; i < n; i++ {
				m.Insert(i, i*3)
				ref[i] = i * 3
			}
			for _, k := range order(n) {
				v, ok := m.Remove(k)
				require.True(t, ok, "key %d", k)
				require.Equal(t, ref[k], v)
				delete(ref, k)
				require.Equal(t, len(ref), m.Len())

				keys, _ := collectKV(m.Iter())
				require.True(t, slices.IsSorted(keys))
				require.Len(t, keys, len(ref))
			}
			assert.Equal(t, 0, m.Len())
		})
	}
}

func BenchmarkBTreeMapInsert(b *testing.B) {
	m := NewBTreeMap[int, int]()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.Insert(i%100000, i)
	}
}

func BenchmarkBTreeMapGet(b *testing.B) {
	m := NewBTreeMap[int, int]()
	for i := 0; i < 100000; i++ {
		m.Insert(i, i)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Get(i % 100000)
	}
}

func FuzzBTreeMap(f *testing.F) {
	f.Add([]byte{1, 2, 3, 2, 1, 5})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) {
		m := NewBTreeMap[byte, int]()
		ref := make(map[byte]int)
		for i, b := range data {
			if i%3 == 2 {
				delM, okM := m.Remove(b)
				delR, okR := ref[b]
				require.Equal(t, okR, okM)
				if okR {
					require.Equal(t, delR, delM)
				}
				delete(ref, b)
			} else {
				m.Insert(b, i)
				ref[b] = i
			}
			require.Equal(t, len(ref), m.Len())
		}
		keys, _ := collectKV(m.Iter())
		require.True(t, slices.IsSorted(keys))
		require.Equal(t, len(ref), len(keys))
		for _, k := range keys {
			v, ok := m.Get(k)
			require.True(t, ok)
			require.Equal(t, ref[k], v)
		}
	})
}
