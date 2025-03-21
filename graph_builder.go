package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
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

func generateGraphHTML(graph *Graph, outputPath string) error {
	// Print debug information first
	println("Preparing to generate HTML with", len(graph.Nodes), "nodes and", len(graph.Edges), "edges")

	// Create a new page
	page := components.NewPage()
	page.PageTitle = "Go Code Structure Graph"
	page.AssetsHost = "" // Use local assets

	// Define categories explicitly
	categories := []*opts.GraphCategory{
		{Name: string(NodeTypeFile)},
		{Name: string(NodeTypeStruct)},
		{Name: string(NodeTypeFunc)},
		{Name: string(NodeTypeComment)},
		{Name: string(NodeTypeOther)},
	}

	// Create a new graph chart with full-page dimensions
	graphChart := charts.NewGraph()
	graphChart.SetGlobalOptions(
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "100%",
			Height: "90vh", // 100% of viewport height
			// PageTitle: "Go Code Structure Graph",
			// BackgroundColor: &opts.BackgroundColor{
			// 	Color: "rgba(255, 255, 255, 1)",
			// },
			// Set renderer to canvas for better performance with large graphs
			Renderer: "canvas",
		}),
		// Add toolbox for navigation options
		charts.WithToolboxOpts(opts.Toolbox{
			Show:  opts.Bool(true),
			Right: "20px",
			Feature: &opts.ToolBoxFeature{
				SaveAsImage: &opts.ToolBoxFeatureSaveAsImage{
					Show:  opts.Bool(true),
					Title: "Save as Image",
				},
				Restore: &opts.ToolBoxFeatureRestore{
					Show:  opts.Bool(true),
					Title: "Reset",
				},
				DataView: &opts.ToolBoxFeatureDataView{
					Show:  opts.Bool(true),
					Title: "Data View",
				},
			},
		}),
	)

	// Convert nodes to a map for quick lookup
	nodeMap := make(map[string]struct{})
	for id := range graph.Nodes {
		nodeMap[id] = struct{}{}
	}

	// Important: create nodes with unique names
	nodes := make([]opts.GraphNode, 0, len(graph.Nodes))
	for id, node := range graph.Nodes {
		// Use shorter names for readability
		displayName := node.Name
		if len(displayName) > 30 {
			displayName = displayName[:30] + "..."
		}

		nodes = append(nodes, opts.GraphNode{
			Name: id,
			// Value:      displayName,
			Category:   string(node.Type), // Use the defined categories
			SymbolSize: nodeSize(node.Type),
			ItemStyle: &opts.ItemStyle{
				Color: nodeColor(node.Type),
			},
		})
	}

	// Check if edges reference valid nodes
	validEdges := make([]opts.GraphLink, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		if _, fromExists := nodeMap[edge.From]; !fromExists {
			println("Warning: Edge references non-existent source node:", edge.From)
			continue
		}
		if _, toExists := nodeMap[edge.To]; !toExists {
			println("Warning: Edge references non-existent target node:", edge.To)
			continue
		}
		validEdges = append(validEdges, opts.GraphLink{
			Source: edge.From,
			Target: edge.To,
			Label: &opts.EdgeLabel{
				Show:      opts.Bool(true),
				Formatter: string(edge.Type),
			},
			LineStyle: &opts.LineStyle{
				Width:     edgeWidth(edge.Type),
				Curveness: 0.2, // Add some curvature to visualize overlapping edges
				Color:     edgeColor(edge.Type),
			},
		})
	}
	println("Valid edges:", len(validEdges), "out of", len(graph.Edges))

	// Add nodes and edges to the graph chart with improved visualization settings
	graphChart.AddSeries("graph", nodes, validEdges).
		SetSeriesOptions(
			charts.WithGraphChartOpts(
				opts.GraphChart{
					Layout: "force",
					Force: &opts.GraphForce{
						Repulsion:  20000, // Increased from 8000 to spread nodes further
						Gravity:    0.02,  // Decreased from 0.1 to reduce clustering
						EdgeLength: 300,   // Increased from 100 for more spacing between nodes
					},
					Roam:       opts.Bool(true), // Allow zooming and panning
					Categories: categories,
				}),
			charts.WithLabelOpts(opts.Label{
				Show:     opts.Bool(true),
				Position: "right",
				Color:    "black",
				FontSize: 12,
			}),
		)

	// Save the graph to an HTML file
	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	page.AddCharts(graphChart)
	return page.Render(f)
}

