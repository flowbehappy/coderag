package main

import (
	"go/parser"
	"go/printer"
	"go/token"
	"os"
)

func main() {
	// Example Go code to parse
	src := `
		package main
		import "fmt"
		func main() {
			fmt.Println("Hello, World!")
		}
	`

	// Create a new token file set
	fset := token.NewFileSet()

	// Parse the source code
	node, err := parser.ParseFile(fset, "example.go", src, parser.AllErrors)
	if err != nil {
		panic(err)
	}

	// Print the AST
	printer.Fprint(os.Stdout, fset, node)
}
