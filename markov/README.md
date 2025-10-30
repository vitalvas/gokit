# markov

A high-performance Markov chain text generator in Go with thread-safe operations and efficient state management.

## Features

- **N-gram support**: Configurable chain order (1, 2, 3, or higher)
- **Thread-safe**: Concurrent-safe operations with efficient locking
- **High performance**: Optimized sparse arrays and state pooling
- **Simple API**: Easy-to-use interface for training and generation
- **Flexible input**: Support for both raw strings and token arrays
- **Probability queries**: Calculate transition probabilities
- **Serialization**: Export and import chains for persistence
- **Zero dependencies**: Only uses Go standard library

## What is a Markov Chain?

A Markov chain is a probabilistic model that generates sequences based on observed patterns. Given a sequence of tokens (words, characters, etc.), it predicts the next token based on the previous N tokens (where N is the chain order).

**Use Cases**: Text generation, name generation, music composition, predictive text, autocomplete systems.

## Installation

```bash
go get github.com/vitalvas/gokit/markov
```

## Quick Start

```go
package main

import (
    "fmt"
    "strings"
    "github.com/vitalvas/gokit/markov"
)

func main() {
    // Create a new chain with order 2
    chain := markov.NewChain(2)

    // Train the chain
    chain.RawAdd("hello world")
    chain.RawAdd("hello there")
    chain.RawAdd("world peace")

    // Generate a sequence
    tokens := make([]string, 0)
    for i := 0; i < chain.Order; i++ {
        tokens = append(tokens, markov.StartToken)
    }

    for len(tokens) < 20 && tokens[len(tokens)-1] != markov.EndToken {
        next, _ := chain.Generate(tokens[len(tokens)-chain.Order:])
        tokens = append(tokens, next)
    }

    // Extract the generated text
    result := strings.Join(tokens[chain.Order:len(tokens)-1], "")
    fmt.Println(result)
}
```

## Creating a Chain

### NewChain

Create a new Markov chain with specified order.

```go
// Order 1: next token depends on 1 previous token
chain1 := markov.NewChain(1)

// Order 2: next token depends on 2 previous tokens (more coherent)
chain2 := markov.NewChain(2)

// Order 3: next token depends on 3 previous tokens (even more coherent)
chain3 := markov.NewChain(3)
```

**Order Trade-offs:**

| Order | Memory | Coherence | Creativity | Best For |
|-------|--------|-----------|------------|----------|
| 1 | Low | Low | High | Random text, simple patterns |
| 2 | Medium | Medium | Medium | Names, short phrases |
| 3 | High | High | Low | Sentences, structured text |
| 5+ | Very High | Very High | Very Low | Exact reproduction |

## Training the Chain

### Add

Add a sequence of tokens to the chain.

```go
chain := markov.NewChain(2)

// Add token sequences
chain.Add([]string{"hello", "world"})
chain.Add([]string{"hello", "there"})
chain.Add([]string{"world", "peace"})
```

### RawAdd

Add a string to the chain (splits into characters).

```go
chain := markov.NewChain(2)

// Add strings (automatically split into characters)
chain.RawAdd("hello")
chain.RawAdd("world")
chain.RawAdd("helloworld")
```

**Character-level Example:**
```go
chain := markov.NewChain(3)

names := []string{
    "Pikachu", "Charizard", "Bulbasaur",
    "Squirtle", "Mewtwo", "Eevee",
}

for _, name := range names {
    chain.RawAdd(name)
}

// Generate a new name
tokens := make([]string, 0)
for i := 0; i < chain.Order; i++ {
    tokens = append(tokens, markov.StartToken)
}

for len(tokens) < 20 && tokens[len(tokens)-1] != markov.EndToken {
    next, _ := chain.Generate(tokens[len(tokens)-chain.Order:])
    tokens = append(tokens, next)
}

name := strings.Join(tokens[chain.Order:len(tokens)-1], "")
fmt.Println(name) // e.g., "Pikichu", "Chartle", etc.
```

**Word-level Example:**
```go
chain := markov.NewChain(2)

sentences := []string{
    "The quick brown fox jumps over the lazy dog",
    "The lazy cat sleeps all day long",
    "The quick cat jumps very high",
}

for _, sentence := range sentences {
    words := strings.Fields(sentence)
    chain.Add(words)
}

// Generate a new sentence
tokens := make([]string, 0)
for i := 0; i < chain.Order; i++ {
    tokens = append(tokens, markov.StartToken)
}

for len(tokens) < 50 && tokens[len(tokens)-1] != markov.EndToken {
    next, _ := chain.Generate(tokens[len(tokens)-chain.Order:])
    tokens = append(tokens, next)
}

sentence := strings.Join(tokens[chain.Order:len(tokens)-1], " ")
fmt.Println(sentence) // e.g., "The quick cat sleeps all day long"
```