// Helper functions for styling

func nodeSize(nodeType NodeType) float32 {
	switch nodeType {
	case NodeTypeFile:
		return 30
	case NodeTypeStruct:
		return 25
	case NodeTypeFunc:
		return 20
	case NodeTypeComment:
		return 15
	default:
		return 18
	}
}

func nodeColor(nodeType NodeType) string {
	switch nodeType {
	case NodeTypeFile:
		return "#1f77b4" // blue
	case NodeTypeStruct:
		return "#ff7f0e" // orange
	case NodeTypeFunc:
		return "#2ca02c" // green
	case NodeTypeComment:
		return "#d62728" // red
	default:
		return "#9467bd" // purple
	}
}

func edgeWidth(edgeType EdgeType) float32 {
	switch edgeType {
	case EdgeTypeInvoke:
		return 2.5
	case EdgeTypeContain:
		return 2
	case EdgeTypeImport:
		return 1.5
	case EdgeTypeOwnership:
		return 2
	case EdgeTypeEncapsulate:
		return 1
	default:
		return 1
	}
}

func edgeColor(edgeType EdgeType) string {
	switch edgeType {
	case EdgeTypeInvoke:
		return "#1f77b4" // blue
	case EdgeTypeContain:
		return "#ff7f0e" // orange
	case EdgeTypeImport:
		return "#2ca02c" // green
	case EdgeTypeOwnership:
		return "#d62728" // red
	case EdgeTypeEncapsulate:
		return "#9467bd" // purple
	default:
		return "#7f7f7f" // gray
	}
}

// PrintGraph prints the graph in text format for debugging purposes.
func PrintGraph(graph *Graph) {
	// Print statistics
	println("Graph Statistics:")
	println("Number of Nodes:", len(graph.Nodes))
	println("Number of Edges:", len(graph.Edges))
	println()

	// Print node details
	println("Nodes:")
	for _, node := range graph.Nodes {
		println("ID:", node.ID, "Type:", node.Type, "Name:", node.Name, "File:", node.File)
	}
	println()

	// Print edge details
	println("Edges:")
	for _, edge := range graph.Edges {
		println("From:", edge.From, "To:", edge.To, "Type:", edge.Type)
	}
	println()
}

// filterGraph reduces the graph to a manageable size for visualization
func filterGraph(graph *Graph, maxNodes int) *Graph {
	// Create a new filtered graph
	filtered := &Graph{
		Nodes: make(map[string]*Node),
		Edges: []Edge{},
	}

	// First, collect file nodes as they're the most important
	fileNodes := []*Node{}
	for id, node := range graph.Nodes {
		if node.Type == NodeTypeFile {
			fileNodes = append(fileNodes, node)
			filtered.Nodes[id] = node
			if len(filtered.Nodes) >= maxNodes {
				break
			}
		}
	}

	// If we still have room, add some struct nodes
	if len(filtered.Nodes) < maxNodes {
		for id, node := range graph.Nodes {
			if node.Type == NodeTypeStruct {
				filtered.Nodes[id] = node
				if len(filtered.Nodes) >= maxNodes {
					break
				}
			}
		}
	}

	// If we still have room, add some function nodes
	if len(filtered.Nodes) < maxNodes {
		for id, node := range graph.Nodes {
			if node.Type == NodeTypeFunc {
				filtered.Nodes[id] = node
				if len(filtered.Nodes) >= maxNodes {
					break
				}
			}
		}
	}

	// Add edges between the selected nodes
	for _, edge := range graph.Edges {
		if _, hasFrom := filtered.Nodes[edge.From]; hasFrom {
			if _, hasTo := filtered.Nodes[edge.To]; hasTo {
				filtered.Edges = append(filtered.Edges, edge)
			}
		}
	}

	return filtered
}

func main() {
	repoPath := "./" // Replace with the path to your repo
	graph, err := ParseRepo(repoPath)
	if err != nil {
		panic(err)
	}

	// Print the original graph stats
	println("Original graph statistics:")
	PrintGraph(graph)

	// Filter the graph to 10 nodes or fewer
	// filteredGraph := filterGraph(graph, 10)

	// Print the filtered graph stats
	println("\nFiltered graph statistics:")
	PrintGraph(graph)

	// Generate the graph as an HTML file using the filtered graph
	outputPath := "graph.html"
	if err := generateGraphHTML(graph, outputPath); err != nil {
		panic(err)
	}

	println("Graph generated successfully at", outputPath)
}
