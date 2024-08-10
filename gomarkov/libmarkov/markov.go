package libmarkov

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
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

// ChainJSON godoc
type ChainJSON struct {
	Order    int                 `json:"int"`
	SpoolMap map[string]int      `json:"spool_map"`
	FreqMat  map[int]sparseArray `json:"freq_mat"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// MarshalJSON godoc
func (chain Chain) MarshalJSON() ([]byte, error) {
	obj := ChainJSON{
		Order:    chain.Order,
		SpoolMap: chain.statePool.stringMap,
		FreqMat:  chain.frequencyMat,
	}
	return json.Marshal(obj)
}

// UnmarshalJSON godoc
func (chain *Chain) UnmarshalJSON(b []byte) error {
	var obj ChainJSON
	err := json.Unmarshal(b, &obj)
	if err != nil {
		return err
	}

	chain.Order = obj.Order
	intMap := make(map[int]string)

	for k, v := range obj.SpoolMap {
		intMap[v] = k
	}
	chain.statePool = &spool{
		stringMap: obj.SpoolMap,
		intMap:    intMap,
	}
	chain.frequencyMat = obj.FreqMat
	chain.lock = new(sync.RWMutex)
	return nil
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

	for i := 0; i < len(pairs); i++ {
		pair := pairs[i]
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
		return 0, errors.New("N-gram length does not match chain order")
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
		return "", errors.New("N-gram length does not match chain order")
	}

	currentIndex, currentExists := chain.statePool.get(current.key())
	if !currentExists {
		return "", fmt.Errorf("Unknown ngram %v", current)
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
