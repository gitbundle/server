// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package bleve

import (
	"github.com/blevesearch/bleve/v2"
)

// FlushingBatch is a batch of operations that automatically flushes to the
// underlying index once it reaches a certain size.
type FlushingBatch struct {
	maxBatchSize int
	batch        *bleve.Batch
	index        bleve.Index
}

// NewFlushingBatch creates a new flushing batch for the specified index. Once
// the number of operations in the batch reaches the specified limit, the batch
// automatically flushes its operations to the index.
func NewFlushingBatch(index bleve.Index, maxBatchSize int) *FlushingBatch {
	return &FlushingBatch{
		maxBatchSize: maxBatchSize,
		batch:        index.NewBatch(),
		index:        index,
	}
}

// Index add a new index to batch
func (b *FlushingBatch) Index(id string, data interface{}) error {
	if err := b.batch.Index(id, data); err != nil {
		return err
	}
	return b.flushIfFull()
}

// Delete add a delete index to batch
func (b *FlushingBatch) Delete(id string) error {
	b.batch.Delete(id)
	return b.flushIfFull()
}

func (b *FlushingBatch) flushIfFull() error {
	if b.batch.Size() < b.maxBatchSize {
		return nil
	}
	return b.Flush()
}

// Flush submit the batch and create a new one
func (b *FlushingBatch) Flush() error {
	err := b.index.Batch(b.batch)
	if err != nil {
		return err
	}
	b.batch = b.index.NewBatch()
	return nil
}
