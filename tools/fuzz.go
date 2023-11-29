// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

//go:build gofuzz

package fuzz

import (
	"bytes"
	"context"
	"io"

	"github.com/gitbundle/modules/markup"
	"github.com/gitbundle/modules/markup/markdown"
	"github.com/gitbundle/modules/setting"
)

// Contains fuzzing functions executed by
// fuzzing engine https://github.com/dvyukov/go-fuzz
//
// The function must return 1 if the fuzzer should increase priority of the given input during subsequent fuzzing
// (for example, the input is lexically correct and was parsed successfully).
// -1 if the input must not be added to corpus even if gives new coverage and 0 otherwise.

var renderContext = markup.RenderContext{
	Ctx:       context.Background(),
	URLPrefix: "https://example.com/go-gitea/gitea",
	Metas: map[string]string{
		"user": "go-gitea",
		"repo": "gitea",
	},
}

func FuzzMarkdownRenderRaw(data []byte) int {
	setting.AppURL = "http://localhost:3000/"
	err := markdown.RenderRaw(&renderContext, bytes.NewReader(data), io.Discard)
	if err != nil {
		return 0
	}
	return 1
}

func FuzzMarkupPostProcess(data []byte) int {
	setting.AppURL = "http://localhost:3000/"
	err := markup.PostProcess(&renderContext, bytes.NewReader(data), io.Discard)
	if err != nil {
		return 0
	}
	return 1
}