## Generating Sequences

### Generate

Generate the next token based on the current state (n-gram).

```go
chain := markov.NewChain(2)
chain.RawAdd("hello")
chain.RawAdd("world")

// Start with start tokens
ngram := markov.NGram{markov.StartToken, markov.StartToken}

// Generate next token
next, err := chain.Generate(ngram)
if err != nil {
    panic(err)
}

fmt.Println(next) // Likely "h" (first character of training data)
```

**Complete Generation Loop:**
```go
func generateText(chain *markov.Chain, maxLength int) string {
    tokens := make([]string, 0)

    // Initialize with start tokens
    for i := 0; i < chain.Order; i++ {
        tokens = append(tokens, markov.StartToken)
    }

    // Generate until end token or max length
    for len(tokens) < maxLength && tokens[len(tokens)-1] != markov.EndToken {
        // Get current n-gram (last N tokens)
        ngram := tokens[len(tokens)-chain.Order:]

        // Generate next token
        next, err := chain.Generate(ngram)
        if err != nil {
            break
        }

        tokens = append(tokens, next)
    }

    // Remove start and end tokens
    return strings.Join(tokens[chain.Order:len(tokens)-1], "")
}
```

## Probability Queries

### TransitionProbability

Get the probability of transitioning to a specific token from a given state.

```go
chain := markov.NewChain(2)

// Train with some data
chain.Add([]string{"a", "b", "c"})
chain.Add([]string{"a", "b", "c"})
chain.Add([]string{"a", "b", "d"})

// Query probability
ngram := markov.NGram{markov.StartToken, markov.StartToken}
prob, err := chain.TransitionProbability("a", ngram)
if err != nil {
    panic(err)
}

fmt.Printf("Probability: %.2f\n", prob) // 1.00 (100%)

// More complex example
ngram2 := markov.NGram{"a", "b"}
probC, _ := chain.TransitionProbability("c", ngram2)
probD, _ := chain.TransitionProbability("d", ngram2)

fmt.Printf("P(c|a,b) = %.2f\n", probC) // 0.67 (66.7%)
fmt.Printf("P(d|a,b) = %.2f\n", probD) // 0.33 (33.3%)
```

## Start and End Tokens

The chain automatically adds start (`^`) and end (`$`) tokens to mark sequence boundaries.

```go
fmt.Println(markov.StartToken) // "^"
fmt.Println(markov.EndToken)   // "$"
```

**How it works:**
```go
chain.RawAdd("hi")
// Internally stored as: [^, ^, h, i, $, $] (for order 2)
```

## Use Cases

### Name Generator

Generate fantasy/sci-fi names based on training data.

```go
chain := markov.NewChain(3)

// Train with fantasy names
names := []string{
    "Aragorn", "Gandalf", "Frodo", "Legolas",
    "Gimli", "Boromir", "Elrond", "Galadriel",
}

for _, name := range names {
    chain.RawAdd(name)
}

// Generate new names
for i := 0; i < 5; i++ {
    name := generateText(chain, 20)
    fmt.Println(name)
}
// Output examples: "Gimrond", "Arolas", "Framir", etc.
```

### Text Generator

Generate sentences based on training corpus.

```go
chain := markov.NewChain(2)

// Train with sentences
corpus := `
The quick brown fox jumps over the lazy dog.
The lazy cat sleeps all day long.
The brown dog runs in the park.
The quick cat jumps very high.
`

for _, line := range strings.Split(corpus, "\n") {
    if line = strings.TrimSpace(line); line != "" {
        words := strings.Fields(line)
        chain.Add(words)
    }
}

// Generate new sentences
for i := 0; i < 3; i++ {
    sentence := generateText(chain, 50)
    fmt.Println(sentence)
}
```

### Predictive Text

Simple autocomplete/next word prediction.

```go
chain := markov.NewChain(2)

// Train with user's text history
chain.Add(strings.Fields("I love programming in Go"))
chain.Add(strings.Fields("I love writing code"))
chain.Add(strings.Fields("Programming is fun"))

// Predict next word after "I love"
ngram := markov.NGram{"I", "love"}
next, _ := chain.Generate(ngram)
fmt.Println(next) // Likely "programming" or "writing"
```

### Music Generation

Generate note sequences for melodies.

