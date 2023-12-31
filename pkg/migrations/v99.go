// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package migrations

import (
	"github.com/gitbundle/modules/timeutil"

	"xorm.io/xorm"
)

func addTaskTable(x *xorm.Engine) error {
	// TaskType defines task type
	type TaskType int

	// TaskStatus defines task status
	type TaskStatus int

	type Task struct {
		ID             int64
		DoerID         int64 `xorm:"index"` // operator
		OwnerID        int64 `xorm:"index"` // repo owner id, when creating, the repoID maybe zero
		RepoID         int64 `xorm:"index"`
		Type           TaskType
		Status         TaskStatus `xorm:"index"`
		StartTime      timeutil.TimeStamp
		EndTime        timeutil.TimeStamp
		PayloadContent string             `xorm:"TEXT"`
		Errors         string             `xorm:"TEXT"` // if task failed, saved the error reason
		Created        timeutil.TimeStamp `xorm:"created"`
	}

	type Repository struct {
		Status int `xorm:"NOT NULL DEFAULT 0"`
	}

	return x.Sync2(new(Task), new(Repository))
}
