// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package migrations

import (
	"fmt"

	"xorm.io/xorm"
)

func addTableCommitStatusIndex(x *xorm.Engine) error {
	// CommitStatusIndex represents a table for commit status index
	type CommitStatusIndex struct {
		ID       int64
		RepoID   int64  `xorm:"unique(repo_sha)"`
		SHA      string `xorm:"unique(repo_sha)"`
		MaxIndex int64  `xorm:"index"`
	}

	if err := x.Sync2(new(CommitStatusIndex)); err != nil {
		return fmt.Errorf("Sync2: %v", err)
	}

	sess := x.NewSession()
	defer sess.Close()

	if err := sess.Begin(); err != nil {
		return err
	}

	// Remove data we're goint to rebuild
	if _, err := sess.Table("commit_status_index").Where("1=1").Delete(&CommitStatusIndex{}); err != nil {
		return err
	}

	// Create current data for all repositories with issues and PRs
	if _, err := sess.Exec("INSERT INTO commit_status_index (repo_id, sha, max_index) " +
		"SELECT max_data.repo_id, max_data.sha, max_data.max_index " +
		"FROM ( SELECT commit_status.repo_id AS repo_id, commit_status.sha AS sha, max(commit_status.`index`) AS max_index " +
		"FROM commit_status GROUP BY commit_status.repo_id, commit_status.sha) AS max_data"); err != nil {
		return err
	}

	return sess.Commit()
}