```go
chain := markov.NewChain(3)

// Train with musical phrases (MIDI note numbers or note names)
melodies := [][]string{
    {"C4", "D4", "E4", "F4", "G4"},
    {"C4", "E4", "G4", "C5"},
    {"G4", "F4", "E4", "D4", "C4"},
}

for _, melody := range melodies {
    chain.Add(melody)
}

// Generate new melody
notes := make([]string, 0)
for i := 0; i < chain.Order; i++ {
    notes = append(notes, markov.StartToken)
}

for len(notes) < 20 && notes[len(notes)-1] != markov.EndToken {
    next, _ := chain.Generate(notes[len(notes)-chain.Order:])
    notes = append(notes, next)
}

fmt.Println(notes[chain.Order:len(notes)-1])
```

## Thread Safety

All operations are thread-safe and can be used concurrently.

```go
chain := markov.NewChain(2)

// Concurrent training
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()
        chain.RawAdd(fmt.Sprintf("message %d", id))
    }(i)
}
wg.Wait()

// Concurrent generation
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        defer wg.Done()
        ngram := markov.NGram{markov.StartToken, markov.StartToken}
        chain.Generate(ngram)
    }()
}
wg.Wait()
```

## Persistence

The chain can be serialized and deserialized for storage and transfer.

### Export

Save a trained chain to a file.

```go
chain := markov.NewChain(2)

// Train the chain
chain.RawAdd("hello world")
chain.RawAdd("hello there")

// Export to bytes
data, err := chain.Export()
if err != nil {
    panic(err)
}

// Save to file
if err := os.WriteFile("chain.dat", data, 0644); err != nil {
    panic(err)
}
```

### Import

Load a previously saved chain.

```go
// Load from file
data, err := os.ReadFile("chain.dat")
if err != nil {
    panic(err)
}

// Import chain
chain, err := markov.ImportChain(data)
if err != nil {
    panic(err)
}

// Use the restored chain
ngram := markov.NGram{markov.StartToken, markov.StartToken}
next, _ := chain.Generate(ngram)
fmt.Println(next)
```

### Use Cases

**Pre-trained Models:**
```go
// Train once
chain := markov.NewChain(3)
for _, text := range largeCorpus {
    chain.RawAdd(text)
}

// Save for reuse
data, _ := chain.Export()
os.WriteFile("pretrained.dat", data, 0644)

// Later, load instantly
data, _ := os.ReadFile("pretrained.dat")
chain, _ := markov.ImportChain(data)
// Ready to generate immediately
```

**Backup and Restore:**
```go
// Periodically backup chain state
ticker := time.NewTicker(1 * time.Hour)
go func() {
    for range ticker.C {
        data, _ := chain.Export()
        os.WriteFile("backup.dat", data, 0644)
    }
}()
```

**Transfer Between Services:**
```go
// Service A: Train and export
chain := markov.NewChain(2)
// ... training ...
data, _ := chain.Export()
sendToService(data)

// Service B: Import and use
data := receiveFromService()
chain, _ := markov.ImportChain(data)
// ... generate ...
```

## Performance

### Benchmarks (Apple M3 Pro)

| Operation | Time | Allocations |
|-----------|------|-------------|
| NewChain | ~28 ns | 0 allocs |
| Add (3 tokens) | ~745 ns | 12 allocs |
| Add (100 tokens) | ~11 µs | 112 allocs |
| Generate (Order 2) | ~158 ns | 1 alloc |
| TransitionProbability | ~104 ns | 1 alloc |
| ConcurrentGenerate | ~156 ns | 1 alloc |
| Export | ~5.5 µs | 94 allocs |
| Import | ~16.3 µs | 281 allocs |

### Memory Usage

- **Order 1**: ~50-100 bytes per unique token
- **Order 2**: ~100-200 bytes per unique bigram
- **Order 3**: ~200-400 bytes per unique trigram

### Optimization Tips

1. **Choose appropriate order**: Higher order = more memory but better coherence
2. **Reuse chains**: Create once, generate many times
3. **Batch training**: Add all training data before generation
4. **Use word-level**: For text generation, use word tokens instead of characters

## Advanced Usage

### Custom Token Splitting

```go
// Split by words
func addSentence(chain *markov.Chain, sentence string) {
    words := strings.Fields(sentence)
    chain.Add(words)
}

// Split by custom delimiter
func addCSV(chain *markov.Chain, line string) {
    tokens := strings.Split(line, ",")
    chain.Add(tokens)
}

// Split by characters
func addText(chain *markov.Chain, text string) {
    chars := strings.Split(text, "")
    chain.Add(chars)
}
```

### Controlling Generation Length

