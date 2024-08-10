package uxid

import (
	"math/rand"
	"strings"
	"time"
)

type ID struct {
	ts   int64
	rand int64
}

const encodeDict = "0123456789abcdefghjkmnopqrstuvwxyz"

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func New() ID {
	return NewWithTime(time.Now())
}

func NewWithTime(t time.Time) ID {
	return ID{
		ts:   t.UTC().UnixNano(),
		rand: rand.Int63(),
	}
}

func (id ID) String() string {
	return encode(id.ts) + encode(id.rand)
}

func encode(n int64) string {
	var chars = make([]string, 12, 12)

	encodeDictSize := int64(len(encodeDict))

	for i := len(chars) - 1; i >= 0; i-- {
		index := n % encodeDictSize
		chars[i] = string(encodeDict[index])
		n = n / encodeDictSize
	}

	return strings.Join(chars, "")
}
