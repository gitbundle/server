// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package files

import (
	"testing"

	api "github.com/gitbundle/api/pkg/structs"
	"github.com/gitbundle/modules/git"
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/test"
	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestCleanUploadFileName(t *testing.T) {
	t.Run("Clean regular file", func(t *testing.T) {
		name := "this/is/test"
		cleanName := CleanUploadFileName(name)
		expectedCleanName := name
		assert.EqualValues(t, expectedCleanName, cleanName)
	})

	t.Run("Clean a .git path", func(t *testing.T) {
		name := "this/is/test/.git"
		cleanName := CleanUploadFileName(name)
		expectedCleanName := ""
		assert.EqualValues(t, expectedCleanName, cleanName)
	})
}

func getExpectedFileResponse() *api.FileResponse {
	treePath := "README.md"
	sha := "4b4851ad51df6a7d9f25c979345979eaeb5b349f"
	encoding := "base64"
	content := "IyByZXBvMQoKRGVzY3JpcHRpb24gZm9yIHJlcG8x"
	selfURL := setting.AppURL + "api/v1/repos/user2/repo1/contents/" + treePath + "?ref=master"
	htmlURL := setting.AppURL + "user2/repo1/src/branch/master/" + treePath
	gitURL := setting.AppURL + "api/v1/repos/user2/repo1/git/blobs/" + sha
	downloadURL := setting.AppURL + "user2/repo1/raw/branch/master/" + treePath
	return &api.FileResponse{
		Content: &api.ContentsResponse{
			Name:        treePath,
			Path:        treePath,
			SHA:         sha,
			Type:        "file",
			Size:        30,
			Encoding:    &encoding,
			Content:     &content,
			URL:         &selfURL,
			HTMLURL:     &htmlURL,
			GitURL:      &gitURL,
			DownloadURL: &downloadURL,
			Links: &api.FileLinksResponse{
				Self:    &selfURL,
				GitURL:  &gitURL,
				HTMLURL: &htmlURL,
			},
		},
		Commit: &api.FileCommitResponse{
			CommitMeta: api.CommitMeta{
				URL: "https://try.gitea.io/api/v1/repos/user2/repo1/git/commits/65f1bf27bc3bf70f64657658635e66094edbcb4d",
				SHA: "65f1bf27bc3bf70f64657658635e66094edbcb4d",
			},
			HTMLURL: "https://try.gitea.io/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d",
			Author: &api.CommitUser{
				Identity: api.Identity{
					Name:  "user1",
					Email: "address1@example.com",
				},
				Date: "2017-03-19T20:47:59Z",
			},
			Committer: &api.CommitUser{
				Identity: api.Identity{
					Name:  "Ethan Koenig",
					Email: "ethantkoenig@gmail.com",
				},
				Date: "2017-03-19T20:47:59Z",
			},
			Parents: []*api.CommitMeta{},
			Message: "Initial commit\n",
			Tree: &api.CommitMeta{
				URL: "https://try.gitea.io/api/v1/repos/user2/repo1/git/trees/2a2f1d4670728a2e10049e345bd7a276468beab6",
				SHA: "2a2f1d4670728a2e10049e345bd7a276468beab6",
			},
		},
		Verification: &api.PayloadCommitVerification{
			Verified:  false,
			Reason:    "gpg.error.not_signed_commit",
			Signature: "",
			Payload:   "",
		},
	}
}

func TestGetFileResponseFromCommit(t *testing.T) {
	unittest.PrepareTestEnv(t)
	ctx := test.MockContext(t, "user2/repo1")
	ctx.SetParams(":id", "1")
	test.LoadRepo(t, ctx, 1)
	test.LoadRepoCommit(t, ctx)
	test.LoadUser(t, ctx, 2)
	test.LoadGitRepo(t, ctx)
	defer ctx.Repo.GitRepo.Close()

	repo := ctx.Repo.Repository
	branch := repo.DefaultBranch
	treePath := "README.md"
	gitRepo, _ := git.OpenRepository(ctx, repo.RepoPath())
	defer gitRepo.Close()
	commit, _ := gitRepo.GetBranchCommit(branch)
	expectedFileResponse := getExpectedFileResponse()

	fileResponse, err := GetFileResponseFromCommit(ctx, repo, commit, branch, treePath)
	assert.NoError(t, err)
	assert.EqualValues(t, expectedFileResponse, fileResponse)
}
