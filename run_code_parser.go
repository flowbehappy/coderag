package main

import (
	"flag"
	"fmt"
	"strings"

	. "github.com/flow/coderag/codeparser"
)

func main() {
	// Parse command line options
	repoPath := flag.String("repo", "./", "Path to the repository to analyze")
	onlyRepo := flag.Bool("only-repo", false, "Only include packages from this repository")
	outputPath := flag.String("output", "graph.html", "Output path for the graph HTML file")
	layoutType := flag.String("layout", "force", "Graph layout type: 'force' (default) or 'circular'")

	// Define a custom flag for node types
	var nodeTypes nodeTypeFlag
	flag.Var(&nodeTypes, "node-type", "Types of nodes to include. Can be: file, package, struct, function, comment, or all. This flag can be specified multiple times.")

	flag.Parse()

	// Create parsing options
	options := DefaultParseOptions()
	options.OnlyRepoPackages = *onlyRepo

	// Apply node type filters if specified
	if len(nodeTypes) > 0 {
		// If any node types are specified, start with none included
		options.IncludeNodeTypes = make(map[NodeType]bool)

		// Add each specified node type
		for _, nodeType := range nodeTypes {
			switch strings.ToLower(nodeType) {
			case "all":
				options.IncludeNodeTypes[NodeTypeFile] = true
				options.IncludeNodeTypes[NodeTypeStruct] = true
				options.IncludeNodeTypes[NodeTypeFunc] = true
				options.IncludeNodeTypes[NodeTypeComment] = true
				options.IncludeNodeTypes[NodeTypePackage] = true
			case "file":
				options.IncludeNodeTypes[NodeTypeFile] = true
			case "package":
				options.IncludeNodeTypes[NodeTypePackage] = true
			case "struct":
				options.IncludeNodeTypes[NodeTypeStruct] = true
			case "function":
				options.IncludeNodeTypes[NodeTypeFunc] = true
			case "comment":
				options.IncludeNodeTypes[NodeTypeComment] = true
			default:
				fmt.Printf("Warning: Unknown node type '%s'\n", nodeType)
			}
		}
	}

	// Parse the repository
	graph, err := ParseRepo(*repoPath, options)
	if err != nil {
		panic(err)
	}

	// Print the graph stats
	println("Graph statistics:")
	PrintGraph(graph)

	// Determine layout type
	layout := GraphLayoutForce
	if *layoutType == "circular" {
		layout = GraphLayoutCircular
	} else if *layoutType != "force" {
		fmt.Printf("Warning: Unknown layout type '%s', using 'force' layout\n", *layoutType)
	}

	// Generate the graph as an HTML file
	if err := DrawGraphInHTML(graph, *outputPath, layout); err != nil {
		panic(err)
	}

	println("Graph generated successfully at", *outputPath)
}

// nodeTypeFlag is a custom flag type for handling multiple node type flags
type nodeTypeFlag []string

func (f *nodeTypeFlag) String() string {
	return strings.Join(*f, ", ")
}

func (f *nodeTypeFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}
