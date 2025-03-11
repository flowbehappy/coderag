package llm

import (
	. "github.com/flow/coderag/codeblock"
)

type AnalyzeAPI interface {
	// AnalyzeSingle analyzes a single file or a code snippet.
	// The path is the path to the code.
	// The code is the content of the code.
	AnalyzeSingle(path CodePath, code string) (CodeBlock, error)
	// AnalyzeMulti analyzes multiple files.
	// The codeBlocks are already analyzed single files or code snippets.
	// And we need to analyze them together to get a abstract view of the code.
	AnalyzeMulti(codeBlocks []CodeBlock) (CodeBlock, error)
}
