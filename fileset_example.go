package main

import (
	"fmt"
	"go/token"
)

func main() {
	// Create a new FileSet
	fset := token.NewFileSet()

	// Add a file to the FileSet
	file := fset.AddFile("example.go", fset.Base(), 100)

	// Add line offsets to the file
	file.AddLine(10)
	file.AddLine(50)

	// Get the position of a specific offset
	pos := file.Pos(20)
	position := fset.Position(pos)

	// Print the position
	fmt.Println("Position:", position)
}
