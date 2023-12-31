// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package migrations

import (
	"xorm.io/xorm"
)

func addIssueResourceIndexTable(x *xorm.Engine) error {
	type ResourceIndex struct {
		GroupID  int64 `xorm:"pk"`
		MaxIndex int64 `xorm:"index"`
	}

	sess := x.NewSession()
	defer sess.Close()

	if err := sess.Begin(); err != nil {
		return err
	}

	if err := sess.Table("issue_index").Sync2(new(ResourceIndex)); err != nil {
		return err
	}

	// Remove data we're goint to rebuild
	if _, err := sess.Table("issue_index").Where("1=1").Delete(&ResourceIndex{}); err != nil {
		return err
	}

	// Create current data for all repositories with issues and PRs
	if _, err := sess.Exec("INSERT INTO issue_index (group_id, max_index) " +
		"SELECT max_data.repo_id, max_data.max_index " +
		"FROM ( SELECT issue.repo_id AS repo_id, max(issue.`index`) AS max_index " +
		"FROM issue GROUP BY issue.repo_id) AS max_data"); err != nil {
		return err
	}

	return sess.Commit()
}
