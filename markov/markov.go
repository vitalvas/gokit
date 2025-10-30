package markov

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
)

// StartToken godoc
const StartToken = "^"

// EndToken godoc
const EndToken = "$"

// Chain godoc
type Chain struct {
	Order        int
	statePool    *spool
	frequencyMat map[int]sparseArray
	lock         *sync.RWMutex
}

// NewChain godoc
func NewChain(order int) *Chain {
	chain := Chain{Order: order}
	chain.statePool = &spool{
		stringMap: make(map[string]int),
		intMap:    make(map[int]string),
	}
	chain.frequencyMat = make(map[int]sparseArray, 0)
	chain.lock = new(sync.RWMutex)
	return &chain
}

// RawAdd godoc
func (chain *Chain) RawAdd(input string) {
	split := func(str string) []string {
		return strings.Split(str, "")
	}

	chain.Add(split(input))
}

// Add godoc
func (chain *Chain) Add(input []string) {
	startTokens := array(StartToken, chain.Order)
	endTokens := array(EndToken, chain.Order)
	tokens := make([]string, 0)
	tokens = append(tokens, startTokens...)
	tokens = append(tokens, input...)
	tokens = append(tokens, endTokens...)
	pairs := MakePairs(tokens, chain.Order)

	for _, pair := range pairs {
		currentIndex := chain.statePool.add(pair.CurrentState.key())
		nextIndex := chain.statePool.add(pair.NextState)
		chain.lock.Lock()

		if chain.frequencyMat[currentIndex] == nil {
			chain.frequencyMat[currentIndex] = make(sparseArray, 0)
		}

		chain.frequencyMat[currentIndex][nextIndex]++
		chain.lock.Unlock()
	}
}

// TransitionProbability godoc
func (chain *Chain) TransitionProbability(next string, current NGram) (float64, error) {
	if len(current) != chain.Order {
		return 0, errors.New("n-gram length does not match chain order")
	}

	currentIndex, currentExists := chain.statePool.get(current.key())
	nextIndex, nextExists := chain.statePool.get(next)
	if !currentExists || !nextExists {
		return 0, nil
	}

	arr := chain.frequencyMat[currentIndex]
	sum := float64(arr.sum())
	freq := float64(arr[nextIndex])

	return freq / sum, nil
}

// Generate godoc
func (chain *Chain) Generate(current NGram) (string, error) {
	if len(current) != chain.Order {
		return "", errors.New("n-gram length does not match chain order")
	}

	currentIndex, currentExists := chain.statePool.get(current.key())
	if !currentExists {
		return "", fmt.Errorf("unknown ngram %v", current)
	}

	arr := chain.frequencyMat[currentIndex]
	sum := arr.sum()
	randN := rand.Intn(sum)

	for i, freq := range arr {
		randN -= freq
		if randN <= 0 {
			return chain.statePool.intMap[i], nil
		}
	}
	return "", nil
}

// Export serializes the chain to bytes using gob encoding.
func (chain *Chain) Export() ([]byte, error) {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	exportData := struct {
		Order        int
		StringMap    map[string]int
		FrequencyMat map[int]sparseArray
	}{
		Order:        chain.Order,
		StringMap:    chain.statePool.stringMap,
		FrequencyMat: chain.frequencyMat,
	}

	if err := encoder.Encode(exportData); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// ImportChain deserializes a chain from bytes.
func ImportChain(data []byte) (*Chain, error) {
	var exportData struct {
		Order        int
		StringMap    map[string]int
		FrequencyMat map[int]sparseArray
	}

	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)

	if err := decoder.Decode(&exportData); err != nil {
		return nil, err
	}

	intMap := make(map[int]string)
	for k, v := range exportData.StringMap {
		intMap[v] = k
	}

	return &Chain{
		Order: exportData.Order,
		statePool: &spool{
			stringMap: exportData.StringMap,
			intMap:    intMap,
		},
		frequencyMat: exportData.FrequencyMat,
		lock:         new(sync.RWMutex),
	}, nil
}
