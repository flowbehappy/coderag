package codeparser

import (
	"os"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
)

// DrawGraphInHTML generates an HTML visualization of the graph
func DrawGraphInHTML(graph *Graph, outputPath string) error {
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
			Width:    "100%",
			Height:   "90vh", // 90% of viewport height
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
		// Add legend for edge types
		// charts.WithLegendOpts(opts.Legend{
		// 	Show:   opts.Bool(true),
		// 	Orient: "vertical",
		// 	Left:   "left",
		// 	Data:   edgeCategoryNames(edgeCategories),
		// }),
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
			Name:       id,
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
						Repulsion:  20000, // Increased to spread nodes further
						Gravity:    0.02,  // Decreased to reduce clustering
						EdgeLength: 300,   // Increased for more spacing between nodes
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

// Helper function to extract category names for legend
func edgeCategoryNames(categories []*opts.GraphCategory) []string {
	names := make([]string, len(categories))
	for i, category := range categories {
		names[i] = category.Name
	}
	return names
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
