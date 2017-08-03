package main

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
	"github.com/payneio/config"
)

type lStruct struct {
	A int
}

var l []lStruct

func main() {
	config.Load()

	fmt.Printf("A: %s\n", config.Get("a"))
	fmt.Printf("B: %s\n", config.Get("b"))
	fmt.Printf("C: %s\n", config.Get("c"))
	fmt.Printf("D: %s\n", config.Get("d"))
	fmt.Printf("E: %s\n", config.Get("e"))

	lGoo := config.GetAny("l")
	mapstructure.Decode(lGoo, &l)
	for _, i := range l {
		fmt.Printf("L.A: %d\n", i.A)
	}

	fmt.Printf("L: %s\n", config.Get("sub:h"))
	fmt.Printf("F:X: %s\n", config.Get("f:x"))
	fmt.Printf("Sub.G: %s\n", config.Get("sub:g"))
	fmt.Printf("Sub.H: %s\n", config.Get("sub:h"))
	fmt.Printf("deep.deeper.deepest: %s\n", config.Get("deep:deeper:deepest"))
	fmt.Printf("Version: %s\n", config.Get("version"))

	fmt.Println("\n\n\n")
	fmt.Println(config.ToYAML())
	fmt.Println(config.GetAll())
}
