package xstd

import (
	"maps"
	"math"
	"math/rand/v2"
	"slices"
	"strconv"
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

func TestBTreeMapGetKeyValueAcrossNodes(t *testing.T) {
	m := filledMap(100)
	for _, key := range []int{0, btreeDegree - 1, 50, 99} {
		k, v, ok := m.GetKeyValue(key)
		assert.True(t, ok)
		assert.Equal(t, key, k)
		assert.Equal(t, key*10, v)
	}
	for _, key := range []int{-1, 100} {
		k, v, ok := m.GetKeyValue(key)
		assert.False(t, ok)
		assert.Zero(t, k)
		assert.Zero(t, v)
	}
}

func TestBTreeMapUpdateDuringChildSplit(t *testing.T) {
	// The right child is full; updating its median splits the child and
	// promotes the existing key before its value is replaced.
	const n = 3*btreeDegree - 1
	const key = 2*btreeDegree - 1
	m := filledMap(n)
	old, existed := m.Insert(key, -1)
	require.True(t, existed)
	assert.Equal(t, key*10, old)
	assert.Equal(t, n, m.Len())
	keys, values := collectKV(m.Iter())
	require.Len(t, keys, n)
	for i, k := range keys {
		assert.Equal(t, i, k)
		if k == key {
			assert.Equal(t, -1, values[i])
		} else {
			assert.Equal(t, k*10, values[i])
		}
	}
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

func TestBTreeMapRemoveLastEntry(t *testing.T) {
	m := filledMap(100)
	for i := 0; i < 100; i++ {
		_, ok := m.Remove(i)
		require.True(t, ok)
	}
	require.True(t, m.IsEmpty())
	_, _, ok := m.FirstKeyValue()
	assert.False(t, ok)
	_, _, ok = m.LastKeyValue()
	assert.False(t, ok)
	_, _, ok = m.PopFirst()
	assert.False(t, ok)
	_, _, ok = m.PopLast()
	assert.False(t, ok)
	m.Insert(1, 10)
	v, ok := m.Get(1)
	assert.True(t, ok)
	assert.Equal(t, 10, v)
}

func TestBTreeMapRemoveMissingCollapsesRoot(t *testing.T) {
	// Make both children minimal so removing an absent key merges them.
	m := filledMap(2 * btreeDegree)
	m.Remove(2*btreeDegree - 1)
	for i := 0; i < 2; i++ {
		v, ok := m.Remove(-1)
		require.False(t, ok)
		require.Zero(t, v)
		require.Equal(t, 2*btreeDegree-1, m.Len())
		require.NotEmpty(t, m.root.keys)
	}
	keys, _ := collectKV(m.Iter())
	want := make([]int, 2*btreeDegree-1)
	for i := range want {
		want[i] = i
	}
	assert.Equal(t, want, keys)
}

func TestBTreeMapRemoveEntryStoredKey(t *testing.T) {
	negativeZero := math.Copysign(0, -1)
	m := NewBTreeMap[float64, string]()
	m.Insert(negativeZero, "zero")
	k, v, ok := m.RemoveEntry(0)
	require.True(t, ok)
	assert.True(t, math.Signbit(k), "return the stored key, not the lookup key")
	assert.Equal(t, "zero", v)
}

func TestBTreeMapRebalancingReleasesReferences(t *testing.T) {
	m := NewBTreeMap[string, *int]()
	const n = 5000
	for i := 0; i < n; i++ {
		m.Insert(strconv.Itoa(i), &i)
	}
	var check func(*btreeNode[string, *int]) int
	check = func(node *btreeNode[string, *int]) int {
		if node == nil {
			return 0
		}
		require.NotEmpty(t, node.keys)
		require.LessOrEqual(t, len(node.keys), 2*btreeDegree-1)
		if node != m.root {
			require.GreaterOrEqual(t, len(node.keys), btreeDegree-1)
		}
		require.Len(t, node.values, len(node.keys))
		require.True(t, slices.IsSorted(node.keys))
		for _, key := range node.keys[len(node.keys):cap(node.keys)] {
			require.Empty(t, key, "unused key slots must release string references")
		}
		for _, value := range node.values[len(node.values):cap(node.values)] {
			require.Nil(t, value, "unused value slots must release pointers")
		}
		for _, child := range node.children[len(node.children):cap(node.children)] {
			require.Nil(t, child, "unused child slots must release subtrees")
		}
		if !node.leaf() {
			require.Len(t, node.children, len(node.keys)+1)
		}
		count := len(node.keys)
		for _, child := range node.children {
			count += check(child)
		}
		return count
	}
	require.Equal(t, m.Len(), check(m.root))
	rng := rand.New(rand.NewPCG(1, 2))
	for i, key := range rng.Perm(n) {
		value, ok := m.Remove(strconv.Itoa(key))
		require.True(t, ok)
		require.Equal(t, key, *value)
		if i%100 == 0 || m.IsEmpty() {
			require.Equal(t, m.Len(), check(m.root))
		}
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
