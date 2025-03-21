package codeparser

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
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
	NodeTypePackage NodeType = "Package"
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
	Nodes   map[string]*Node // Map of node ID to Node
	Edges   []Edge           // List of edges
	edgeMap map[string]bool  // Map for O(1) edge lookup
}

// AddEdge adds an edge to the graph if it doesn't already exist
func (g *Graph) AddEdge(from, to string, edgeType EdgeType) {
	// Create a unique edge signature
	edgeKey := from + "|" + to + "|" + string(edgeType)

	// Check if the edge already exists in O(1) time
	if g.edgeMap == nil {
		g.edgeMap = make(map[string]bool)
	}

	if g.edgeMap[edgeKey] {
		return // Edge already exists, don't add a duplicate
	}

	// Add the new edge
	g.Edges = append(g.Edges, Edge{
		From: from,
		To:   to,
		Type: edgeType,
	})

	// Mark this edge as existing
	g.edgeMap[edgeKey] = true
}

// ParseOptions represents options for parsing a repository
type ParseOptions struct {
	// OnlyRepoPackages when true, only includes packages that are part of the repository
	// and excludes external dependencies and standard library
	OnlyRepoPackages bool

	// IncludeNodeTypes specifies which node types to include in the graph
	// If empty, all node types are included
	IncludeNodeTypes map[NodeType]bool
}

// DefaultParseOptions returns the default parsing options
func DefaultParseOptions() ParseOptions {
	return ParseOptions{
		OnlyRepoPackages: false,
		IncludeNodeTypes: map[NodeType]bool{
			NodeTypeFile:    true,
			NodeTypeStruct:  true,
			NodeTypeFunc:    true,
			NodeTypeComment: true,
			NodeTypePackage: true,
		},
	}
}

// ShouldIncludeNodeType checks if a node type should be included based on the options
func (opts ParseOptions) ShouldIncludeNodeType(nodeType NodeType) bool {
	// If no node types are specified, include all types
	if len(opts.IncludeNodeTypes) == 0 {
		return true
	}

	// Otherwise, check if this type is in the map
	return opts.IncludeNodeTypes[nodeType]
}

// Cache for module paths
var modulePathCache = make(map[string]string)

// getModulePath returns the Go module path for the specified repository path
func getModulePath(repoPath string) string {
	// Check if we've already determined the module path for this repo
	if modulePath, ok := modulePathCache[repoPath]; ok {
		return modulePath
	}

	// Look for go.mod file in the repository
	goModPath := filepath.Join(repoPath, "go.mod")
	file, err := os.Open(goModPath)
	if err != nil {
		// If go.mod doesn't exist, return empty string
		modulePathCache[repoPath] = ""
		return ""
	}
	defer file.Close()

	// Regular expression to extract module path
	moduleRegex := regexp.MustCompile(`^module\s+(.+)$`)

	// Scan the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := moduleRegex.FindStringSubmatch(line)
		if len(matches) == 2 {
			modulePath := strings.TrimSpace(matches[1])
			modulePathCache[repoPath] = modulePath
			return modulePath
		}
	}

	// If module declaration not found
	modulePathCache[repoPath] = ""
	return ""
}

// isInRepo checks if a package path is within the repository
func isInRepo(pkgPath string, repoPath string) bool {
	// Get the module path for this repository
	modulePath := getModulePath(repoPath)

	// If we couldn't determine the module path, use a simpler heuristic
	if modulePath == "" {
		// Standard library packages don't contain a dot or slash
		if !strings.Contains(pkgPath, ".") && !strings.Contains(pkgPath, "/") {
			return false
		}

		// Packages directly under the repository path are considered part of the repo
		// Strip any vendor directory from consideration
		relPath := strings.TrimPrefix(pkgPath, "vendor/")

		// This is a simple heuristic - packages with the same directory structure
		// as the repository might be considered part of it
		absRepoPath, _ := filepath.Abs(repoPath)
		absPkgPath, _ := filepath.Abs(relPath)
		return strings.HasPrefix(absPkgPath, absRepoPath)
	}

	// If we know the module path, check if the package path starts with it
	return strings.HasPrefix(pkgPath, modulePath) || pkgPath == modulePath
}

// getPackageFullPath returns the full path of a package based on its location in the repo
func getPackageFullPath(filePath, repoPath string, packageName string) string {
	// Get the module path
	modulePath := getModulePath(repoPath)
	if modulePath == "" {
		// If no module path found, just use the package name
		return packageName
	}

	// Convert absolute file path to path relative to repo root
	absFilePath, _ := filepath.Abs(filePath)
	absRepoPath, _ := filepath.Abs(repoPath)

	// Get the relative path within the repo
	relPath, err := filepath.Rel(absRepoPath, filepath.Dir(absFilePath))
	if err != nil {
		return packageName // Fallback to just the package name
	}

	// Handle special case for root package
	if relPath == "." {
		return modulePath
	}

	// Replace Windows backslashes with forward slashes for Go package paths
	relPath = strings.ReplaceAll(relPath, "\\", "/")

	// Construct the full package path
	return modulePath + "/" + relPath
}

