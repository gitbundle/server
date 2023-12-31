// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package issue_test

import (
	"testing"

	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/action"
	issues_model "github.com/gitbundle/server/pkg/issues"
	issue_service "github.com/gitbundle/server/pkg/issues/manager"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/repo/repository"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestUpdateIssuesCommit(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pushCommits := []*repository.PushCommit{
		{
			Sha1:           "abcdef1",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user4@example.com",
			AuthorName:     "User Four",
			Message:        "start working on #FST-1, #1",
		},
		{
			Sha1:           "abcdef2",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User Two",
			Message:        "a plain message",
		},
		{
			Sha1:           "abcdef2",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User Two",
			Message:        "close #2",
		},
	}

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)
	repo.Owner = user

	commentBean := &issues_model.Comment{
		Type:      issues_model.CommentTypeCommitRef,
		CommitSHA: "abcdef1",
		PosterID:  user.ID,
		IssueID:   1,
	}
	issueBean := &issues_model.Issue{RepoID: repo.ID, Index: 4}

	unittest.AssertNotExistsBean(t, commentBean)
	unittest.AssertNotExistsBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 2}, "is_closed=1")
	assert.NoError(t, issue_service.UpdateIssuesCommit(user, repo, pushCommits, repo.DefaultBranch))
	unittest.AssertExistsAndLoadBean(t, commentBean)
	unittest.AssertExistsAndLoadBean(t, issueBean, "is_closed=1")
	unittest.CheckConsistencyFor(t, &action.Action{})

	// Test that push to a non-default branch closes no issue.
	pushCommits = []*repository.PushCommit{
		{
			Sha1:           "abcdef1",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user4@example.com",
			AuthorName:     "User Four",
			Message:        "close #1",
		},
	}
	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3}).(*repo_model.Repository)
	commentBean = &issues_model.Comment{
		Type:      issues_model.CommentTypeCommitRef,
		CommitSHA: "abcdef1",
		PosterID:  user.ID,
		IssueID:   6,
	}
	issueBean = &issues_model.Issue{RepoID: repo.ID, Index: 1}

	unittest.AssertNotExistsBean(t, commentBean)
	unittest.AssertNotExistsBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1}, "is_closed=1")
	assert.NoError(t, issue_service.UpdateIssuesCommit(user, repo, pushCommits, "non-existing-branch"))
	unittest.AssertExistsAndLoadBean(t, commentBean)
	unittest.AssertNotExistsBean(t, issueBean, "is_closed=1")
	unittest.CheckConsistencyFor(t, &action.Action{})

	pushCommits = []*repository.PushCommit{
		{
			Sha1:           "abcdef3",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User Two",
			Message:        "close " + setting.AppURL + repo.FullName() + "/pulls/1",
		},
	}
	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 3}).(*repo_model.Repository)
	commentBean = &issues_model.Comment{
		Type:      issues_model.CommentTypeCommitRef,
		CommitSHA: "abcdef3",
		PosterID:  user.ID,
		IssueID:   6,
	}
	issueBean = &issues_model.Issue{RepoID: repo.ID, Index: 1}

	unittest.AssertNotExistsBean(t, commentBean)
	unittest.AssertNotExistsBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 1}, "is_closed=1")
	assert.NoError(t, issue_service.UpdateIssuesCommit(user, repo, pushCommits, repo.DefaultBranch))
	unittest.AssertExistsAndLoadBean(t, commentBean)
	unittest.AssertExistsAndLoadBean(t, issueBean, "is_closed=1")
	unittest.CheckConsistencyFor(t, &action.Action{})
}

func TestUpdateIssuesCommit_Colon(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pushCommits := []*repository.PushCommit{
		{
			Sha1:           "abcdef2",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User Two",
			Message:        "close: #2",
		},
	}

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)
	repo.Owner = user

	issueBean := &issues_model.Issue{RepoID: repo.ID, Index: 4}

	unittest.AssertNotExistsBean(t, &issues_model.Issue{RepoID: repo.ID, Index: 2}, "is_closed=1")
	assert.NoError(t, issue_service.UpdateIssuesCommit(user, repo, pushCommits, repo.DefaultBranch))
	unittest.AssertExistsAndLoadBean(t, issueBean, "is_closed=1")
	unittest.CheckConsistencyFor(t, &action.Action{})
}

func TestUpdateIssuesCommit_Issue5957(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)

	// Test that push to a non-default branch closes an issue.
	pushCommits := []*repository.PushCommit{
		{
			Sha1:           "abcdef1",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user4@example.com",
			AuthorName:     "User Four",
			Message:        "close #2",
		},
	}

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2}).(*repo_model.Repository)
	commentBean := &issues_model.Comment{
		Type:      issues_model.CommentTypeCommitRef,
		CommitSHA: "abcdef1",
		PosterID:  user.ID,
		IssueID:   7,
	}

	issueBean := &issues_model.Issue{RepoID: repo.ID, Index: 2, ID: 7}

	unittest.AssertNotExistsBean(t, commentBean)
	unittest.AssertNotExistsBean(t, issueBean, "is_closed=1")
	assert.NoError(t, issue_service.UpdateIssuesCommit(user, repo, pushCommits, "non-existing-branch"))
	unittest.AssertExistsAndLoadBean(t, commentBean)
	unittest.AssertExistsAndLoadBean(t, issueBean, "is_closed=1")
	unittest.CheckConsistencyFor(t, &action.Action{})
}

