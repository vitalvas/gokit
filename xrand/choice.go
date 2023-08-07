package xrand

import "math/rand"

func ChoiceInt(list []int) int {
	randomIndex := rand.Intn(len(list))
	return list[randomIndex]
}
