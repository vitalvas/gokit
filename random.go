package gokit

import (
	"math/rand"
	"time"
)

func RandInt(min, max int) int {
	randSource := rand.NewSource(time.Now().UnixNano())
	r := rand.New(randSource)
	return r.Intn(max-min) + min
}
