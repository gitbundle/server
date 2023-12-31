// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package issues_test

import (
	"testing"

	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestPullRequest_LoadAttributes(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}).(*issues_model.PullRequest)
	assert.NoError(t, pr.LoadAttributes())
	assert.NotNil(t, pr.Merger)
	assert.Equal(t, pr.MergerID, pr.Merger.ID)
}

func TestPullRequest_LoadIssue(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}).(*issues_model.PullRequest)
	assert.NoError(t, pr.LoadIssue())
	assert.NotNil(t, pr.Issue)
	assert.Equal(t, int64(2), pr.Issue.ID)
	assert.NoError(t, pr.LoadIssue())
	assert.NotNil(t, pr.Issue)
	assert.Equal(t, int64(2), pr.Issue.ID)
}

func TestPullRequest_LoadBaseRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}).(*issues_model.PullRequest)
	assert.NoError(t, pr.LoadBaseRepo())
	assert.NotNil(t, pr.BaseRepo)
	assert.Equal(t, pr.BaseRepoID, pr.BaseRepo.ID)
	assert.NoError(t, pr.LoadBaseRepo())
	assert.NotNil(t, pr.BaseRepo)
	assert.Equal(t, pr.BaseRepoID, pr.BaseRepo.ID)
}

func TestPullRequest_LoadHeadRepo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}).(*issues_model.PullRequest)
	assert.NoError(t, pr.LoadHeadRepo())
	assert.NotNil(t, pr.HeadRepo)
	assert.Equal(t, pr.HeadRepoID, pr.HeadRepo.ID)
}

// TODO TestMerge

// TODO TestNewPullRequest

func TestPullRequestsNewest(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	prs, count, err := issues_model.PullRequests(1, &issues_model.PullRequestsOptions{
		ListOptions: db.ListOptions{
			Page: 1,
		},
		State:    "open",
		SortType: "newest",
		Labels:   []string{},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 3, count)
	if assert.Len(t, prs, 3) {
		assert.EqualValues(t, 5, prs[0].ID)
		assert.EqualValues(t, 2, prs[1].ID)
		assert.EqualValues(t, 1, prs[2].ID)
	}
}

func TestPullRequestsOldest(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	prs, count, err := issues_model.PullRequests(1, &issues_model.PullRequestsOptions{
		ListOptions: db.ListOptions{
			Page: 1,
		},
		State:    "open",
		SortType: "oldest",
		Labels:   []string{},
	})
	assert.NoError(t, err)
	assert.EqualValues(t, 3, count)
	if assert.Len(t, prs, 3) {
		assert.EqualValues(t, 1, prs[0].ID)
		assert.EqualValues(t, 2, prs[1].ID)
		assert.EqualValues(t, 5, prs[2].ID)
	}
}

func TestGetUnmergedPullRequest(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetUnmergedPullRequest(1, 1, "branch2", "master", issues_model.PullRequestFlowGithub)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), pr.ID)

	_, err = issues_model.GetUnmergedPullRequest(1, 9223372036854775807, "branch1", "master", issues_model.PullRequestFlowGithub)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestHasUnmergedPullRequestsByHeadInfo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	exist, err := issues_model.HasUnmergedPullRequestsByHeadInfo(db.DefaultContext, 1, "branch2")
	assert.NoError(t, err)
	assert.Equal(t, true, exist)

	exist, err = issues_model.HasUnmergedPullRequestsByHeadInfo(db.DefaultContext, 1, "not_exist_branch")
	assert.NoError(t, err)
	assert.Equal(t, false, exist)
}

func TestGetUnmergedPullRequestsByHeadInfo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	prs, err := issues_model.GetUnmergedPullRequestsByHeadInfo(1, "branch2")
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	for _, pr := range prs {
		assert.Equal(t, int64(1), pr.HeadRepoID)
		assert.Equal(t, "branch2", pr.HeadBranch)
	}
}

func TestGetUnmergedPullRequestsByBaseInfo(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	prs, err := issues_model.GetUnmergedPullRequestsByBaseInfo(1, "master")
	assert.NoError(t, err)
	assert.Len(t, prs, 1)
	pr := prs[0]
	assert.Equal(t, int64(2), pr.ID)
	assert.Equal(t, int64(1), pr.BaseRepoID)
	assert.Equal(t, "master", pr.BaseBranch)
}

