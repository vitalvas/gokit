package markov

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewChain(t *testing.T) {
	tests := []struct {
		name  string
		order int
	}{
		{"Order 1", 1},
		{"Order 2", 2},
		{"Order 3", 3},
		{"Order 5", 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain := NewChain(tt.order)
			assert.NotNil(t, chain)
			assert.Equal(t, tt.order, chain.Order)
			assert.NotNil(t, chain.statePool)
			assert.NotNil(t, chain.frequencyMat)
			assert.NotNil(t, chain.lock)
		})
	}
}

func TestChain_Add(t *testing.T) {
	t.Run("Add single sequence", func(t *testing.T) {
		chain := NewChain(2)
		input := []string{"a", "b", "c"}
		chain.Add(input)

		assert.NotEmpty(t, chain.frequencyMat)
		assert.NotEmpty(t, chain.statePool.stringMap)
	})

	t.Run("Add multiple sequences", func(t *testing.T) {
		chain := NewChain(2)
		sequences := [][]string{
			{"a", "b", "c"},
			{"a", "b", "d"},
			{"b", "c", "d"},
		}

		for _, seq := range sequences {
			chain.Add(seq)
		}

		assert.NotEmpty(t, chain.frequencyMat)
	})

	t.Run("Add empty sequence", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{})
		assert.NotEmpty(t, chain.frequencyMat)
	})
}

func TestChain_RawAdd(t *testing.T) {
	t.Run("Add string", func(t *testing.T) {
		chain := NewChain(2)
		chain.RawAdd("hello")

		assert.NotEmpty(t, chain.frequencyMat)
		assert.NotEmpty(t, chain.statePool.stringMap)
	})

	t.Run("Add empty string", func(t *testing.T) {
		chain := NewChain(2)
		chain.RawAdd("")

		assert.NotEmpty(t, chain.frequencyMat)
	})
}

func TestChain_TransitionProbability(t *testing.T) {
	t.Run("Valid transition", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})
		chain.Add([]string{"a", "b", "c"})
		chain.Add([]string{"a", "b", "d"})

		prob, err := chain.TransitionProbability("a", NGram{StartToken, StartToken})
		require.NoError(t, err)
		assert.Greater(t, prob, 0.0)
		assert.Equal(t, 1.0, prob)
	})

	t.Run("Invalid n-gram length", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})

		_, err := chain.TransitionProbability("c", NGram{"a"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "n-gram length does not match chain order")
	})

	t.Run("Unknown n-gram", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})

		prob, err := chain.TransitionProbability("z", NGram{"x", "y"})
		require.NoError(t, err)
		assert.Equal(t, 0.0, prob)
	})

	t.Run("Unknown next state", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})

		prob, err := chain.TransitionProbability("z", NGram{StartToken, StartToken})
		require.NoError(t, err)
		assert.Equal(t, 0.0, prob)
	})
}

func TestChain_Generate(t *testing.T) {
	t.Run("Valid generation", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})
		chain.Add([]string{"a", "b", "d"})

		next, err := chain.Generate(NGram{StartToken, StartToken})
		require.NoError(t, err)
		assert.NotEmpty(t, next)
	})

	t.Run("Invalid n-gram length", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})

		_, err := chain.Generate(NGram{"a"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "n-gram length does not match chain order")
	})

	t.Run("Unknown n-gram", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})

		_, err := chain.Generate(NGram{"x", "y"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown ngram")
	})
}

