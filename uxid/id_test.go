package uxid

import (
	"testing"
	"time"
)

func BenchmarkNew(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New()
		}
	})
}

func BenchmarkNewWithTime(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		ts := time.Now()
		for pb.Next() {
			_ = NewWithTime(ts)
		}
	})
}

func BenchmarkNewString(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = New().String()
		}
	})
}

func BenchmarkNewWithTimeString(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		ts := time.Now()
		for pb.Next() {
			_ = NewWithTime(ts).String()
		}
	})
}