func TestGetPullRequestByIndex(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByIndex(db.DefaultContext, 1, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pr.BaseRepoID)
	assert.Equal(t, int64(2), pr.Index)

	_, err = issues_model.GetPullRequestByIndex(db.DefaultContext, 9223372036854775807, 9223372036854775807)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))

	_, err = issues_model.GetPullRequestByIndex(db.DefaultContext, 1, 0)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestGetPullRequestByID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByID(db.DefaultContext, 1)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), pr.ID)
	assert.Equal(t, int64(2), pr.IssueID)

	_, err = issues_model.GetPullRequestByID(db.DefaultContext, 9223372036854775807)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestGetPullRequestByIssueID(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr, err := issues_model.GetPullRequestByIssueID(db.DefaultContext, 2)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), pr.IssueID)

	_, err = issues_model.GetPullRequestByIssueID(db.DefaultContext, 9223372036854775807)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrPullRequestNotExist(err))
}

func TestPullRequest_Update(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}).(*issues_model.PullRequest)
	pr.BaseBranch = "baseBranch"
	pr.HeadBranch = "headBranch"
	pr.Update()

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: pr.ID}).(*issues_model.PullRequest)
	assert.Equal(t, "baseBranch", pr.BaseBranch)
	assert.Equal(t, "headBranch", pr.HeadBranch)
	unittest.CheckConsistencyFor(t, pr)
}

func TestPullRequest_UpdateCols(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	pr := &issues_model.PullRequest{
		ID:         1,
		BaseBranch: "baseBranch",
		HeadBranch: "headBranch",
	}
	assert.NoError(t, pr.UpdateCols("head_branch"))

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}).(*issues_model.PullRequest)
	assert.Equal(t, "master", pr.BaseBranch)
	assert.Equal(t, "headBranch", pr.HeadBranch)
	unittest.CheckConsistencyFor(t, pr)
}

func TestPullRequestList_LoadAttributes(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	prs := []*issues_model.PullRequest{
		unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 1}).(*issues_model.PullRequest),
		unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}).(*issues_model.PullRequest),
	}
	assert.NoError(t, issues_model.PullRequestList(prs).LoadAttributes())
	for _, pr := range prs {
		assert.NotNil(t, pr.Issue)
		assert.Equal(t, pr.IssueID, pr.Issue.ID)
	}

	assert.NoError(t, issues_model.PullRequestList([]*issues_model.PullRequest{}).LoadAttributes())
}

// TODO TestAddTestPullRequestTask

func TestPullRequest_IsWorkInProgress(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}).(*issues_model.PullRequest)
	pr.LoadIssue()

	assert.False(t, pr.IsWorkInProgress())

	pr.Issue.Title = "WIP: " + pr.Issue.Title
	assert.True(t, pr.IsWorkInProgress())

	pr.Issue.Title = "[wip]: " + pr.Issue.Title
	assert.True(t, pr.IsWorkInProgress())
}

func TestPullRequest_GetWorkInProgressPrefixWorkInProgress(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}).(*issues_model.PullRequest)
	pr.LoadIssue()

	assert.Empty(t, pr.GetWorkInProgressPrefix())

	original := pr.Issue.Title
	pr.Issue.Title = "WIP: " + original
	assert.Equal(t, "WIP:", pr.GetWorkInProgressPrefix())

	pr.Issue.Title = "[wip] " + original
	assert.Equal(t, "[wip]", pr.GetWorkInProgressPrefix())
}

func TestDeleteOrphanedObjects(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	countBefore, err := db.GetEngine(db.DefaultContext).Count(&issues_model.PullRequest{})
	assert.NoError(t, err)

	_, err = db.GetEngine(db.DefaultContext).Insert(&issues_model.PullRequest{IssueID: 1000}, &issues_model.PullRequest{IssueID: 1001}, &issues_model.PullRequest{IssueID: 1003})
	assert.NoError(t, err)

	orphaned, err := db.CountOrphanedObjects("pull_request", "issue", "pull_request.issue_id=issue.id")
	assert.NoError(t, err)
	assert.EqualValues(t, 3, orphaned)

	err = db.DeleteOrphanedObjects("pull_request", "issue", "pull_request.issue_id=issue.id")
	assert.NoError(t, err)

	countAfter, err := db.GetEngine(db.DefaultContext).Count(&issues_model.PullRequest{})
	assert.NoError(t, err)
	assert.EqualValues(t, countBefore, countAfter)
}
