// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package org

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gitbundle/modules/base"
	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/context"
	"github.com/gitbundle/server/pkg/db"
	"github.com/gitbundle/server/pkg/forms"
	org_service "github.com/gitbundle/server/pkg/organization/manager"
	container_service "github.com/gitbundle/server/pkg/packages/manager/container"
	repo_model "github.com/gitbundle/server/pkg/repo"
	repo_module "github.com/gitbundle/server/pkg/repo/repository"
	repo_service "github.com/gitbundle/server/pkg/repo/repository_manager"
	user_setting "github.com/gitbundle/server/pkg/routers/web/user/setting"
	user_model "github.com/gitbundle/server/pkg/user"
	user_service "github.com/gitbundle/server/pkg/user/manager"
	"github.com/gitbundle/server/pkg/web"
	"github.com/gitbundle/server/pkg/webhook"
)

const (
	// tplSettingsOptions template path for render settings
	tplSettingsOptions base.TplName = "org/settings/options"
	// tplSettingsDelete template path for render delete repository
	tplSettingsDelete base.TplName = "org/settings/delete"
	// tplSettingsHooks template path for render hook settings
	tplSettingsHooks base.TplName = "org/settings/hooks"
	// tplSettingsLabels template path for render labels settings
	tplSettingsLabels base.TplName = "org/settings/labels"
)

// Settings render the main settings page
func Settings(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("org.settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsOptions"] = true
	ctx.Data["CurrentVisibility"] = ctx.Org.Organization.Visibility
	ctx.Data["RepoAdminChangeTeamAccess"] = ctx.Org.Organization.RepoAdminChangeTeamAccess
	ctx.Data["IsPackagesEnableForAllOrgs"] = setting.Packages.EnableForAllOrgs
	ctx.Data["IsOrgOwner"] = ctx.Org.IsOwner

	ctx.HTML(http.StatusOK, tplSettingsOptions)
}

// NOTE: if the kubernetes config is override by organization, we do not provide resource limitions
// SettingsPost response for settings change submitted
func SettingsPost(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.UpdateOrgSettingForm)
	ctx.Data["Title"] = ctx.Tr("org.settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsOptions"] = true
	ctx.Data["CurrentVisibility"] = ctx.Org.Organization.Visibility

	if ctx.HasError() {
		ctx.HTML(http.StatusOK, tplSettingsOptions)
		return
	}

	org := ctx.Org.Organization
	nameChanged := org.Name != form.Name

	// Check if organization name has been changed.
	if org.LowerName != strings.ToLower(form.Name) {
		isExist, err := user_model.IsUserExist(ctx, org.ID, form.Name)
		if err != nil {
			ctx.ServerError("IsUserExist", err)
			return
		} else if isExist {
			ctx.Data["OrgName"] = true
			ctx.RenderWithErr(ctx.Tr("form.username_been_taken"), tplSettingsOptions, &form)
			return
		} else if err = user_model.ChangeUserName(org.AsUser(), form.Name); err != nil {
			switch {
			case db.IsErrNameReserved(err):
				ctx.Data["OrgName"] = true
				ctx.RenderWithErr(ctx.Tr("repo.form.name_reserved", err.(db.ErrNameReserved).Name), tplSettingsOptions, &form)
			case db.IsErrNamePatternNotAllowed(err):
				ctx.Data["OrgName"] = true
				ctx.RenderWithErr(ctx.Tr("repo.form.name_pattern_not_allowed", err.(db.ErrNamePatternNotAllowed).Pattern), tplSettingsOptions, &form)
			default:
				ctx.ServerError("ChangeUserName", err)
			}
			return
		}

		if err := container_service.UpdateRepositoryNames(ctx, org.AsUser(), form.Name); err != nil {
			ctx.ServerError("UpdateRepositoryNames", err)
			return
		}

		// reset ctx.org.OrgLink with new name
		ctx.Org.OrgLink = setting.AppSubURL + "/org/" + url.PathEscape(form.Name)
		log.Trace("Organization name changed: %s -> %s", org.Name, form.Name)
		nameChanged = false
	}

	// In case it's just a case change.
	org.Name = form.Name
	org.LowerName = strings.ToLower(form.Name)

	if ctx.Doer.IsAdmin {
		org.MaxRepoCreation = form.MaxRepoCreation
		if !setting.Packages.EnableForAllOrgs {
			org.EnabledPackages = form.EnabledPackages
		}
	}

	org.FullName = form.FullName
	org.Description = form.Description
	org.Website = form.Website
	org.Location = form.Location
	org.RepoAdminChangeTeamAccess = form.RepoAdminChangeTeamAccess

	visibilityChanged := form.Visibility != org.Visibility
	org.Visibility = form.Visibility

	if err := user_model.UpdateUser(ctx, org.AsUser(), false); err != nil {
		ctx.ServerError("UpdateUser", err)
		return
	}

	// update forks visibility
	if visibilityChanged {
		repos, _, err := repo_model.GetUserRepositories(&repo_model.SearchRepoOptions{
			Actor: org.AsUser(), Private: true, ListOptions: db.ListOptions{Page: 1, PageSize: org.NumRepos},
		})
		if err != nil {
			ctx.ServerError("GetRepositories", err)
			return
		}
		for _, repo := range repos {
			repo.OwnerName = org.Name
			if err := repo_service.UpdateRepository(repo, true); err != nil {
				ctx.ServerError("UpdateRepository", err)
				return
			}
		}
	} else if nameChanged {
		if err := repo_model.UpdateRepositoryOwnerNames(org.ID, org.Name); err != nil {
			ctx.ServerError("UpdateRepository", err)
			return
		}
	}

	log.Trace("Organization setting updated: %s", org.Name)
	ctx.Flash.Success(ctx.Tr("org.settings.update_setting_success"))
	ctx.Redirect(ctx.Org.OrgLink + "/settings")
}

