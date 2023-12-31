// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package repo

import (
	"context"

	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/db"
	"github.com/gitbundle/server/pkg/unit"
)

// ___________.__             ___________                     __
// \__    ___/|__| _____   ___\__    ___/___________    ____ |  | __ ___________
// |    |   |  |/     \_/ __ \|    |  \_  __ \__  \ _/ ___\|  |/ // __ \_  __ \
// |    |   |  |  Y Y  \  ___/|    |   |  | \// __ \\  \___|    <\  ___/|  | \/
// |____|   |__|__|_|  /\___  >____|   |__|  (____  /\___  >__|_ \\___  >__|
// \/     \/                    \/     \/     \/    \/

// CanEnableTimetracker returns true when the server admin enabled time tracking
// This overrules IsTimetrackerEnabled
func (repo *Repository) CanEnableTimetracker() bool {
	return setting.Service.EnableTimetracking
}

// IsTimetrackerEnabled returns whether or not the timetracker is enabled. It returns the default value from config if an error occurs.
func (repo *Repository) IsTimetrackerEnabled() bool {
	return repo.IsTimetrackerEnabledCtx(db.DefaultContext)
}

// IsTimetrackerEnabledCtx returns whether or not the timetracker is enabled. It returns the default value from config if an error occurs.
func (repo *Repository) IsTimetrackerEnabledCtx(ctx context.Context) bool {
	if !setting.Service.EnableTimetracking {
		return false
	}

	var u *RepoUnit
	var err error
	if u, err = repo.GetUnitCtx(ctx, unit.TypeIssues); err != nil {
		return setting.Service.DefaultEnableTimetracking
	}
	return u.IssuesConfig().EnableTimetracker
}

// AllowOnlyContributorsToTrackTime returns value of IssuesConfig or the default value
func (repo *Repository) AllowOnlyContributorsToTrackTime() bool {
	var u *RepoUnit
	var err error
	if u, err = repo.GetUnit(unit.TypeIssues); err != nil {
		return setting.Service.DefaultAllowOnlyContributorsToTrackTime
	}
	return u.IssuesConfig().AllowOnlyContributorsToTrackTime
}

// IsDependenciesEnabled returns if dependencies are enabled and returns the default setting if not set.
func (repo *Repository) IsDependenciesEnabled() bool {
	return repo.IsDependenciesEnabledCtx(db.DefaultContext)
}

// IsDependenciesEnabledCtx returns if dependencies are enabled and returns the default setting if not set.
func (repo *Repository) IsDependenciesEnabledCtx(ctx context.Context) bool {
	var u *RepoUnit
	var err error
	if u, err = repo.GetUnitCtx(ctx, unit.TypeIssues); err != nil {
		log.Trace("%s", err)
		return setting.Service.DefaultEnableDependencies
	}
	return u.IssuesConfig().EnableDependencies
}