```go
func generateWithLength(chain *markov.Chain, minLen, maxLen int) string {
    tokens := make([]string, 0)

    // Initialize
    for i := 0; i < chain.Order; i++ {
        tokens = append(tokens, markov.StartToken)
    }

    // Generate
    for len(tokens) < maxLen {
        ngram := tokens[len(tokens)-chain.Order:]
        next, err := chain.Generate(ngram)
        if err != nil {
            break
        }

        tokens = append(tokens, next)

        // Stop if we hit end token and have minimum length
        if next == markov.EndToken && len(tokens) >= minLen+chain.Order {
            break
        }
    }

    return strings.Join(tokens[chain.Order:len(tokens)-1], "")
}
```

### Filtering Output

```go
func generateValid(chain *markov.Chain, validator func(string) bool, maxAttempts int) string {
    for i := 0; i < maxAttempts; i++ {
        result := generateText(chain, 50)
        if validator(result) {
            return result
        }
    }
    return ""
}

// Example: generate name between 5-10 characters
name := generateValid(chain, func(s string) bool {
    return len(s) >= 5 && len(s) <= 10
}, 100)
```

## Error Handling

The chain returns descriptive errors:

```go
chain := markov.NewChain(2)
chain.Add([]string{"a", "b", "c"})

// Wrong n-gram length
_, err := chain.Generate(markov.NGram{"a"})
// err: "n-gram length does not match chain order"

// Unknown n-gram
_, err = chain.Generate(markov.NGram{"x", "y"})
// err: "unknown ngram [x y]"

// Wrong probability query
_, err = chain.TransitionProbability("z", markov.NGram{"a"})
// err: "n-gram length does not match chain order"
```

## Best Practices

### 1. Choose the Right Order

```go
// Character-level name generation: order 2-3
nameChain := markov.NewChain(3)

// Word-level text generation: order 1-2
textChain := markov.NewChain(2)

// Code generation: order 3-4
codeChain := markov.NewChain(3)
```

### 2. Provide Sufficient Training Data

```go
// Bad: too little data
chain := markov.NewChain(3)
chain.RawAdd("hi")
// Not enough patterns to generate interesting text

// Good: sufficient data
chain := markov.NewChain(3)
for _, name := range hundredsOfNames {
    chain.RawAdd(name)
}
```

### 3. Validate Generated Output

```go
func isValidName(name string) bool {
    // Check length
    if len(name) < 3 || len(name) > 12 {
        return false
    }

    // Check it starts with uppercase
    if !unicode.IsUpper(rune(name[0])) {
        return false
    }

    // Check for valid characters
    for _, c := range name {
        if !unicode.IsLetter(c) {
            return false
        }
    }

    return true
}
```

### 4. Handle Edge Cases

```go
func safeGenerate(chain *markov.Chain, maxLength int) (string, error) {
    tokens := make([]string, 0)

    for i := 0; i < chain.Order; i++ {
        tokens = append(tokens, markov.StartToken)
    }

    iterations := 0
    maxIterations := maxLength * 2 // Prevent infinite loops

    for len(tokens) < maxLength && iterations < maxIterations {
        ngram := tokens[len(tokens)-chain.Order:]

        next, err := chain.Generate(ngram)
        if err != nil {
            return "", err
        }

        tokens = append(tokens, next)

        if next == markov.EndToken {
            break
        }

        iterations++
    }

    if iterations >= maxIterations {
        return "", fmt.Errorf("generation exceeded maximum iterations")
    }

    return strings.Join(tokens[chain.Order:len(tokens)-1], ""), nil
}
```

## Examples

### Complete Name Generator

```go
package main

import (
    "fmt"
    "strings"
    "github.com/vitalvas/gokit/markov"
)

func main() {
    chain := markov.NewChain(3)

    // Training data
    names := []string{
        "Alexander", "Benjamin", "Christopher", "Daniel",
        "Elizabeth", "Gabriella", "Isabella", "Jennifer",
        "Katherine", "Maximilian", "Nathaniel", "Sebastian",
    }

    for _, name := range names {
        chain.RawAdd(name)
    }

    // Generate 10 new names
    fmt.Println("Generated names:")
    for i := 0; i < 10; i++ {
        name := generateName(chain)
        if len(name) >= 4 && len(name) <= 12 {
            fmt.Printf("%d. %s\n", i+1, name)
        }
    }
}

func generateName(chain *markov.Chain) string {
    tokens := make([]string, 0)

    for i := 0; i < chain.Order; i++ {
        tokens = append(tokens, markov.StartToken)
    }

    for len(tokens) < 30 && tokens[len(tokens)-1] != markov.EndToken {
        ngram := tokens[len(tokens)-chain.Order:]
        next, err := chain.Generate(ngram)
        if err != nil {
            break
        }
        tokens = append(tokens, next)
    }

    return strings.Join(tokens[chain.Order:len(tokens)-1], "")
}
```

## License

This project is part of the [gokit](https://github.com/vitalvas/gokit) library.
