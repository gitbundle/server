// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package migrations

import (
	"fmt"

	"github.com/gitbundle/modules/setting"

	"xorm.io/xorm"
)

func fixLanguageStatsToSaveSize(x *xorm.Engine) error {
	// LanguageStat see models/repo_language_stats.go
	type LanguageStat struct {
		Size int64 `xorm:"NOT NULL DEFAULT 0"`
	}

	// RepoIndexerType specifies the repository indexer type
	type RepoIndexerType int

	const (
		// RepoIndexerTypeCode code indexer
		RepoIndexerTypeCode RepoIndexerType = iota // 0
		// RepoIndexerTypeStats repository stats indexer
		RepoIndexerTypeStats // 1
	)

	// RepoIndexerStatus see models/repo_indexer.go
	type RepoIndexerStatus struct {
		IndexerType RepoIndexerType `xorm:"INDEX(s) NOT NULL DEFAULT 0"`
	}

	if err := x.Sync2(new(LanguageStat)); err != nil {
		return fmt.Errorf("Sync2: %v", err)
	}

	x.Delete(&RepoIndexerStatus{IndexerType: RepoIndexerTypeStats})

	// Delete language stat statuses
	truncExpr := "TRUNCATE TABLE"
	if setting.Database.UseSQLite3 {
		truncExpr = "DELETE FROM"
	}

	// Delete language stats
	if _, err := x.Exec(fmt.Sprintf("%s language_stat", truncExpr)); err != nil {
		return err
	}

	sess := x.NewSession()
	defer sess.Close()
	return dropTableColumns(sess, "language_stat", "percentage")
}
