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

func addUploaderIDForAttachment(x *xorm.Engine) error {
	type Attachment struct {
		ID            int64  `xorm:"pk autoincr"`
		UUID          string `xorm:"uuid UNIQUE"`
		IssueID       int64  `xorm:"INDEX"`
		ReleaseID     int64  `xorm:"INDEX"`
		UploaderID    int64  `xorm:"INDEX DEFAULT 0"`
		CommentID     int64
		Name          string
		DownloadCount int64              `xorm:"DEFAULT 0"`
		Size          int64              `xorm:"DEFAULT 0"`
		CreatedUnix   timeutil.TimeStamp `xorm:"created"`
	}

	return x.Sync2(new(Attachment))
}