func TestUpdateIssuesCommit_AnotherRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)

	// Test that a push to default branch closes issue in another repo
	// If the user also has push permissions to that repo
	pushCommits := []*repository.PushCommit{
		{
			Sha1:           "abcdef1",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User Two",
			Message:        "close user2/repo1#1",
		},
	}

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2}).(*repo_model.Repository)
	commentBean := &issues_model.Comment{
		Type:      issues_model.CommentTypeCommitRef,
		CommitSHA: "abcdef1",
		PosterID:  user.ID,
		IssueID:   1,
	}

	issueBean := &issues_model.Issue{RepoID: 1, Index: 1, ID: 1}

	unittest.AssertNotExistsBean(t, commentBean)
	unittest.AssertNotExistsBean(t, issueBean, "is_closed=1")
	assert.NoError(t, issue_service.UpdateIssuesCommit(user, repo, pushCommits, repo.DefaultBranch))
	unittest.AssertExistsAndLoadBean(t, commentBean)
	unittest.AssertExistsAndLoadBean(t, issueBean, "is_closed=1")
	unittest.CheckConsistencyFor(t, &action.Action{})
}

func TestUpdateIssuesCommit_AnotherRepo_FullAddress(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)

	// Test that a push to default branch closes issue in another repo
	// If the user also has push permissions to that repo
	pushCommits := []*repository.PushCommit{
		{
			Sha1:           "abcdef1",
			CommitterEmail: "user2@example.com",
			CommitterName:  "User Two",
			AuthorEmail:    "user2@example.com",
			AuthorName:     "User Two",
			Message:        "close " + setting.AppURL + "user2/repo1/issues/1",
		},
	}

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 2}).(*repo_model.Repository)
	commentBean := &issues_model.Comment{
		Type:      issues_model.CommentTypeCommitRef,
		CommitSHA: "abcdef1",
		PosterID:  user.ID,
		IssueID:   1,
	}

	issueBean := &issues_model.Issue{RepoID: 1, Index: 1, ID: 1}

	unittest.AssertNotExistsBean(t, commentBean)
	unittest.AssertNotExistsBean(t, issueBean, "is_closed=1")
	assert.NoError(t, issue_service.UpdateIssuesCommit(user, repo, pushCommits, repo.DefaultBranch))
	unittest.AssertExistsAndLoadBean(t, commentBean)
	unittest.AssertExistsAndLoadBean(t, issueBean, "is_closed=1")
	unittest.CheckConsistencyFor(t, &action.Action{})
}

func TestUpdateIssuesCommit_AnotherRepoNoPermission(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 10}).(*user_model.User)

	// Test that a push with close reference *can not* close issue
	// If the committer doesn't have push rights in that repo
	pushCommits := []*repository.PushCommit{
		{
			Sha1:           "abcdef3",
			CommitterEmail: "user10@example.com",
			CommitterName:  "User Ten",
			AuthorEmail:    "user10@example.com",
			AuthorName:     "User Ten",
			Message:        "close user3/repo3#1",
		},
		{
			Sha1:           "abcdef4",
			CommitterEmail: "user10@example.com",
			CommitterName:  "User Ten",
			AuthorEmail:    "user10@example.com",
			AuthorName:     "User Ten",
			Message:        "close " + setting.AppURL + "user3/repo3/issues/1",
		},
	}

	repo := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 6}).(*repo_model.Repository)
	commentBean := &issues_model.Comment{
		Type:      issues_model.CommentTypeCommitRef,
		CommitSHA: "abcdef3",
		PosterID:  user.ID,
		IssueID:   6,
	}
	commentBean2 := &issues_model.Comment{
		Type:      issues_model.CommentTypeCommitRef,
		CommitSHA: "abcdef4",
		PosterID:  user.ID,
		IssueID:   6,
	}

	issueBean := &issues_model.Issue{RepoID: 3, Index: 1, ID: 6}

	unittest.AssertNotExistsBean(t, commentBean)
	unittest.AssertNotExistsBean(t, commentBean2)
	unittest.AssertNotExistsBean(t, issueBean, "is_closed=1")
	assert.NoError(t, issue_service.UpdateIssuesCommit(user, repo, pushCommits, repo.DefaultBranch))
	unittest.AssertNotExistsBean(t, commentBean)
	unittest.AssertNotExistsBean(t, commentBean2)
	unittest.AssertNotExistsBean(t, issueBean, "is_closed=1")
	unittest.CheckConsistencyFor(t, &action.Action{})
}
