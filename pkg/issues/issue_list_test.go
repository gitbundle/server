// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package issues_test

import (
	"testing"

	"github.com/gitbundle/modules/setting"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestIssueList_LoadRepositories(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	issueList := issues_model.IssueList{
		unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1}).(*issues_model.Issue),
		unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2}).(*issues_model.Issue),
		unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 4}).(*issues_model.Issue),
	}

	repos, err := issueList.LoadRepositories()
	assert.NoError(t, err)
	assert.Len(t, repos, 2)
	for _, issue := range issueList {
		assert.EqualValues(t, issue.RepoID, issue.Repo.ID)
	}
}

func TestIssueList_LoadAttributes(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	setting.Service.EnableTimetracking = true
	issueList := issues_model.IssueList{
		unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1}).(*issues_model.Issue),
		unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 4}).(*issues_model.Issue),
	}

	assert.NoError(t, issueList.LoadAttributes())
	for _, issue := range issueList {
		assert.EqualValues(t, issue.RepoID, issue.Repo.ID)
		for _, label := range issue.Labels {
			assert.EqualValues(t, issue.RepoID, label.RepoID)
			unittest.AssertExistsAndLoadBean(t, &issues_model.IssueLabel{IssueID: issue.ID, LabelID: label.ID})
		}
		if issue.PosterID > 0 {
			assert.EqualValues(t, issue.PosterID, issue.Poster.ID)
		}
		if issue.AssigneeID > 0 {
			assert.EqualValues(t, issue.AssigneeID, issue.Assignee.ID)
		}
		if issue.MilestoneID > 0 {
			assert.EqualValues(t, issue.MilestoneID, issue.Milestone.ID)
		}
		if issue.IsPull {
			assert.EqualValues(t, issue.ID, issue.PullRequest.IssueID)
		}
		for _, attachment := range issue.Attachments {
			assert.EqualValues(t, issue.ID, attachment.IssueID)
		}
		for _, comment := range issue.Comments {
			assert.EqualValues(t, issue.ID, comment.IssueID)
		}
		if issue.ID == int64(1) {
			assert.Equal(t, int64(400), issue.TotalTrackedTime)
		} else if issue.ID == int64(2) {
			assert.Equal(t, int64(3682), issue.TotalTrackedTime)
		}
	}
}
