// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package org

import (
	"net/http"

	"github.com/gitbundle/server/pkg/context"
	"github.com/gitbundle/server/pkg/db"
	"github.com/gitbundle/server/pkg/forms"
	issues_model "github.com/gitbundle/server/pkg/issues"
	repo_module "github.com/gitbundle/server/pkg/repo/repository"
	"github.com/gitbundle/server/pkg/web"
)

// RetrieveLabels find all the labels of an organization
func RetrieveLabels(ctx *context.Context) {
	labels, err := issues_model.GetLabelsByOrgID(ctx, ctx.Org.Organization.ID, ctx.FormString("sort"), db.ListOptions{})
	if err != nil {
		ctx.ServerError("RetrieveLabels.GetLabels", err)
		return
	}
	for _, l := range labels {
		l.CalOpenIssues()
	}
	ctx.Data["Labels"] = labels
	ctx.Data["NumLabels"] = len(labels)
	ctx.Data["SortType"] = ctx.FormString("sort")
}

// NewLabel create new label for organization
func NewLabel(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateLabelForm)
	ctx.Data["Title"] = ctx.Tr("repo.labels")
	ctx.Data["PageIsLabels"] = true
	ctx.Data["PageIsOrgSettings"] = true

	if ctx.HasError() {
		ctx.Flash.Error(ctx.Data["ErrorMsg"].(string))
		ctx.Redirect(ctx.Org.OrgLink + "/settings/labels")
		return
	}

	l := &issues_model.Label{
		OrgID:       ctx.Org.Organization.ID,
		Name:        form.Title,
		Description: form.Description,
		Color:       form.Color,
	}
	if err := issues_model.NewLabel(ctx, l); err != nil {
		ctx.ServerError("NewLabel", err)
		return
	}
	ctx.Redirect(ctx.Org.OrgLink + "/settings/labels")
}

// UpdateLabel update a label's name and color
func UpdateLabel(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.CreateLabelForm)
	l, err := issues_model.GetLabelInOrgByID(ctx, ctx.Org.Organization.ID, form.ID)
	if err != nil {
		switch {
		case issues_model.IsErrOrgLabelNotExist(err):
			ctx.Error(http.StatusNotFound)
		default:
			ctx.ServerError("UpdateLabel", err)
		}
		return
	}

	l.Name = form.Title
	l.Description = form.Description
	l.Color = form.Color
	if err := issues_model.UpdateLabel(l); err != nil {
		ctx.ServerError("UpdateLabel", err)
		return
	}
	ctx.Redirect(ctx.Org.OrgLink + "/settings/labels")
}

// DeleteLabel delete a label
func DeleteLabel(ctx *context.Context) {
	if err := issues_model.DeleteLabel(ctx.Org.Organization.ID, ctx.FormInt64("id")); err != nil {
		ctx.Flash.Error("DeleteLabel: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("repo.issues.label_deletion_success"))
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Org.OrgLink + "/settings/labels",
	})
}

// InitializeLabels init labels for an organization
func InitializeLabels(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.InitializeLabelsForm)
	if ctx.HasError() {
		ctx.Redirect(ctx.Org.OrgLink + "/labels")
		return
	}

	if err := repo_module.InitializeLabels(ctx, ctx.Org.Organization.ID, form.TemplateName, true); err != nil {
		if repo_module.IsErrIssueLabelTemplateLoad(err) {
			originalErr := err.(repo_module.ErrIssueLabelTemplateLoad).OriginalError
			ctx.Flash.Error(ctx.Tr("repo.issues.label_templates.fail_to_load_file", form.TemplateName, originalErr))
			ctx.Redirect(ctx.Org.OrgLink + "/settings/labels")
			return
		}
		ctx.ServerError("InitializeLabels", err)
		return
	}
	ctx.Redirect(ctx.Org.OrgLink + "/settings/labels")
}
