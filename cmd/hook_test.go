// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package cmd

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPktLine(t *testing.T) {
	// test read
	s := strings.NewReader("0000")
	r := bufio.NewReader(s)
	result, err := readPktLine(r, pktLineTypeFlush)
	assert.NoError(t, err)
	assert.Equal(t, pktLineTypeFlush, result.Type)

	s = strings.NewReader("0006a\n")
	r = bufio.NewReader(s)
	result, err = readPktLine(r, pktLineTypeData)
	assert.NoError(t, err)
	assert.Equal(t, pktLineTypeData, result.Type)
	assert.Equal(t, []byte("a\n"), result.Data)

	// test write
	w := bytes.NewBuffer([]byte{})
	err = writeFlushPktLine(w)
	assert.NoError(t, err)
	assert.Equal(t, []byte("0000"), w.Bytes())

	w.Reset()
	err = writeDataPktLine(w, []byte("a\nb"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("0007a\nb"), w.Bytes())
}
