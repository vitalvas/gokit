package arccache

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	t.Run("new list is empty", func(t *testing.T) {
		l := newList[string, int]()

		assert.Equal(t, 0, l.len())
		assert.Nil(t, l.head())
		assert.Nil(t, l.tail())
	})

	t.Run("pushFront single element", func(t *testing.T) {
		l := newList[string, int]()
		e := &element[string, int]{key: "a", value: 1}

		l.pushFront(e)

		assert.Equal(t, 1, l.len())
		assert.Equal(t, e, l.head())
		assert.Equal(t, e, l.tail())
	})

	t.Run("pushFront multiple elements", func(t *testing.T) {
		l := newList[string, int]()
		e1 := &element[string, int]{key: "a", value: 1}
		e2 := &element[string, int]{key: "b", value: 2}
		e3 := &element[string, int]{key: "c", value: 3}

		l.pushFront(e1)
		l.pushFront(e2)
		l.pushFront(e3)

		assert.Equal(t, 3, l.len())
		assert.Equal(t, e3, l.head())
		assert.Equal(t, e1, l.tail())
	})

	t.Run("remove head", func(t *testing.T) {
		l := newList[string, int]()
		e1 := &element[string, int]{key: "a", value: 1}
		e2 := &element[string, int]{key: "b", value: 2}

		l.pushFront(e1)
		l.pushFront(e2)
		l.remove(e2)

		assert.Equal(t, 1, l.len())
		assert.Equal(t, e1, l.head())
		assert.Equal(t, e1, l.tail())
	})

	t.Run("remove tail", func(t *testing.T) {
		l := newList[string, int]()
		e1 := &element[string, int]{key: "a", value: 1}
		e2 := &element[string, int]{key: "b", value: 2}

		l.pushFront(e1)
		l.pushFront(e2)
		l.remove(e1)

		assert.Equal(t, 1, l.len())
		assert.Equal(t, e2, l.head())
		assert.Equal(t, e2, l.tail())
	})

	t.Run("remove middle", func(t *testing.T) {
		l := newList[string, int]()
		e1 := &element[string, int]{key: "a", value: 1}
		e2 := &element[string, int]{key: "b", value: 2}
		e3 := &element[string, int]{key: "c", value: 3}

		l.pushFront(e1)
		l.pushFront(e2)
		l.pushFront(e3)
		l.remove(e2)

		assert.Equal(t, 2, l.len())
		assert.Equal(t, e3, l.head())
		assert.Equal(t, e1, l.tail())
	})

	t.Run("remove only element", func(t *testing.T) {
		l := newList[string, int]()
		e := &element[string, int]{key: "a", value: 1}

		l.pushFront(e)
		l.remove(e)

		assert.Equal(t, 0, l.len())
		assert.Nil(t, l.head())
		assert.Nil(t, l.tail())
	})

	t.Run("moveToFront already at front", func(t *testing.T) {
		l := newList[string, int]()
		e1 := &element[string, int]{key: "a", value: 1}
		e2 := &element[string, int]{key: "b", value: 2}

		l.pushFront(e1)
		l.pushFront(e2)
		l.moveToFront(e2) // already at front

		assert.Equal(t, e2, l.head())
		assert.Equal(t, e1, l.tail())
	})

	t.Run("moveToFront from tail", func(t *testing.T) {
		l := newList[string, int]()
		e1 := &element[string, int]{key: "a", value: 1}
		e2 := &element[string, int]{key: "b", value: 2}
		e3 := &element[string, int]{key: "c", value: 3}

		l.pushFront(e1)
		l.pushFront(e2)
		l.pushFront(e3)

		l.moveToFront(e1) // tail -> front

		assert.Equal(t, 3, l.len())
		assert.Equal(t, e1, l.head())
		assert.Equal(t, e2, l.tail())
	})

	t.Run("moveToFront from middle", func(t *testing.T) {
		l := newList[string, int]()
		e1 := &element[string, int]{key: "a", value: 1}
		e2 := &element[string, int]{key: "b", value: 2}
		e3 := &element[string, int]{key: "c", value: 3}

		l.pushFront(e1)
		l.pushFront(e2)
		l.pushFront(e3)

		l.moveToFront(e2)

		assert.Equal(t, e2, l.head())
		assert.Equal(t, e1, l.tail())
		assert.Equal(t, e3, l.head().next)
	})

	t.Run("clear", func(t *testing.T) {
		l := newList[string, int]()
		l.pushFront(&element[string, int]{key: "a", value: 1})
		l.pushFront(&element[string, int]{key: "b", value: 2})
		l.pushFront(&element[string, int]{key: "c", value: 3})

		l.clear()

		assert.Equal(t, 0, l.len())
		assert.Nil(t, l.head())
		assert.Nil(t, l.tail())
	})

	t.Run("iteration forward", func(t *testing.T) {
		l := newList[string, int]()
		l.pushFront(&element[string, int]{key: "c", value: 3})
		l.pushFront(&element[string, int]{key: "b", value: 2})
		l.pushFront(&element[string, int]{key: "a", value: 1})

		var keys []string
		sentinel := &l.root
		for e := l.head(); e != nil && e != sentinel; e = e.next {
			keys = append(keys, e.key)
		}

		assert.Equal(t, []string{"a", "b", "c"}, keys)
	})
}

func BenchmarkList_PushFront(b *testing.B) {
	l := newList[string, int]()

	b.ReportAllocs()
	for b.Loop() {
		e := &element[string, int]{key: "k", value: 1}
		l.pushFront(e)
	}
}

func BenchmarkList_Remove(b *testing.B) {
	l := newList[string, int]()
	elements := make([]*element[string, int], b.N)
	for i := range elements {
		elements[i] = &element[string, int]{key: "k", value: i}
		l.pushFront(elements[i])
	}

	b.ReportAllocs()
	b.ResetTimer()
	for _, e := range elements {
		l.remove(e)
	}
}

func BenchmarkList_MoveToFront(b *testing.B) {
	l := newList[string, int]()
	e := &element[string, int]{key: "k", value: 1}
	l.pushFront(e)
	l.pushFront(&element[string, int]{key: "k2", value: 2})

	b.ReportAllocs()
	for b.Loop() {
		l.moveToFront(e)
	}
}
