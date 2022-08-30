package main

import (
	"os"

	"github.com/wanabe/link_doc/model"
)

func main() {
	dir := "."
	if len(os.Args) >= 2 {
		dir = os.Args[1]
	}
	linker, err := model.NewLinker(dir)
	if err != nil {
		panic(err)
	}

	err = linker.LinkDocs()
	if err != nil {
		panic(err)
	}
}
