// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package repo_test

import (
	"testing"

	"github.com/gitbundle/server/pkg/db"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestRepoAssignees(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2}).(*repo_model.Repository)
	users, err := repo_model.GetRepoAssignees(db.DefaultContext, repo2)
	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, users[0].ID, int64(2))

	repo21 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 21}).(*repo_model.Repository)
	users, err = repo_model.GetRepoAssignees(db.DefaultContext, repo21)
	assert.NoError(t, err)
	assert.Len(t, users, 3)
	assert.Equal(t, users[0].ID, int64(15))
	assert.Equal(t, users[1].ID, int64(18))
	assert.Equal(t, users[2].ID, int64(16))
}

func TestRepoGetReviewers(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// test public repo
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)

	ctx := db.DefaultContext
	reviewers, err := repo_model.GetReviewers(ctx, repo1, 2, 2)
	assert.NoError(t, err)
	assert.Len(t, reviewers, 4)

	// should include doer if doer is not PR poster.
	reviewers, err = repo_model.GetReviewers(ctx, repo1, 11, 2)
	assert.NoError(t, err)
	assert.Len(t, reviewers, 4)

	// should not include PR poster, if PR poster would be otherwise eligible
	reviewers, err = repo_model.GetReviewers(ctx, repo1, 11, 4)
	assert.NoError(t, err)
	assert.Len(t, reviewers, 3)

	// test private user repo
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2}).(*repo_model.Repository)

	reviewers, err = repo_model.GetReviewers(ctx, repo2, 2, 4)
	assert.NoError(t, err)
	assert.Len(t, reviewers, 1)
	assert.EqualValues(t, reviewers[0].ID, 2)

	// test private org repo
	repo3 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3}).(*repo_model.Repository)

	reviewers, err = repo_model.GetReviewers(ctx, repo3, 2, 1)
	assert.NoError(t, err)
	assert.Len(t, reviewers, 2)

	reviewers, err = repo_model.GetReviewers(ctx, repo3, 2, 2)
	assert.NoError(t, err)
	assert.Len(t, reviewers, 1)
}
