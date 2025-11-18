//go:build ignore

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("hello")
	for _, a := range os.Args {
		fmt.Println(a)
	}
}