// ParseRepo parses all Go files in a repository and builds a graph.
func ParseRepo(repoPath string, options ...ParseOptions) (*Graph, error) {
	// Apply options, using defaults if none provided
	opts := DefaultParseOptions()
	if len(options) > 0 {
		opts = options[0]
	}

	graph := &Graph{
		Nodes:   make(map[string]*Node),
		Edges:   []Edge{},
		edgeMap: make(map[string]bool),
	}

	// Get the absolute path of the repo for filtering purposes
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	err = filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if filepath.Ext(path) == ".go" {
			if err := parseFile(path, graph, absRepoPath, opts); err != nil {
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
func parseFile(filePath string, graph *Graph, repoPath string, opts ParseOptions) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return err
	}

	// Add the file node
	fileNodeID := "file:" + filePath
	if opts.ShouldIncludeNodeType(NodeTypeFile) {
		graph.Nodes[fileNodeID] = &Node{
			ID:   fileNodeID,
			Type: NodeTypeFile,
			Name: filepath.Base(filePath),
			File: filePath,
		}
	}

	// Current package
	var currentPackageID string

	// Add the package node and connect it to the file
	if node.Name != nil {
		packageName := node.Name.Name

		// Get the full package path
		fullPackagePath := getPackageFullPath(filePath, repoPath, packageName)
		packageNodeID := "package:" + fullPackagePath
		currentPackageID = packageNodeID

		// Check if the package node already exists
		if opts.ShouldIncludeNodeType(NodeTypePackage) {
			if _, exists := graph.Nodes[packageNodeID]; !exists {
				graph.Nodes[packageNodeID] = &Node{
					ID:   packageNodeID,
					Type: NodeTypePackage,
					Name: fullPackagePath, // Use full path as name
					File: "",              // Package doesn't belong to a single file
				}
			}

			// Connect file to package
			if opts.ShouldIncludeNodeType(NodeTypeFile) {
				graph.AddEdge(fileNodeID, packageNodeID, EdgeTypeContain)
			}
		}
	}

	// Track the current function for detecting invocations
	var currentFunc string

	// Traverse the AST
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.GenDecl:
			if x.Tok == token.IMPORT && currentPackageID != "" {
				for _, spec := range x.Specs {
					importSpec := spec.(*ast.ImportSpec)
					importPath := strings.Trim(importSpec.Path.Value, `"`)

					// Skip external packages if OnlyRepoPackages is true
					if opts.OnlyRepoPackages && !isInRepo(importPath, repoPath) {
						continue
					}

					// Use the full import path instead of just the last component
					importedPkgName := importPath

					// Handle aliased imports
					if importSpec.Name != nil {
						// If an import has an alias, we still use the full path as its identifier
						// Skip dot imports (.) and blank imports (_)
						if importSpec.Name.Name == "." || importSpec.Name.Name == "_" {
							continue
						}
					}

					importedPackageID := "package:" + importedPkgName

					// Create the imported package node if it doesn't exist and if package nodes are included
					if opts.ShouldIncludeNodeType(NodeTypePackage) {
						if _, exists := graph.Nodes[importedPackageID]; !exists {
							graph.Nodes[importedPackageID] = &Node{
								ID:   importedPackageID,
								Type: NodeTypePackage,
								Name: importedPkgName, // Use full path
								File: "",
							}
						}

						// Create edges if the relevant node types are included
						if opts.ShouldIncludeNodeType(NodeTypeFile) {
							graph.AddEdge(fileNodeID, importedPackageID, EdgeTypeImport)
						}
						graph.AddEdge(currentPackageID, importedPackageID, EdgeTypeImport)
					}
				}
			}
		case *ast.FuncDecl:
			if opts.ShouldIncludeNodeType(NodeTypeFunc) {
				funcNodeID := "func:" + x.Name.Name
				graph.Nodes[funcNodeID] = &Node{
					ID:   funcNodeID,
					Type: NodeTypeFunc,
					Name: x.Name.Name,
					File: filePath,
					Pos:  x.Pos(),
				}

				if opts.ShouldIncludeNodeType(NodeTypeFile) {
					graph.AddEdge(funcNodeID, fileNodeID, EdgeTypeEncapsulate)
				}

				// Set current function for tracking invocations
				currentFunc = funcNodeID

				if x.Recv != nil && opts.ShouldIncludeNodeType(NodeTypeStruct) {
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
								graph.AddEdge(funcNodeID, structNodeID, EdgeTypeOwnership)
							}
						}
					}
				}
			}
		case *ast.CallExpr:
			// Skip if we're not inside a function or if function nodes are not included
			if currentFunc == "" || !opts.ShouldIncludeNodeType(NodeTypeFunc) {
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
				graph.AddEdge(currentFunc, calledFunc, EdgeTypeInvoke)
			}
		case *ast.TypeSpec:
			if _, ok := x.Type.(*ast.StructType); ok && opts.ShouldIncludeNodeType(NodeTypeStruct) {
				structNodeID := "struct:" + x.Name.Name
				graph.Nodes[structNodeID] = &Node{
					ID:   structNodeID,
					Type: NodeTypeStruct,
					Name: x.Name.Name,
					File: filePath,
					Pos:  x.Pos(),
				}

				if opts.ShouldIncludeNodeType(NodeTypeFile) {
					graph.AddEdge(structNodeID, fileNodeID, EdgeTypeEncapsulate)
				}
			}
		case *ast.CommentGroup:
			if opts.ShouldIncludeNodeType(NodeTypeComment) {
				commentNodeID := "comment:" + filePath + ":" + strconv.Itoa(int(x.Pos()))
				graph.Nodes[commentNodeID] = &Node{
					ID:   commentNodeID,
					Type: NodeTypeComment,
					Name: strings.Join(getCommentText(x), " "),
					File: filePath,
					Pos:  x.Pos(),
				}

				if opts.ShouldIncludeNodeType(NodeTypeFile) {
					graph.AddEdge(commentNodeID, fileNodeID, EdgeTypeEncapsulate)
				}
			}
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
