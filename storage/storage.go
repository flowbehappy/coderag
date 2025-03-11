package storage

import (
	. "github.com/flow/coderag/codeblock"
)

type CodeBlockStorage interface {
	Put(codeBlock CodeBlock) error
	Search(vector []float64) ([]CodeBlock, error)
}
