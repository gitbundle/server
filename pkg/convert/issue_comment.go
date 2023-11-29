// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package convert

import (
	api "github.com/gitbundle/api/pkg/structs"
	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	repo_model "github.com/gitbundle/server/pkg/repo"
	user_model "github.com/gitbundle/server/pkg/user"
)

// ToComment converts a issues_model.Comment to the api.Comment format
func ToComment(c *issues_model.Comment) *api.Comment {
	return &api.Comment{
		ID:       c.ID,
		Poster:   ToUser(c.Poster, nil),
		HTMLURL:  c.HTMLURL(),
		IssueURL: c.IssueURL(),
		PRURL:    c.PRURL(),
		Body:     c.Content,
		Created:  c.CreatedUnix.AsTime(),
		Updated:  c.UpdatedUnix.AsTime(),
	}
}

// ToTimelineComment converts a issues_model.Comment to the api.TimelineComment format
func ToTimelineComment(c *issues_model.Comment, doer *user_model.User) *api.TimelineComment {
	err := c.LoadMilestone()
	if err != nil {
		log.Error("LoadMilestone: %v", err)
		return nil
	}

	err = c.LoadAssigneeUserAndTeam()
	if err != nil {
		log.Error("LoadAssigneeUserAndTeam: %v", err)
		return nil
	}

	err = c.LoadResolveDoer()
	if err != nil {
		log.Error("LoadResolveDoer: %v", err)
		return nil
	}

	err = c.LoadDepIssueDetails()
	if err != nil {
		log.Error("LoadDepIssueDetails: %v", err)
		return nil
	}

	err = c.LoadTime()
	if err != nil {
		log.Error("LoadTime: %v", err)
		return nil
	}

	err = c.LoadLabel()
	if err != nil {
		log.Error("LoadLabel: %v", err)
		return nil
	}

	comment := &api.TimelineComment{
		ID:       c.ID,
		Type:     c.Type.String(),
		Poster:   ToUser(c.Poster, nil),
		HTMLURL:  c.HTMLURL(),
		IssueURL: c.IssueURL(),
		PRURL:    c.PRURL(),
		Body:     c.Content,
		Created:  c.CreatedUnix.AsTime(),
		Updated:  c.UpdatedUnix.AsTime(),

		OldProjectID: c.OldProjectID,
		ProjectID:    c.ProjectID,

		OldTitle: c.OldTitle,
		NewTitle: c.NewTitle,

		OldRef: c.OldRef,
		NewRef: c.NewRef,

		RefAction:    c.RefAction.String(),
		RefCommitSHA: c.CommitSHA,

		ReviewID: c.ReviewID,

		RemovedAssignee: c.RemovedAssignee,
	}

	if c.OldMilestone != nil {
		comment.OldMilestone = ToAPIMilestone(c.OldMilestone)
	}
	if c.Milestone != nil {
		comment.Milestone = ToAPIMilestone(c.Milestone)
	}

	if c.Time != nil {
		comment.TrackedTime = ToTrackedTime(c.Time)
	}

	if c.RefIssueID != 0 {
		issue, err := issues_model.GetIssueByID(db.DefaultContext, c.RefIssueID)
		if err != nil {
			log.Error("GetIssueByID(%d): %v", c.RefIssueID, err)
			return nil
		}
		comment.RefIssue = ToAPIIssue(issue)
	}

	if c.RefCommentID != 0 {
		com, err := issues_model.GetCommentByID(db.DefaultContext, c.RefCommentID)
		if err != nil {
			log.Error("GetCommentByID(%d): %v", c.RefCommentID, err)
			return nil
		}
		err = com.LoadPoster()
		if err != nil {
			log.Error("LoadPoster: %v", err)
			return nil
		}
		comment.RefComment = ToComment(com)
	}

	if c.Label != nil {
		var org *user_model.User
		var repo *repo_model.Repository
		if c.Label.BelongsToOrg() {
			var err error
			org, err = user_model.GetUserByID(c.Label.OrgID)
			if err != nil {
				log.Error("GetUserByID(%d): %v", c.Label.OrgID, err)
				return nil
			}
		}
		if c.Label.BelongsToRepo() {
			var err error
			repo, err = repo_model.GetRepositoryByID(c.Label.RepoID)
			if err != nil {
				log.Error("GetRepositoryByID(%d): %v", c.Label.RepoID, err)
				return nil
			}
		}
		comment.Label = ToLabel(c.Label, repo, org)
	}

	if c.Assignee != nil {
		comment.Assignee = ToUser(c.Assignee, nil)
	}
	if c.AssigneeTeam != nil {
		comment.AssigneeTeam, _ = ToTeam(c.AssigneeTeam)
	}

	if c.ResolveDoer != nil {
		comment.ResolveDoer = ToUser(c.ResolveDoer, nil)
	}

	if c.DependentIssue != nil {
		comment.DependentIssue = ToAPIIssue(c.DependentIssue)
	}

	return comment
}