// SettingsAvatar response for change avatar on settings page
func SettingsAvatar(ctx *context.Context) {
	form := web.GetForm(ctx).(*forms.AvatarForm)
	form.Source = forms.AvatarLocal
	if err := user_setting.UpdateAvatarSetting(ctx, form, ctx.Org.Organization.AsUser()); err != nil {
		ctx.Flash.Error(err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("org.settings.update_avatar_success"))
	}

	ctx.Redirect(ctx.Org.OrgLink + "/settings")
}

// SettingsDeleteAvatar response for delete avatar on settings page
func SettingsDeleteAvatar(ctx *context.Context) {
	if err := user_service.DeleteAvatar(ctx.Org.Organization.AsUser()); err != nil {
		ctx.Flash.Error(err.Error())
	}

	ctx.Redirect(ctx.Org.OrgLink + "/settings")
}

// SettingsDelete response for deleting an organization
func SettingsDelete(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("org.settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsDelete"] = true

	if ctx.Req.Method == "POST" {
		if ctx.Org.Organization.Name != ctx.FormString("org_name") {
			ctx.Data["Err_OrgName"] = true
			ctx.RenderWithErr(ctx.Tr("form.enterred_invalid_org_name"), tplSettingsDelete, nil)
			return
		}

		if err := org_service.DeleteOrganization(ctx.Org.Organization); err != nil {
			if user_model.IsErrUserOwnRepos(err) {
				ctx.Flash.Error(ctx.Tr("form.org_still_own_repo"))
				ctx.Redirect(ctx.Org.OrgLink + "/settings/delete")
			} else if user_model.IsErrUserOwnPackages(err) {
				ctx.Flash.Error(ctx.Tr("form.org_still_own_packages"))
				ctx.Redirect(ctx.Org.OrgLink + "/settings/delete")
			} else {
				ctx.ServerError("DeleteOrganization", err)
			}
		} else {
			log.Trace("Organization deleted: %s", ctx.Org.Organization.Name)
			ctx.Redirect(setting.AppSubURL + "/")
		}
		return
	}

	ctx.HTML(http.StatusOK, tplSettingsDelete)
}

// Webhooks render webhook list page
func Webhooks(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("org.settings")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsSettingsHooks"] = true
	ctx.Data["BaseLink"] = ctx.Org.OrgLink + "/settings/hooks"
	ctx.Data["BaseLinkNew"] = ctx.Org.OrgLink + "/settings/hooks"
	ctx.Data["Description"] = ctx.Tr("org.settings.hooks_desc")

	ws, err := webhook.ListWebhooksByOpts(ctx, &webhook.ListWebhookOptions{OrgID: ctx.Org.Organization.ID})
	if err != nil {
		ctx.ServerError("GetWebhooksByOrgId", err)
		return
	}

	ctx.Data["Webhooks"] = ws
	ctx.HTML(http.StatusOK, tplSettingsHooks)
}

// DeleteWebhook response for delete webhook
func DeleteWebhook(ctx *context.Context) {
	if err := webhook.DeleteWebhookByOrgID(ctx.Org.Organization.ID, ctx.FormInt64("id")); err != nil {
		ctx.Flash.Error("DeleteWebhookByOrgID: " + err.Error())
	} else {
		ctx.Flash.Success(ctx.Tr("repo.settings.webhook_deletion_success"))
	}

	ctx.JSON(http.StatusOK, map[string]interface{}{
		"redirect": ctx.Org.OrgLink + "/settings/hooks",
	})
}

// Labels render organization labels page
func Labels(ctx *context.Context) {
	ctx.Data["Title"] = ctx.Tr("repo.labels")
	ctx.Data["PageIsOrgSettings"] = true
	ctx.Data["PageIsOrgSettingsLabels"] = true
	ctx.Data["RequireTribute"] = true
	ctx.Data["LabelTemplates"] = repo_module.LabelTemplates
	ctx.HTML(http.StatusOK, tplSettingsLabels)
}
