// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package appstate

import (
	"path/filepath"
	"testing"

	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	unittest.MainTest(m, &unittest.TestOptions{
		GiteaRootPath: filepath.Join("..", ".."),
		FixtureFiles:  []string{""}, // load nothing
	})
}

type testItem1 struct {
	Val1 string
	Val2 int
}

func (*testItem1) Name() string {
	return "test-item1"
}

type testItem2 struct {
	K string
}

func (*testItem2) Name() string {
	return "test-item2"
}

func TestAppStateDB(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	as := &DBStore{}

	item1 := new(testItem1)
	assert.NoError(t, as.Get(item1))
	assert.Equal(t, "", item1.Val1)
	assert.EqualValues(t, 0, item1.Val2)

	item1 = new(testItem1)
	item1.Val1 = "a"
	item1.Val2 = 2
	assert.NoError(t, as.Set(item1))

	item2 := new(testItem2)
	item2.K = "V"
	assert.NoError(t, as.Set(item2))

	item1 = new(testItem1)
	assert.NoError(t, as.Get(item1))
	assert.Equal(t, "a", item1.Val1)
	assert.EqualValues(t, 2, item1.Val2)

	item2 = new(testItem2)
	assert.NoError(t, as.Get(item2))
	assert.Equal(t, "V", item2.K)
}