func TestChain_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent Add", func(t *testing.T) {
		chain := NewChain(2)
		sequences := [][]string{
			{"a", "b", "c"},
			{"d", "e", "f"},
			{"g", "h", "i"},
		}

		done := make(chan bool, len(sequences))

		for _, seq := range sequences {
			go func(s []string) {
				chain.Add(s)
				done <- true
			}(seq)
		}

		for i := 0; i < len(sequences); i++ {
			<-done
		}

		assert.NotEmpty(t, chain.frequencyMat)
	})

	t.Run("Concurrent Generate", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})
		chain.Add([]string{"a", "b", "d"})

		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func() {
				_, err := chain.Generate(NGram{StartToken, StartToken})
				assert.NoError(t, err)
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

func TestChain_CompleteSequenceGeneration(t *testing.T) {
	t.Run("Generate complete sequence", func(t *testing.T) {
		chain := NewChain(2)

		words := []string{"hello", "world", "test"}
		for _, word := range words {
			chain.RawAdd(word)
		}

		tokens := make([]string, 0)
		for i := 0; i < chain.Order; i++ {
			tokens = append(tokens, StartToken)
		}

		maxLen := 100
		for i := 0; i < maxLen && tokens[len(tokens)-1] != EndToken; i++ {
			next, err := chain.Generate(tokens[len(tokens)-chain.Order:])
			require.NoError(t, err)
			tokens = append(tokens, next)
		}

		assert.Equal(t, EndToken, tokens[len(tokens)-1])
	})
}

func TestStartAndEndTokens(t *testing.T) {
	t.Run("Verify token constants", func(t *testing.T) {
		assert.Equal(t, "^", StartToken)
		assert.Equal(t, "$", EndToken)
	})
}

func TestChain_Export(t *testing.T) {
	t.Run("Export chain", func(t *testing.T) {
		chain := NewChain(2)
		chain.Add([]string{"a", "b", "c"})
		chain.Add([]string{"a", "b", "d"})

		data, err := chain.Export()
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("Export empty chain", func(t *testing.T) {
		chain := NewChain(3)

		data, err := chain.Export()
		require.NoError(t, err)
		assert.NotEmpty(t, data)
	})
}

func TestImportChain(t *testing.T) {
	t.Run("Import valid chain", func(t *testing.T) {
		original := NewChain(2)
		original.Add([]string{"a", "b", "c"})
		original.Add([]string{"a", "b", "d"})

		data, err := original.Export()
		require.NoError(t, err)

		restored, err := ImportChain(data)
		require.NoError(t, err)

		assert.Equal(t, original.Order, restored.Order)
		assert.NotNil(t, restored.statePool)
		assert.NotNil(t, restored.frequencyMat)
		assert.NotNil(t, restored.lock)
	})

	t.Run("Import invalid data", func(t *testing.T) {
		_, err := ImportChain([]byte("invalid data"))
		assert.Error(t, err)
	})

	t.Run("Round trip", func(t *testing.T) {
		original := NewChain(3)
		sequences := [][]string{
			{"h", "e", "l", "l", "o"},
			{"w", "o", "r", "l", "d"},
			{"h", "e", "l", "l", "o"},
		}

		for _, seq := range sequences {
			original.Add(seq)
		}

		data, err := original.Export()
		require.NoError(t, err)

		restored, err := ImportChain(data)
		require.NoError(t, err)

		assert.Equal(t, original.Order, restored.Order)

		next1, err1 := original.Generate(NGram{StartToken, StartToken, StartToken})
		next2, err2 := restored.Generate(NGram{StartToken, StartToken, StartToken})
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEmpty(t, next1)
		assert.NotEmpty(t, next2)
	})

	t.Run("Verify state preservation", func(t *testing.T) {
		original := NewChain(2)
		original.RawAdd("hello")
		original.RawAdd("world")

		data, err := original.Export()
		require.NoError(t, err)

		restored, err := ImportChain(data)
		require.NoError(t, err)

		prob1, _ := original.TransitionProbability("h", NGram{StartToken, StartToken})
		prob2, _ := restored.TransitionProbability("h", NGram{StartToken, StartToken})
		assert.Equal(t, prob1, prob2)
	})
}

func BenchmarkNewChain(b *testing.B) {
	orders := []int{1, 2, 3, 5, 10}

	for _, order := range orders {
		b.Run(fmt.Sprintf("Order_%d", order), func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				_ = NewChain(order)
			}
		})
	}
}

func BenchmarkChain_Add(b *testing.B) {
	sequences := []struct {
		name   string
		tokens []string
	}{
		{"Short_3", []string{"a", "b", "c"}},
		{"Medium_10", []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
		{"Long_50", make([]string, 50)},
		{"VeryLong_100", make([]string, 100)},
	}

	for i := range sequences[2].tokens {
		sequences[2].tokens[i] = string(rune('a' + i%26))
	}
	for i := range sequences[3].tokens {
		sequences[3].tokens[i] = string(rune('a' + i%26))
	}

	for _, seq := range sequences {
		b.Run(seq.name, func(b *testing.B) {
			chain := NewChain(2)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				chain.Add(seq.tokens)
			}
		})
	}
}

func BenchmarkChain_RawAdd(b *testing.B) {
	strings := []struct {
		name  string
		input string
	}{
		{"Short", "hello"},
		{"Medium", "hello world test"},
		{"Long", "Lorem ipsum dolor sit amet, consectetur adipiscing elit"},
		{"VeryLong", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris."},
	}

	for _, str := range strings {
		b.Run(str.name, func(b *testing.B) {
			chain := NewChain(2)
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				chain.RawAdd(str.input)
			}
		})
	}
}

func BenchmarkChain_Generate(b *testing.B) {
	orders := []int{1, 2, 3, 5}

	for _, order := range orders {
		b.Run(fmt.Sprintf("Order_%d", order), func(b *testing.B) {
			chain := NewChain(order)

			for i := 0; i < 100; i++ {
				chain.Add([]string{"a", "b", "c", "d", "e"})
				chain.Add([]string{"b", "c", "d", "e", "f"})
			}

			ngram := make(NGram, order)
			for i := 0; i < order; i++ {
				ngram[i] = StartToken
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				_, _ = chain.Generate(ngram)
			}
		})
	}
}

func BenchmarkChain_TransitionProbability(b *testing.B) {
	chain := NewChain(2)

	for i := 0; i < 100; i++ {
		chain.Add([]string{"a", "b", "c"})
		chain.Add([]string{"a", "b", "d"})
	}

	ngram := NGram{StartToken, StartToken}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = chain.TransitionProbability("a", ngram)
	}
}

