// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package doctor

import (
	"context"

	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/server/pkg/db"
	"github.com/gitbundle/server/pkg/migrations"
)

func checkDBVersion(ctx context.Context, logger log.Logger, autofix bool) error {
	if err := db.InitEngineWithMigration(ctx, migrations.EnsureUpToDate); err != nil {
		if !autofix {
			logger.Critical("Error: %v during ensure up to date", err)
			return err
		}
		logger.Warn("Got Error: %v during ensure up to date", err)
		logger.Warn("Attempting to migrate to the latest DB version to fix this.")

		err = db.InitEngineWithMigration(ctx, migrations.Migrate)
		if err != nil {
			logger.Critical("Error: %v during migration", err)
		}
		return err
	}
	return nil
}

func init() {
	Register(&Check{
		Title:         "Check Database Version",
		Name:          "check-db-version",
		IsDefault:     true,
		Run:           checkDBVersion,
		AbortIfFailed: false,
		Priority:      2,
	})
}
