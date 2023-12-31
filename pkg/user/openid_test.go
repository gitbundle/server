// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package user_test

import (
	"testing"

	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestGetUserOpenIDs(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	oids, err := user_model.GetUserOpenIDs(int64(1))
	if assert.NoError(t, err) && assert.Len(t, oids, 2) {
		assert.Equal(t, "https://user1.domain1.tld/", oids[0].URI)
		assert.False(t, oids[0].Show)
		assert.Equal(t, "http://user1.domain2.tld/", oids[1].URI)
		assert.True(t, oids[1].Show)
	}

	oids, err = user_model.GetUserOpenIDs(int64(2))
	if assert.NoError(t, err) && assert.Len(t, oids, 1) {
		assert.Equal(t, "https://domain1.tld/user2/", oids[0].URI)
		assert.True(t, oids[0].Show)
	}
}

func TestToggleUserOpenIDVisibility(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	oids, err := user_model.GetUserOpenIDs(int64(2))
	if !assert.NoError(t, err) || !assert.Len(t, oids, 1) {
		return
	}
	assert.True(t, oids[0].Show)

	err = user_model.ToggleUserOpenIDVisibility(oids[0].ID)
	if !assert.NoError(t, err) {
		return
	}

	oids, err = user_model.GetUserOpenIDs(int64(2))
	if !assert.NoError(t, err) || !assert.Len(t, oids, 1) {
		return
	}
	assert.False(t, oids[0].Show)
	err = user_model.ToggleUserOpenIDVisibility(oids[0].ID)
	if !assert.NoError(t, err) {
		return
	}

	oids, err = user_model.GetUserOpenIDs(int64(2))
	if !assert.NoError(t, err) {
		return
	}
	if assert.Len(t, oids, 1) {
		assert.True(t, oids[0].Show)
	}
}