func BenchmarkChain_ConcurrentAdd(b *testing.B) {
	concurrency := []int{1, 2, 4, 8, 16}

	for _, c := range concurrency {
		b.Run(fmt.Sprintf("Goroutines_%d", c), func(b *testing.B) {
			chain := NewChain(2)
			tokens := []string{"a", "b", "c", "d", "e"}

			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					chain.Add(tokens)
				}
			})
		})
	}
}

func BenchmarkChain_ConcurrentGenerate(b *testing.B) {
	concurrency := []int{1, 2, 4, 8, 16}

	for _, c := range concurrency {
		b.Run(fmt.Sprintf("Goroutines_%d", c), func(b *testing.B) {
			chain := NewChain(2)

			for i := 0; i < 1000; i++ {
				chain.Add([]string{"a", "b", "c", "d", "e"})
			}

			ngram := NGram{StartToken, StartToken}

			b.SetParallelism(c)
			b.ReportAllocs()
			b.ResetTimer()

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					_, _ = chain.Generate(ngram)
				}
			})
		})
	}
}

func BenchmarkChain_SequenceGeneration(b *testing.B) {
	datasets := []struct {
		name  string
		words []string
	}{
		{
			name:  "Small_10",
			words: []string{"hello", "world", "test", "foo", "bar", "baz", "qux", "alpha", "beta", "gamma"},
		},
		{
			name:  "Medium_50",
			words: make([]string, 50),
		},
		{
			name:  "Large_100",
			words: make([]string, 100),
		},
	}

	for i := range datasets[1].words {
		datasets[1].words[i] = fmt.Sprintf("word%d", i)
	}
	for i := range datasets[2].words {
		datasets[2].words[i] = fmt.Sprintf("word%d", i)
	}

	for _, ds := range datasets {
		b.Run(ds.name, func(b *testing.B) {
			chain := NewChain(2)

			for _, word := range ds.words {
				chain.RawAdd(word)
			}

			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				tokens := make([]string, 0)
				for j := 0; j < chain.Order; j++ {
					tokens = append(tokens, StartToken)
				}

				for len(tokens) < 100 && tokens[len(tokens)-1] != EndToken {
					next, _ := chain.Generate(tokens[len(tokens)-chain.Order:])
					tokens = append(tokens, next)
				}
			}
		})
	}
}

func BenchmarkFullWorkflow(b *testing.B) {
	words := []string{
		"hello", "world", "test", "example", "markov", "chain",
		"generator", "random", "probability", "sequence",
	}

	b.Run("TrainAndGenerate", func(b *testing.B) {
		b.ReportAllocs()

		for b.Loop() {
			chain := NewChain(2)

			for _, word := range words {
				chain.RawAdd(word)
			}

			tokens := make([]string, 0)
			for j := 0; j < chain.Order; j++ {
				tokens = append(tokens, StartToken)
			}

			for len(tokens) < 50 && tokens[len(tokens)-1] != EndToken {
				next, _ := chain.Generate(tokens[len(tokens)-chain.Order:])
				tokens = append(tokens, next)
			}
		}
	})
}

func BenchmarkChain_Export(b *testing.B) {
	chain := NewChain(2)

	for i := 0; i < 100; i++ {
		chain.Add([]string{"a", "b", "c", "d", "e"})
		chain.Add([]string{"b", "c", "d", "e", "f"})
	}

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = chain.Export()
	}
}

func BenchmarkImportChain(b *testing.B) {
	chain := NewChain(2)

	for i := 0; i < 100; i++ {
		chain.Add([]string{"a", "b", "c", "d", "e"})
		chain.Add([]string{"b", "c", "d", "e", "f"})
	}

	data, _ := chain.Export()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = ImportChain(data)
	}
}

func FuzzChain_Add(f *testing.F) {
	f.Add("hello")
	f.Add("world")
	f.Add("")
	f.Add("test data with spaces")

	f.Fuzz(func(_ *testing.T, s string) {
		chain := NewChain(2)
		chain.RawAdd(s)
	})
}

func FuzzChain_ExportImport(f *testing.F) {
	f.Add("hello")
	f.Add("world")
	f.Add("test")

	f.Fuzz(func(t *testing.T, s string) {
		chain := NewChain(2)
		chain.RawAdd(s)
		data, err := chain.Export()
		if err != nil {
			t.Errorf("export failed: %v", err)
			return
		}
		_, err = ImportChain(data)
		if err != nil {
			t.Errorf("import failed: %v", err)
		}
	})
}
