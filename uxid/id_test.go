package uxid

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	t.Run("creates unique IDs", func(t *testing.T) {
		id1 := New()
		id2 := New()
		assert.NotEqual(t, id1.String(), id2.String())
	})

	t.Run("string has correct length", func(t *testing.T) {
		id := New()
		assert.Len(t, id.String(), 24)
	})
}

func TestNewWithTime(t *testing.T) {
	t.Run("uses provided time", func(t *testing.T) {
		ts := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		id1 := NewWithTime(ts)
		id2 := NewWithTime(ts)
		assert.NotEqual(t, id1.String(), id2.String())
	})
}

func BenchmarkID_New(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = New()
	}
}

func BenchmarkID_NewWithTime(b *testing.B) {
	ts := time.Now()
	b.ReportAllocs()
	for b.Loop() {
		_ = NewWithTime(ts)
	}
}

func BenchmarkID_String(b *testing.B) {
	id := New()
	b.ReportAllocs()
	for b.Loop() {
		_ = id.String()
	}
}

func BenchmarkID_NewParallel(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New()
		}
	})
}

func FuzzID_Encode(f *testing.F) {
	f.Add(int64(0), int64(0))
	f.Add(int64(1234567890), int64(9876543210))
	f.Add(int64(1), int64(1))

	f.Fuzz(func(t *testing.T, ts, randVal int64) {
		if ts < 0 || randVal < 0 {
			return
		}
		id := ID{ts: ts, rand: randVal}
		s := id.String()
		if len(s) != 24 {
			t.Errorf("expected length 24, got %d", len(s))
		}
	})
}
