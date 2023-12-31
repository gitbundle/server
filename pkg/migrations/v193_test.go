// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_addRepoIDForAttachment(t *testing.T) {
	type Attachment struct {
		ID         int64  `xorm:"pk autoincr"`
		UUID       string `xorm:"uuid UNIQUE"`
		RepoID     int64  `xorm:"INDEX"` // this should not be zero
		IssueID    int64  `xorm:"INDEX"` // maybe zero when creating
		ReleaseID  int64  `xorm:"INDEX"` // maybe zero when creating
		UploaderID int64  `xorm:"INDEX DEFAULT 0"`
	}

	type Issue struct {
		ID     int64
		RepoID int64
	}

	type Release struct {
		ID     int64
		RepoID int64
	}

	// Prepare and load the testing database
	x, deferrable := prepareTestEnv(t, 0, new(Attachment), new(Issue), new(Release))
	defer deferrable()
	if x == nil || t.Failed() {
		return
	}

	// Run the migration
	if err := addRepoIDForAttachment(x); err != nil {
		assert.NoError(t, err)
		return
	}

	var issueAttachments []*Attachment
	err := x.Where("issue_id > 0").Find(&issueAttachments)
	assert.NoError(t, err)
	for _, attach := range issueAttachments {
		assert.Greater(t, attach.RepoID, 0)
		assert.Greater(t, attach.IssueID, 0)
		var issue Issue
		has, err := x.ID(attach.IssueID).Get(&issue)
		assert.NoError(t, err)
		assert.True(t, has)
		assert.EqualValues(t, attach.RepoID, issue.RepoID)
	}

	var releaseAttachments []*Attachment
	err = x.Where("release_id > 0").Find(&releaseAttachments)
	assert.NoError(t, err)
	for _, attach := range releaseAttachments {
		assert.Greater(t, attach.RepoID, 0)
		assert.Greater(t, attach.IssueID, 0)
		var release Release
		has, err := x.ID(attach.ReleaseID).Get(&release)
		assert.NoError(t, err)
		assert.True(t, has)
		assert.EqualValues(t, attach.RepoID, release.RepoID)
	}
}
