package index

import (
	. "github.com/flow/coderag/storage"
)

type Repository interface {
	FullIndex(store CodeBlockStorage) error
	IncrementalIndex(store CodeBlockStorage) error
}
