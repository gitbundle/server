// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package git_test

import (
	"testing"

	"github.com/gitbundle/server/pkg/db"
	git_model "github.com/gitbundle/server/pkg/git"
	issues_model "github.com/gitbundle/server/pkg/issues"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestAddDeletedBranch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)
	firstBranch := unittest.AssertExistsAndLoadBean(t, &git_model.DeletedBranch{ID: 1}).(*git_model.DeletedBranch)

	assert.Error(t, git_model.AddDeletedBranch(repo.ID, firstBranch.Name, firstBranch.Commit, firstBranch.DeletedByID))
	assert.NoError(t, git_model.AddDeletedBranch(repo.ID, "test", "5655464564554545466464656", int64(1)))
}

func TestGetDeletedBranches(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)

	branches, err := git_model.GetDeletedBranches(repo.ID)
	assert.NoError(t, err)
	assert.Len(t, branches, 2)
}

func TestGetDeletedBranch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	firstBranch := unittest.AssertExistsAndLoadBean(t, &git_model.DeletedBranch{ID: 1}).(*git_model.DeletedBranch)

	assert.NotNil(t, getDeletedBranch(t, firstBranch))
}

func TestDeletedBranchLoadUser(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	firstBranch := unittest.AssertExistsAndLoadBean(t, &git_model.DeletedBranch{ID: 1}).(*git_model.DeletedBranch)
	secondBranch := unittest.AssertExistsAndLoadBean(t, &git_model.DeletedBranch{ID: 2}).(*git_model.DeletedBranch)

	branch := getDeletedBranch(t, firstBranch)
	assert.Nil(t, branch.DeletedBy)
	branch.LoadUser()
	assert.NotNil(t, branch.DeletedBy)
	assert.Equal(t, "user1", branch.DeletedBy.Name)

	branch = getDeletedBranch(t, secondBranch)
	assert.Nil(t, branch.DeletedBy)
	branch.LoadUser()
	assert.NotNil(t, branch.DeletedBy)
	assert.Equal(t, "Ghost", branch.DeletedBy.Name)
}

func TestRemoveDeletedBranch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)

	firstBranch := unittest.AssertExistsAndLoadBean(t, &git_model.DeletedBranch{ID: 1}).(*git_model.DeletedBranch)

	err := git_model.RemoveDeletedBranchByID(repo.ID, 1)
	assert.NoError(t, err)
	unittest.AssertNotExistsBean(t, firstBranch)
	unittest.AssertExistsAndLoadBean(t, &git_model.DeletedBranch{ID: 2})
}

func getDeletedBranch(t *testing.T, branch *git_model.DeletedBranch) *git_model.DeletedBranch {
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)

	deletedBranch, err := git_model.GetDeletedBranchByID(repo.ID, branch.ID)
	assert.NoError(t, err)
	assert.Equal(t, branch.ID, deletedBranch.ID)
	assert.Equal(t, branch.Name, deletedBranch.Name)
	assert.Equal(t, branch.Commit, deletedBranch.Commit)
	assert.Equal(t, branch.DeletedByID, deletedBranch.DeletedByID)

	return deletedBranch
}

func TestFindRenamedBranch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	branch, exist, err := git_model.FindRenamedBranch(1, "dev")
	assert.NoError(t, err)
	assert.Equal(t, true, exist)
	assert.Equal(t, "master", branch.To)

	_, exist, err = git_model.FindRenamedBranch(1, "unknow")
	assert.NoError(t, err)
	assert.Equal(t, false, exist)
}

func TestRenameBranch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)
	_isDefault := false

	ctx, committer, err := db.TxContext()
	defer committer.Close()
	assert.NoError(t, err)
	assert.NoError(t, git_model.UpdateProtectBranch(ctx, repo1, &git_model.ProtectedBranch{
		RepoID:     repo1.ID,
		BranchName: "master",
	}, git_model.WhitelistOptions{}))
	assert.NoError(t, committer.Commit())

	assert.NoError(t, git_model.RenameBranch(repo1, "master", "main", func(isDefault bool) error {
		_isDefault = isDefault
		return nil
	}))

	assert.Equal(t, true, _isDefault)
	repo1 = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)
	assert.Equal(t, "main", repo1.DefaultBranch)

	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}).(*issues_model.PullRequest) // merged
	assert.Equal(t, "master", pull.BaseBranch)

	pull = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}).(*issues_model.PullRequest) // open
	assert.Equal(t, "main", pull.BaseBranch)

	renamedBranch := unittest.AssertExistsAndLoadBean(t, &git_model.RenamedBranch{ID: 2}).(*git_model.RenamedBranch)
	assert.Equal(t, "master", renamedBranch.From)
	assert.Equal(t, "main", renamedBranch.To)
	assert.Equal(t, int64(1), renamedBranch.RepoID)

	unittest.AssertExistsAndLoadBean(t, &git_model.ProtectedBranch{
		RepoID:     repo1.ID,
		BranchName: "main",
	})
}

func TestOnlyGetDeletedBranchOnCorrectRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Get deletedBranch with ID of 1 on repo with ID 2.
	// This should return a nil branch as this deleted branch
	// is actually on repo with ID 1.
	repo2 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2}).(*repo_model.Repository)

	deletedBranch, err := git_model.GetDeletedBranchByID(repo2.ID, 1)

	// Expect no error, and the returned branch is nil.
	assert.NoError(t, err)
	assert.Nil(t, deletedBranch)

	// Now get the deletedBranch with ID of 1 on repo with ID 1.
	// This should return the deletedBranch.
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)

	deletedBranch, err = git_model.GetDeletedBranchByID(repo1.ID, 1)

	// Expect no error, and the returned branch to be not nil.
	assert.NoError(t, err)
	assert.NotNil(t, deletedBranch)
}
