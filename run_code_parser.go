package main

import (
	. "github.com/flow/coderag/codeparser"
)

func main() {
	repoPath := "./" // Replace with the path to your repo
	graph, err := ParseRepo(repoPath)
	if err != nil {
		panic(err)
	}

	// Print the original graph stats
	println("Original graph statistics:")
	PrintGraph(graph)

	// Uncomment to filter the graph to a specific number of nodes
	// filteredGraph := filterGraph(graph, 10)
	// PrintGraph(filteredGraph)

	// Print the filtered graph stats
	println("\nFiltered graph statistics:")
	PrintGraph(graph)

	// Generate the graph as an HTML file
	outputPath := "graph.html"
	if err := DrawGraphInHTML(graph, outputPath); err != nil {
		panic(err)
	}

	println("Graph generated successfully at", outputPath)
}
