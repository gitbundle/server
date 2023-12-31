// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package issues_test

import (
	"testing"

	"github.com/gitbundle/modules/timeutil"
	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestCancelStopwatch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	user1, err := user_model.GetUserByID(1)
	assert.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	assert.NoError(t, err)
	issue2, err := issues_model.GetIssueByID(db.DefaultContext, 2)
	assert.NoError(t, err)

	err = issues_model.CancelStopwatch(user1, issue1)
	assert.NoError(t, err)
	unittest.AssertNotExistsBean(t, &issues_model.Stopwatch{UserID: user1.ID, IssueID: issue1.ID})

	_ = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{Type: issues_model.CommentTypeCancelTracking, PosterID: user1.ID, IssueID: issue1.ID})

	assert.Nil(t, issues_model.CancelStopwatch(user1, issue2))
}

func TestStopwatchExists(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	assert.True(t, issues_model.StopwatchExists(1, 1))
	assert.False(t, issues_model.StopwatchExists(1, 2))
}

func TestHasUserStopwatch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	exists, sw, err := issues_model.HasUserStopwatch(db.DefaultContext, 1)
	assert.NoError(t, err)
	assert.True(t, exists)
	assert.Equal(t, int64(1), sw.ID)

	exists, _, err = issues_model.HasUserStopwatch(db.DefaultContext, 3)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCreateOrStopIssueStopwatch(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	user2, err := user_model.GetUserByID(2)
	assert.NoError(t, err)
	user3, err := user_model.GetUserByID(3)
	assert.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	assert.NoError(t, err)
	issue2, err := issues_model.GetIssueByID(db.DefaultContext, 2)
	assert.NoError(t, err)

	assert.NoError(t, issues_model.CreateOrStopIssueStopwatch(user3, issue1))
	sw := unittest.AssertExistsAndLoadBean(t, &issues_model.Stopwatch{UserID: 3, IssueID: 1}).(*issues_model.Stopwatch)
	assert.LessOrEqual(t, sw.CreatedUnix, timeutil.TimeStampNow())

	assert.NoError(t, issues_model.CreateOrStopIssueStopwatch(user2, issue2))
	unittest.AssertNotExistsBean(t, &issues_model.Stopwatch{UserID: 2, IssueID: 2})
	unittest.AssertExistsAndLoadBean(t, &issues_model.TrackedTime{UserID: 2, IssueID: 2})
}
