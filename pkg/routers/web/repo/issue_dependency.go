// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package repo

import (
	"net/http"

	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/context"
	issues_model "github.com/gitbundle/server/pkg/issues"
)

// AddDependency adds new dependencies
func AddDependency(ctx *context.Context) {
	issueIndex := ctx.ParamsInt64("index")
	issue, err := issues_model.GetIssueByIndex(ctx.Repo.Repository.ID, issueIndex)
	if err != nil {
		ctx.ServerError("GetIssueByIndex", err)
		return
	}

	// Check if the Repo is allowed to have dependencies
	if !ctx.Repo.CanCreateIssueDependencies(ctx.Doer, issue.IsPull) {
		ctx.Error(http.StatusForbidden, "CanCreateIssueDependencies")
		return
	}

	depID := ctx.FormInt64("newDependency")

	if err = issue.LoadRepo(ctx); err != nil {
		ctx.ServerError("LoadRepo", err)
		return
	}

	// Redirect
	defer ctx.Redirect(issue.HTMLURL())

	// Dependency
	dep, err := issues_model.GetIssueByID(ctx, depID)
	if err != nil {
		ctx.Flash.Error(ctx.Tr("repo.issues.dependency.add_error_dep_issue_not_exist"))
		return
	}

	// Check if both issues are in the same repo if cross repository dependencies is not enabled
	if issue.RepoID != dep.RepoID && !setting.Service.AllowCrossRepositoryDependencies {
		ctx.Flash.Error(ctx.Tr("repo.issues.dependency.add_error_dep_not_same_repo"))
		return
	}

	// Check if issue and dependency is the same
	if dep.ID == issue.ID {
		ctx.Flash.Error(ctx.Tr("repo.issues.dependency.add_error_same_issue"))
		return
	}

	err = issues_model.CreateIssueDependency(ctx.Doer, issue, dep)
	if err != nil {
		if issues_model.IsErrDependencyExists(err) {
			ctx.Flash.Error(ctx.Tr("repo.issues.dependency.add_error_dep_exists"))
			return
		} else if issues_model.IsErrCircularDependency(err) {
			ctx.Flash.Error(ctx.Tr("repo.issues.dependency.add_error_cannot_create_circular"))
			return
		} else {
			ctx.ServerError("CreateOrUpdateIssueDependency", err)
			return
		}
	}
}

// RemoveDependency removes the dependency
func RemoveDependency(ctx *context.Context) {
	issueIndex := ctx.ParamsInt64("index")
	issue, err := issues_model.GetIssueByIndex(ctx.Repo.Repository.ID, issueIndex)
	if err != nil {
		ctx.ServerError("GetIssueByIndex", err)
		return
	}

	// Check if the Repo is allowed to have dependencies
	if !ctx.Repo.CanCreateIssueDependencies(ctx.Doer, issue.IsPull) {
		ctx.Error(http.StatusForbidden, "CanCreateIssueDependencies")
		return
	}

	depID := ctx.FormInt64("removeDependencyID")

	if err = issue.LoadRepo(ctx); err != nil {
		ctx.ServerError("LoadRepo", err)
		return
	}

	// Dependency Type
	depTypeStr := ctx.Req.PostForm.Get("dependencyType")

	var depType issues_model.DependencyType

	switch depTypeStr {
	case "blockedBy":
		depType = issues_model.DependencyTypeBlockedBy
	case "blocking":
		depType = issues_model.DependencyTypeBlocking
	default:
		ctx.Error(http.StatusBadRequest, "GetDependecyType")
		return
	}

	// Dependency
	dep, err := issues_model.GetIssueByID(ctx, depID)
	if err != nil {
		ctx.ServerError("GetIssueByID", err)
		return
	}

	if err = issues_model.RemoveIssueDependency(ctx.Doer, issue, dep, depType); err != nil {
		if issues_model.IsErrDependencyNotExists(err) {
			ctx.Flash.Error(ctx.Tr("repo.issues.dependency.add_error_dep_not_exist"))
			return
		}
		ctx.ServerError("RemoveIssueDependency", err)
		return
	}

	// Redirect
	ctx.Redirect(issue.HTMLURL())
}
