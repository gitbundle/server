// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

// Package private includes all internal routes. The package name internal is ideal but Golang is not allowed, so we use private as package name instead.
package private

import (
	"fmt"
	"net/http"

	"github.com/gitbundle/modules/git"
	gitea_context "github.com/gitbundle/server/pkg/context"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/repo/private"
)

// ________          _____             .__   __
// \______ \   _____/ ____\____   __ __|  |_/  |_
//  |    |  \_/ __ \   __\\__  \ |  |  \  |\   __\
//  |    `   \  ___/|  |   / __ \|  |  /  |_|  |
// /_______  /\___  >__|  (____  /____/|____/__|
//         \/     \/           \/
// __________                             .__
// \______   \____________    ____   ____ |  |__
//  |    |  _/\_  __ \__  \  /    \_/ ___\|  |  \
//  |    |   \ |  | \// __ \|   |  \  \___|   Y  \
//  |______  / |__|  (____  /___|  /\___  >___|  /
//         \/             \/     \/     \/     \/

// SetDefaultBranch updates the default branch
func SetDefaultBranch(ctx *gitea_context.PrivateContext) {
	ownerName := ctx.Params(":owner")
	repoName := ctx.Params(":repo")
	branch := ctx.Params(":branch")

	ctx.Repo.Repository.DefaultBranch = branch
	if err := ctx.Repo.GitRepo.SetDefaultBranch(ctx.Repo.Repository.DefaultBranch); err != nil {
		if !git.IsErrUnsupportedVersion(err) {
			ctx.JSON(http.StatusInternalServerError, private.Response{
				Err: fmt.Sprintf("Unable to set default branch on repository: %s/%s Error: %v", ownerName, repoName, err),
			})
			return
		}
	}

	if err := repo_model.UpdateDefaultBranch(ctx.Repo.Repository); err != nil {
		ctx.JSON(http.StatusInternalServerError, private.Response{
			Err: fmt.Sprintf("Unable to set default branch on repository: %s/%s Error: %v", ownerName, repoName, err),
		})
		return
	}
	ctx.PlainText(http.StatusOK, "success")
}
