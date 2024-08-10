package libmarkov

import "strings"

// Pair godoc
type Pair struct {
	CurrentState NGram
	NextState    string
}

// NGram godoc
type NGram []string

type sparseArray map[int]int

func (ngram NGram) key() string {
	return strings.Join(ngram, ":")
}

func (s sparseArray) sum() int {
	sum := 0
	for _, count := range s {
		sum += count
	}
	return sum
}

func array(value string, count int) []string {
	arr := make([]string, count)
	for i := range arr {
		arr[i] = value
	}
	return arr
}

// MakePairs godoc
func MakePairs(tokens []string, order int) []Pair {
	var pairs []Pair
	for i := 0; i < len(tokens)-order; i++ {
		pair := Pair{
			CurrentState: tokens[i : i+order],
			NextState:    tokens[i+order],
		}
		pairs = append(pairs, pair)
	}
	return pairs
}
