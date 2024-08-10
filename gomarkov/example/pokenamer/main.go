package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/vitalvas/gokit/gomarkov/libmarkov"
)

func main() {
	train := flag.Bool("train", false, "Train the markov chain")
	order := flag.Int("order", 3, "Chain order to use")
	sourceFile := flag.String("file", "names.txt", "File name with source")

	flag.Parse()
	if *train {
		chain := buildModel(*order, *sourceFile)
		saveModel(chain)
	} else {
		chain, err := loadModel()
		if err != nil {
			fmt.Println(err)
			return
		}

		name := generatePokemon(chain)
		fmt.Println(name)
	}
}

func buildModel(order int, file string) *libmarkov.Chain {
	chain := libmarkov.NewChain(order)
	for _, data := range getDataset(file) {
		if len(data) > 0 {
			chain.Add(split(data))
		}
	}
	return chain
}

func split(str string) []string {
	return strings.Split(str, "")
}

func getDataset(fileName string) []string {
	file, _ := os.Open(fileName)
	scanner := bufio.NewScanner(file)
	var list []string
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	return list
}

func saveModel(chain *libmarkov.Chain) {
	jsonObj, _ := json.Marshal(chain)
	err := ioutil.WriteFile("model.json", jsonObj, 0644)
	if err != nil {
		fmt.Println(err)
	}
}

func loadModel() (*libmarkov.Chain, error) {
	var chain libmarkov.Chain
	data, err := ioutil.ReadFile("model.json")
	if err != nil {
		return &chain, err
	}
	err = json.Unmarshal(data, &chain)
	if err != nil {
		return &chain, err
	}
	return &chain, nil
}

func generatePokemon(chain *libmarkov.Chain) string {
	order := chain.Order
	tokens := make([]string, 0)

	for i := 0; i < order; i++ {
		tokens = append(tokens, libmarkov.StartToken)
	}

	for tokens[len(tokens)-1] != libmarkov.EndToken {
		next, _ := chain.Generate(tokens[(len(tokens) - order):])
		tokens = append(tokens, next)
	}

	return strings.Join(tokens[order:len(tokens)-1], "")
}

func newAppendModel(name string) {
	f, err := os.OpenFile("names_auto.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = f.WriteString(name)
	if err != nil {
		log.Println(err.Error())
	}

	_, err = f.WriteString("\n")
	if err != nil {
		log.Println(err.Error())
	}
}
