package codeblock

type BlockType int

const (
	SingleFile BlockType = 10000
	MultiFile  BlockType = 10001
	CodeSnip   BlockType = 10002

	Code BlockType = 20000
	Doc  BlockType = 20001
	Test BlockType = 20002
)

// 128-bit hash of the code block
type BlockKey struct {
	High int64
	Low  int64
}

// The SubPath is used to identify a block of code in a file.
type SubPath struct {
	Path      string // For example: api.go or llm/api.go
	LineStart int    // The line start of the code block
	LineCount int    // The number of lines in the code block
}

// The CodePath is used to identify a block of code
type CodePath struct {
	Key BlockKey // Unique hash of the code block. It is calculated based on the following fields.

	Repo string // The repository name
	// The path to the code.
	// Example:
	// 	github.com/llimllib/llm/api.go (single file or code snippet)
	// 	github.com/llimllib/llm (multiple files)
	Path    string
	SubPath []SubPath // The sub path of the code block

	Extra string // Extra information, optional
}

// The CodeBlock is used to store a block of code.
// A block could be a single file, multiple files, or a code snippet.
type CodeBlock struct {
	CodePath
	Summary       string      // Tight summary of the block
	Skeleton      string      // The skeleton of the block
	Description   string      // Detailed description of the block
	FileName      []string    // The file name, could be multiple
	Reference     []CodePath  // Reference to other blocks, for example import statements
	Backreference []CodePath  // Known references to other blocks
	Type          []BlockType // The types of the block
	Code          []string    // The code block content
	Vector        []float64   // The vector representation of the block
}
