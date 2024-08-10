package main

import (
	"fmt"

	"github.com/vitalvas/gokit/uxid"
)

func main() {
	id := uxid.New().String()

	fmt.Println("id:", id)
	fmt.Println("id len:", len(id))
}
