package markov

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNGram_key(t *testing.T) {
	tests := []struct {
		name     string
		ngram    NGram
		expected string
	}{
		{
			name:     "Single element",
			ngram:    NGram{"a"},
			expected: "a",
		},
		{
			name:     "Two elements",
			ngram:    NGram{"a", "b"},
			expected: "a:b",
		},
		{
			name:     "Three elements",
			ngram:    NGram{"a", "b", "c"},
			expected: "a:b:c",
		},
		{
			name:     "Empty ngram",
			ngram:    NGram{},
			expected: "",
		},
		{
			name:     "With special characters",
			ngram:    NGram{"hello", "world", "!"},
			expected: "hello:world:!",
		},
		{
			name:     "With start and end tokens",
			ngram:    NGram{StartToken, "a", EndToken},
			expected: "^:a:$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ngram.key()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSparseArray_sum(t *testing.T) {
	tests := []struct {
		name     string
		arr      sparseArray
		expected int
	}{
		{
			name:     "Empty array",
			arr:      sparseArray{},
			expected: 0,
		},
		{
			name:     "Single element",
			arr:      sparseArray{0: 5},
			expected: 5,
		},
		{
			name:     "Multiple elements",
			arr:      sparseArray{0: 5, 1: 10, 2: 3},
			expected: 18,
		},
		{
			name:     "Non-sequential indices",
			arr:      sparseArray{5: 2, 10: 7, 100: 1},
			expected: 10,
		},
		{
			name:     "Zero values",
			arr:      sparseArray{0: 0, 1: 0, 2: 0},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.arr.sum()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestArray(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		count    int
		expected []string
	}{
		{
			name:     "Zero count",
			value:    "a",
			count:    0,
			expected: []string{},
		},
		{
			name:     "Single element",
			value:    "a",
			count:    1,
			expected: []string{"a"},
		},
		{
			name:     "Multiple elements",
			value:    "x",
			count:    5,
			expected: []string{"x", "x", "x", "x", "x"},
		},
		{
			name:     "Start token",
			value:    StartToken,
			count:    3,
			expected: []string{StartToken, StartToken, StartToken},
		},
		{
			name:     "End token",
			value:    EndToken,
			count:    2,
			expected: []string{EndToken, EndToken},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := array(tt.value, tt.count)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, tt.count)
		})
	}
}

func TestMakePairs(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []string
		order    int
		expected []Pair
	}{
		{
			name:   "Order 1",
			tokens: []string{"a", "b", "c"},
			order:  1,
			expected: []Pair{
				{CurrentState: NGram{"a"}, NextState: "b"},
				{CurrentState: NGram{"b"}, NextState: "c"},
			},
		},
		{
			name:   "Order 2",
			tokens: []string{"a", "b", "c", "d"},
			order:  2,
			expected: []Pair{
				{CurrentState: NGram{"a", "b"}, NextState: "c"},
				{CurrentState: NGram{"b", "c"}, NextState: "d"},
			},
		},
		{
			name:   "Order 3",
			tokens: []string{"a", "b", "c", "d", "e"},
			order:  3,
			expected: []Pair{
				{CurrentState: NGram{"a", "b", "c"}, NextState: "d"},
				{CurrentState: NGram{"b", "c", "d"}, NextState: "e"},
			},
		},
		{
			name:     "Tokens equal to order",
			tokens:   []string{"a", "b"},
			order:    2,
			expected: nil,
		},
		{
			name:     "Tokens less than order",
			tokens:   []string{"a"},
			order:    2,
			expected: nil,
		},
		{
			name:   "With start and end tokens",
			tokens: []string{StartToken, StartToken, "a", "b", EndToken},
			order:  2,
			expected: []Pair{
				{CurrentState: NGram{StartToken, StartToken}, NextState: "a"},
				{CurrentState: NGram{StartToken, "a"}, NextState: "b"},
				{CurrentState: NGram{"a", "b"}, NextState: EndToken},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MakePairs(tt.tokens, tt.order)
			assert.Equal(t, tt.expected, result)
			assert.Len(t, result, len(tt.expected))
		})
	}
}

func TestPair(t *testing.T) {
	t.Run("Create pair", func(t *testing.T) {
		pair := Pair{
			CurrentState: NGram{"a", "b"},
			NextState:    "c",
		}

		assert.Equal(t, NGram{"a", "b"}, pair.CurrentState)
		assert.Equal(t, "c", pair.NextState)
	})

	t.Run("Pair with start tokens", func(t *testing.T) {
		pair := Pair{
			CurrentState: NGram{StartToken, StartToken},
			NextState:    "a",
		}

		assert.Equal(t, "^:^", pair.CurrentState.key())
		assert.Equal(t, "a", pair.NextState)
	})

	t.Run("Pair with end token", func(t *testing.T) {
		pair := Pair{
			CurrentState: NGram{"a", "b"},
			NextState:    EndToken,
		}

		assert.Equal(t, "a:b", pair.CurrentState.key())
		assert.Equal(t, "$", pair.NextState)
	})
}

func TestMakePairs_EdgeCases(t *testing.T) {
	t.Run("Empty tokens", func(t *testing.T) {
		result := MakePairs([]string{}, 2)
		assert.Empty(t, result)
	})

	t.Run("Large order", func(t *testing.T) {
		tokens := []string{"a", "b", "c", "d", "e"}
		result := MakePairs(tokens, 10)
		assert.Empty(t, result)
	})

	t.Run("Order 1 with single token", func(t *testing.T) {
		result := MakePairs([]string{"a"}, 1)
		assert.Empty(t, result)
	})
}

func BenchmarkMakePairs(b *testing.B) {
	orders := []int{1, 2, 3, 5}
	tokens := make([]string, 100)
	for i := range tokens {
		tokens[i] = string(rune('a' + i%26))
	}

	for _, order := range orders {
		b.Run(fmt.Sprintf("Order_%d", order), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = MakePairs(tokens, order)
			}
		})
	}
}

func BenchmarkNGram_key(b *testing.B) {
	ngrams := []struct {
		name  string
		ngram NGram
	}{
		{"Size_1", NGram{"a"}},
		{"Size_2", NGram{"a", "b"}},
		{"Size_3", NGram{"a", "b", "c"}},
		{"Size_5", NGram{"a", "b", "c", "d", "e"}},
		{"Size_10", NGram{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
	}

	for _, ng := range ngrams {
		b.Run(ng.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				_ = ng.ngram.key()
			}
		})
	}
}

func BenchmarkSparseArray_sum(b *testing.B) {
	sizes := []int{10, 50, 100, 500, 1000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			arr := make(sparseArray)
			for i := 0; i < size; i++ {
				arr[i] = i + 1
			}

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = arr.sum()
			}
		})
	}
}
