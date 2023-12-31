// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package repo_test

import (
	"testing"

	"github.com/gitbundle/modules/markup"
	"github.com/gitbundle/modules/util"
	"github.com/gitbundle/server/pkg/db"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/unit"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

var (
	countRepospts        = repo_model.CountRepositoryOptions{OwnerID: 10}
	countReposptsPublic  = repo_model.CountRepositoryOptions{OwnerID: 10, Private: util.OptionalBoolFalse}
	countReposptsPrivate = repo_model.CountRepositoryOptions{OwnerID: 10, Private: util.OptionalBoolTrue}
)

func TestGetRepositoryCount(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	ctx := db.DefaultContext
	count, err1 := repo_model.CountRepositories(ctx, countRepospts)
	privateCount, err2 := repo_model.CountRepositories(ctx, countReposptsPrivate)
	publicCount, err3 := repo_model.CountRepositories(ctx, countReposptsPublic)
	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.NoError(t, err3)
	assert.Equal(t, int64(3), count)
	assert.Equal(t, privateCount+publicCount, count)
}

func TestGetPublicRepositoryCount(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	count, err := repo_model.CountRepositories(db.DefaultContext, countReposptsPublic)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestGetPrivateRepositoryCount(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	count, err := repo_model.CountRepositories(db.DefaultContext, countReposptsPrivate)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

func TestRepoAPIURL(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 10}).(*repo_model.Repository)

	assert.Equal(t, "https://try.gitea.io/api/v1/repos/user12/repo10", repo.APIURL())
}

func TestWatchRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	const repoID = 3
	const userID = 2

	assert.NoError(t, repo_model.WatchRepo(db.DefaultContext, userID, repoID, true))
	unittest.AssertExistsAndLoadBean(t, &repo_model.Watch{RepoID: repoID, UserID: userID})
	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: repoID})

	assert.NoError(t, repo_model.WatchRepo(db.DefaultContext, userID, repoID, false))
	unittest.AssertNotExistsBean(t, &repo_model.Watch{RepoID: repoID, UserID: userID})
	unittest.CheckConsistencyFor(t, &repo_model.Repository{ID: repoID})
}

func TestMetas(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	repo := &repo_model.Repository{Name: "testRepo"}
	repo.Owner = &user_model.User{Name: "testOwner"}
	repo.OwnerName = repo.Owner.Name

	repo.Units = nil

	metas := repo.ComposeMetas()
	assert.Equal(t, "testRepo", metas["repo"])
	assert.Equal(t, "testOwner", metas["user"])

	externalTracker := repo_model.RepoUnit{
		Type: unit.TypeExternalTracker,
		Config: &repo_model.ExternalTrackerConfig{
			ExternalTrackerFormat: "https://someurl.com/{user}/{repo}/{issue}",
		},
	}

	testSuccess := func(expectedStyle string) {
		repo.Units = []*repo_model.RepoUnit{&externalTracker}
		repo.RenderingMetas = nil
		metas := repo.ComposeMetas()
		assert.Equal(t, expectedStyle, metas["style"])
		assert.Equal(t, "testRepo", metas["repo"])
		assert.Equal(t, "testOwner", metas["user"])
		assert.Equal(t, "https://someurl.com/{user}/{repo}/{issue}", metas["format"])
	}

	testSuccess(markup.IssueNameStyleNumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleAlphanumeric
	testSuccess(markup.IssueNameStyleAlphanumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleNumeric
	testSuccess(markup.IssueNameStyleNumeric)

	externalTracker.ExternalTrackerConfig().ExternalTrackerStyle = markup.IssueNameStyleRegexp
	testSuccess(markup.IssueNameStyleRegexp)

	repo, err := repo_model.GetRepositoryByID(3)
	assert.NoError(t, err)

	metas = repo.ComposeMetas()
	assert.Contains(t, metas, "org")
	assert.Contains(t, metas, "teams")
	assert.Equal(t, "user3", metas["org"])
	assert.Equal(t, ",owners,team1,", metas["teams"])
}
