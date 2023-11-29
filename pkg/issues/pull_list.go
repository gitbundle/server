// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package issues

import (
	"context"
	"fmt"

	"github.com/gitbundle/modules/base"
	"github.com/gitbundle/modules/git"
	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/server/pkg/db"
	access_model "github.com/gitbundle/server/pkg/perm/access"
	"github.com/gitbundle/server/pkg/unit"
	user_model "github.com/gitbundle/server/pkg/user"

	"xorm.io/xorm"
)

// PullRequestsOptions holds the options for PRs
type PullRequestsOptions struct {
	db.ListOptions
	State       string
	SortType    string
	Labels      []string
	MilestoneID int64
}

func listPullRequestStatement(baseRepoID int64, opts *PullRequestsOptions) (*xorm.Session, error) {
	sess := db.GetEngine(db.DefaultContext).Where("pull_request.base_repo_id=?", baseRepoID)

	sess.Join("INNER", "issue", "pull_request.issue_id = issue.id")
	switch opts.State {
	case "closed", "open":
		sess.And("issue.is_closed=?", opts.State == "closed")
	}

	if labelIDs, err := base.StringsToInt64s(opts.Labels); err != nil {
		return nil, err
	} else if len(labelIDs) > 0 {
		sess.Join("INNER", "issue_label", "issue.id = issue_label.issue_id").
			In("issue_label.label_id", labelIDs)
	}

	if opts.MilestoneID > 0 {
		sess.And("issue.milestone_id=?", opts.MilestoneID)
	}

	return sess, nil
}

// GetUnmergedPullRequestsByHeadInfo returns all pull requests that are open and has not been merged
// by given head information (repo and branch).
func GetUnmergedPullRequestsByHeadInfo(repoID int64, branch string) ([]*PullRequest, error) {
	prs := make([]*PullRequest, 0, 2)
	return prs, db.GetEngine(db.DefaultContext).
		Where("head_repo_id = ? AND head_branch = ? AND has_merged = ? AND issue.is_closed = ? AND flow = ?",
			repoID, branch, false, false, PullRequestFlowGithub).
		Join("INNER", "issue", "issue.id = pull_request.issue_id").
		Find(&prs)
}

// CanMaintainerWriteToBranch check whether user is a matainer and could write to the branch
func CanMaintainerWriteToBranch(p access_model.Permission, branch string, user *user_model.User) bool {
	if p.CanWrite(unit.TypeCode) {
		return true
	}

	if len(p.Units) < 1 {
		return false
	}

	prs, err := GetUnmergedPullRequestsByHeadInfo(p.Units[0].RepoID, branch)
	if err != nil {
		return false
	}

	for _, pr := range prs {
		if pr.AllowMaintainerEdit {
			err = pr.LoadBaseRepo()
			if err != nil {
				continue
			}
			prPerm, err := access_model.GetUserRepoPermission(db.DefaultContext, pr.BaseRepo, user)
			if err != nil {
				continue
			}
			if prPerm.CanWrite(unit.TypeCode) {
				return true
			}
		}
	}
	return false
}

// HasUnmergedPullRequestsByHeadInfo checks if there are open and not merged pull request
// by given head information (repo and branch)
func HasUnmergedPullRequestsByHeadInfo(ctx context.Context, repoID int64, branch string) (bool, error) {
	return db.GetEngine(ctx).
		Where("head_repo_id = ? AND head_branch = ? AND has_merged = ? AND issue.is_closed = ? AND flow = ?",
			repoID, branch, false, false, PullRequestFlowGithub).
		Join("INNER", "issue", "issue.id = pull_request.issue_id").
		Exist(&PullRequest{})
}

// GetUnmergedPullRequestsByBaseInfo returns all pull requests that are open and has not been merged
// by given base information (repo and branch).
func GetUnmergedPullRequestsByBaseInfo(repoID int64, branch string) ([]*PullRequest, error) {
	prs := make([]*PullRequest, 0, 2)
	return prs, db.GetEngine(db.DefaultContext).
		Where("base_repo_id=? AND base_branch=? AND has_merged=? AND issue.is_closed=?",
			repoID, branch, false, false).
		Join("INNER", "issue", "issue.id=pull_request.issue_id").
		Find(&prs)
}

// GetPullRequestIDsByCheckStatus returns all pull requests according the special checking status.
func GetPullRequestIDsByCheckStatus(status PullRequestStatus) ([]int64, error) {
	prs := make([]int64, 0, 10)
	return prs, db.GetEngine(db.DefaultContext).Table("pull_request").
		Where("status=?", status).
		Cols("pull_request.id").
		Find(&prs)
}

// PullRequests returns all pull requests for a base Repo by the given conditions
func PullRequests(baseRepoID int64, opts *PullRequestsOptions) ([]*PullRequest, int64, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}

	countSession, err := listPullRequestStatement(baseRepoID, opts)
	if err != nil {
		log.Error("listPullRequestStatement: %v", err)
		return nil, 0, err
	}
	maxResults, err := countSession.Count(new(PullRequest))
	if err != nil {
		log.Error("Count PRs: %v", err)
		return nil, maxResults, err
	}

	findSession, err := listPullRequestStatement(baseRepoID, opts)
	sortIssuesSession(findSession, opts.SortType, 0)
	if err != nil {
		log.Error("listPullRequestStatement: %v", err)
		return nil, maxResults, err
	}
	findSession = db.SetSessionPagination(findSession, opts)
	prs := make([]*PullRequest, 0, opts.PageSize)
	return prs, maxResults, findSession.Find(&prs)
}

// PullRequestList defines a list of pull requests
type PullRequestList []*PullRequest

func (prs PullRequestList) loadAttributes(ctx context.Context) error {
	if len(prs) == 0 {
		return nil
	}

	// Load issues.
	issueIDs := prs.getIssueIDs()
	issues := make([]*Issue, 0, len(issueIDs))
	if err := db.GetEngine(ctx).
		Where("id > 0").
		In("id", issueIDs).
		Find(&issues); err != nil {
		return fmt.Errorf("find issues: %v", err)
	}

	set := make(map[int64]*Issue)
	for i := range issues {
		set[issues[i].ID] = issues[i]
	}
	for i := range prs {
		prs[i].Issue = set[prs[i].IssueID]
	}
	return nil
}

func (prs PullRequestList) getIssueIDs() []int64 {
	issueIDs := make([]int64, 0, len(prs))
	for i := range prs {
		issueIDs = append(issueIDs, prs[i].IssueID)
	}
	return issueIDs
}

// LoadAttributes load all the prs attributes
func (prs PullRequestList) LoadAttributes() error {
	return prs.loadAttributes(db.DefaultContext)
}

// InvalidateCodeComments will lookup the prs for code comments which got invalidated by change
func (prs PullRequestList) InvalidateCodeComments(ctx context.Context, doer *user_model.User, repo *git.Repository, branch string) error {
	if len(prs) == 0 {
		return nil
	}
	issueIDs := prs.getIssueIDs()
	var codeComments []*Comment
	if err := db.GetEngine(ctx).
		Where("type = ? and invalidated = ?", CommentTypeCode, false).
		In("issue_id", issueIDs).
		Find(&codeComments); err != nil {
		return fmt.Errorf("find code comments: %v", err)
	}
	for _, comment := range codeComments {
		if err := comment.CheckInvalidation(repo, doer, branch); err != nil {
			return err
		}
	}
	return nil
}