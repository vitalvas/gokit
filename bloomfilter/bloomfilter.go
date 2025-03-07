package bloomfilter

import (
	"bytes"
	"encoding/gob"
	"hash"
	"hash/fnv"
	"math"
	"sync"
)

type BloomFilter struct {
	Bitset []byte `json:"bitset"`
	Size   uint   `json:"size"`
	K      uint   `json:"k"`

	hashFuncs []hash.Hash64
	lock      sync.RWMutex
}

func NewBloomFilter(n uint, p float64) *BloomFilter {
	m := optimalM(n, p)
	k := optimalK(n, m)

	hashFuncs := make([]hash.Hash64, k)

	for i := 0; i < int(k); i++ {
		hashFuncs[i] = fnv.New64()
	}

	return &BloomFilter{
		Bitset:    make([]byte, m),
		Size:      m,
		K:         k,
		hashFuncs: hashFuncs,
	}
}

func (bf *BloomFilter) Add(element string) {
	bf.lock.Lock()
	defer bf.lock.Unlock()

	data := []byte(element)

	for _, hashFunc := range bf.hashFuncs {
		hashFunc.Reset()
		hashFunc.Write(data)
		index := hashFunc.Sum64() % uint64(bf.Size)

		bf.Bitset[index] = 1
	}
}

func (bf *BloomFilter) Contains(element string) bool {
	bf.lock.RLock()
	defer bf.lock.RUnlock()

	data := []byte(element)

	for _, hashFunc := range bf.hashFuncs {
		hashFunc.Reset()
		hashFunc.Write(data)
		index := hashFunc.Sum64() % uint64(bf.Size)

		if bf.Bitset[index] == 0 {
			return false
		}
	}
	return true
}

func (bf *BloomFilter) Export() ([]byte, error) {
	bf.lock.RLock()
	defer bf.lock.RUnlock()

	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	err := encoder.Encode(bf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func ImportBloomFilter(data []byte) (*BloomFilter, error) {
	var bf BloomFilter
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	err := decoder.Decode(&bf)
	if err != nil {
		return nil, err
	}

	bf.hashFuncs = make([]hash.Hash64, bf.K)
	for i := 0; i < int(bf.K); i++ {
		bf.hashFuncs[i] = fnv.New64()
	}

	return &bf, nil
}

// optimalM calculates the optimal number of bits
func optimalM(n uint, p float64) uint {
	return uint(math.Ceil(float64(n) * math.Abs(math.Log(p)) / math.Pow(math.Log(2), 2)))
}

// optimalK calculates the optimal number of hash functions
func optimalK(n, m uint) uint {
	return uint(math.Round(float64(m) / float64(n) * math.Log(2)))
}
