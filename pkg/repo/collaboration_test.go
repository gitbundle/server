// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package repo_test

import (
	"testing"

	"github.com/gitbundle/server/pkg/db"
	"github.com/gitbundle/server/pkg/perm"
	access_model "github.com/gitbundle/server/pkg/perm/access"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestRepository_GetCollaborators(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	test := func(repoID int64) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID}).(*repo_model.Repository)
		collaborators, err := repo_model.GetCollaborators(db.DefaultContext, repo.ID, db.ListOptions{})
		assert.NoError(t, err)
		expectedLen, err := db.GetEngine(db.DefaultContext).Count(&repo_model.Collaboration{RepoID: repoID})
		assert.NoError(t, err)
		assert.Len(t, collaborators, int(expectedLen))
		for _, collaborator := range collaborators {
			assert.EqualValues(t, collaborator.User.ID, collaborator.Collaboration.UserID)
			assert.EqualValues(t, repoID, collaborator.Collaboration.RepoID)
		}
	}
	test(1)
	test(2)
	test(3)
	test(4)
}

func TestRepository_IsCollaborator(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	test := func(repoID, userID int64, expected bool) {
		repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: repoID}).(*repo_model.Repository)
		actual, err := repo_model.IsCollaborator(db.DefaultContext, repo.ID, userID)
		assert.NoError(t, err)
		assert.Equal(t, expected, actual)
	}
	test(3, 2, true)
	test(3, unittest.NonexistentID, false)
	test(4, 2, false)
	test(4, 4, true)
}

func TestRepository_ChangeCollaborationAccessMode(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 4}).(*repo_model.Repository)
	assert.NoError(t, repo_model.ChangeCollaborationAccessMode(repo, 4, perm.AccessModeAdmin))

	collaboration := unittest.AssertExistsAndLoadBean(t, &repo_model.Collaboration{RepoID: repo.ID, UserID: 4}).(*repo_model.Collaboration)
	assert.EqualValues(t, perm.AccessModeAdmin, collaboration.Mode)

	access := unittest.AssertExistsAndLoadBean(t, &access_model.Access{UserID: 4, RepoID: repo.ID}).(*access_model.Access)
	assert.EqualValues(t, perm.AccessModeAdmin, access.Mode)

	assert.NoError(t, repo_model.ChangeCollaborationAccessMode(repo, 4, perm.AccessModeAdmin))

	assert.NoError(t, repo_model.ChangeCollaborationAccessMode(repo, unittest.NonexistentID, perm.AccessModeAdmin))

	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: repo.ID})
}
