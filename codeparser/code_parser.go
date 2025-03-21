package codeparser

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// NodeType represents the type of a node in the graph.
type NodeType string

const (
	NodeTypeFile    NodeType = "File"
	NodeTypeStruct  NodeType = "Struct"
	NodeTypeFunc    NodeType = "Function"
	NodeTypeComment NodeType = "Comment"
	NodeTypeOther   NodeType = "Other"
)

// EdgeType represents the type of an edge in the graph.
type EdgeType string

const (
	EdgeTypeInvoke      EdgeType = "Invoke"
	EdgeTypeContain     EdgeType = "Contain"
	EdgeTypeImport      EdgeType = "Import"
	EdgeTypeOwnership   EdgeType = "Ownership"
	EdgeTypeEncapsulate EdgeType = "Encapsulate"
)

// Node represents a node in the graph.
type Node struct {
	ID   string    // Unique identifier for the node
	Type NodeType  // Type of the node
	Name string    // Name of the node
	File string    // File where the node is defined
	Pos  token.Pos // Position in the file
}

// Edge represents an edge in the graph.
type Edge struct {
	From string   // ID of the source node
	To   string   // ID of the target node
	Type EdgeType // Type of the edge
}

// Graph represents the entire graph.
type Graph struct {
	Nodes map[string]*Node // Map of node ID to Node
	Edges []Edge           // List of edges
}

// ParseRepo parses all Go files in a repository and builds a graph.
func ParseRepo(repoPath string) (*Graph, error) {
	graph := &Graph{
		Nodes: make(map[string]*Node),
		Edges: []Edge{},
	}

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".go" {
			if err := parseFile(path, graph); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return graph, nil
}

// parseFile parses a single Go file and updates the graph.
func parseFile(filePath string, graph *Graph) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return err
	}

	// Add the file node
	fileNodeID := "file:" + filePath
	graph.Nodes[fileNodeID] = &Node{
		ID:   fileNodeID,
		Type: NodeTypeFile,
		Name: filepath.Base(filePath),
		File: filePath,
	}

	// Track the current function for detecting invocations
	var currentFunc string

	// Traverse the AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.IMPORT {
				for _, spec := range x.Specs {
					importSpec := spec.(*ast.ImportSpec)
					importPath := strings.Trim(importSpec.Path.Value, `"`)
					importNodeID := "import:" + importPath
					if _, exists := graph.Nodes[importNodeID]; !exists {
						graph.Nodes[importNodeID] = &Node{
							ID:   importNodeID,
							Type: NodeTypeOther,
							Name: importPath,
						}
					}
					graph.Edges = append(graph.Edges, Edge{
						From: fileNodeID,
						To:   importNodeID,
						Type: EdgeTypeImport,
					})
				}
			}
		case *ast.FuncDecl:
			funcNodeID := "func:" + x.Name.Name
			graph.Nodes[funcNodeID] = &Node{
				ID:   funcNodeID,
				Type: NodeTypeFunc,
				Name: x.Name.Name,
				File: filePath,
				Pos:  x.Pos(),
			}
			graph.Edges = append(graph.Edges, Edge{
				From: funcNodeID,
				To:   fileNodeID,
				Type: EdgeTypeEncapsulate,
			})

			// Set current function for tracking invocations
			currentFunc = funcNodeID

			if x.Recv != nil {
				for _, field := range x.Recv.List {
					if starExpr, ok := field.Type.(*ast.StarExpr); ok {
						if ident, ok := starExpr.X.(*ast.Ident); ok {
							structNodeID := "struct:" + ident.Name
							if _, exists := graph.Nodes[structNodeID]; !exists {
								graph.Nodes[structNodeID] = &Node{
									ID:   structNodeID,
									Type: NodeTypeStruct,
									Name: ident.Name,
									File: filePath,
								}
							}
							graph.Edges = append(graph.Edges, Edge{
								From: funcNodeID,
								To:   structNodeID,
								Type: EdgeTypeOwnership,
							})
						}
					}
				}
			}
		case *ast.CallExpr:
			// Skip if we're not inside a function
			if currentFunc == "" {
				break
			}

			// Determine the function being called
			var calledFunc string

			switch fun := x.Fun.(type) {
			case *ast.Ident:
				// Direct function call: funcName()
				calledFunc = "func:" + fun.Name
			case *ast.SelectorExpr:
				// Method or package function call: pkg.funcName() or obj.method()
				calledFunc = "func:" + fun.Sel.Name
			}

			// Add an edge if we identified a called function
			if calledFunc != "" && calledFunc != currentFunc {
				// Check if the called function exists (it might be in another file)
				if _, exists := graph.Nodes[calledFunc]; !exists {
					// Create a placeholder node for the function
					graph.Nodes[calledFunc] = &Node{
						ID:   calledFunc,
						Type: NodeTypeFunc,
						Name: strings.TrimPrefix(calledFunc, "func:"),
						File: "unknown", // Function might be defined in another file
					}
				}

				// Add the invocation edge
				graph.Edges = append(graph.Edges, Edge{
					From: currentFunc,
					To:   calledFunc,
					Type: EdgeTypeInvoke,
				})
			}
		case *ast.TypeSpec:
			if _, ok := x.Type.(*ast.StructType); ok {
				structNodeID := "struct:" + x.Name.Name
				graph.Nodes[structNodeID] = &Node{
					ID:   structNodeID,
					Type: NodeTypeStruct,
					Name: x.Name.Name,
					File: filePath,
					Pos:  x.Pos(),
				}
				graph.Edges = append(graph.Edges, Edge{
					From: structNodeID,
					To:   fileNodeID,
					Type: EdgeTypeEncapsulate,
				})
			}
		case *ast.CommentGroup:
			commentNodeID := "comment:" + filePath + ":" + strconv.Itoa(int(x.Pos()))
			graph.Nodes[commentNodeID] = &Node{
				ID:   commentNodeID,
				Type: NodeTypeComment,
				Name: strings.Join(getCommentText(x), " "),
				File: filePath,
				Pos:  x.Pos(),
			}
			graph.Edges = append(graph.Edges, Edge{
				From: commentNodeID,
				To:   fileNodeID,
				Type: EdgeTypeEncapsulate,
			})
		}
		return true
	})

	return nil
}

// getCommentText extracts the text from a CommentGroup.
func getCommentText(cg *ast.CommentGroup) []string {
	var comments []string
	for _, comment := range cg.List {
		comments = append(comments, comment.Text)
	}
	return comments
}
