package gokit

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandInt(min, max int) int {
	return rand.Intn(max-min) + min
}
