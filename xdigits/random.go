package xdigits

import (
	"math/rand"
	"time"
)

func RandInt(minValue, maxValue int) int {
	randSource := rand.NewSource(time.Now().UnixNano())
	r := rand.New(randSource)
	return r.Intn(maxValue-minValue) + minValue
}
