// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package issues_test

import (
	"testing"
	"time"

	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestAddTime(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	user3, err := user_model.GetUserByID(3)
	assert.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	assert.NoError(t, err)

	// 3661 = 1h 1min 1s
	trackedTime, err := issues_model.AddTime(user3, issue1, 3661, time.Now())
	assert.NoError(t, err)
	assert.Equal(t, int64(3), trackedTime.UserID)
	assert.Equal(t, int64(1), trackedTime.IssueID)
	assert.Equal(t, int64(3661), trackedTime.Time)

	tt := unittest.AssertExistsAndLoadBean(t, &issues_model.TrackedTime{UserID: 3, IssueID: 1}).(*issues_model.TrackedTime)
	assert.Equal(t, int64(3661), tt.Time)

	comment := unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{Type: issues_model.CommentTypeAddTimeManual, PosterID: 3, IssueID: 1}).(*issues_model.Comment)
	assert.Equal(t, comment.Content, "1 hour 1 minute")
}

func TestGetTrackedTimes(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	// by Issue
	times, err := issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: 1})
	assert.NoError(t, err)
	assert.Len(t, times, 1)
	assert.Equal(t, int64(400), times[0].Time)

	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: -1})
	assert.NoError(t, err)
	assert.Len(t, times, 0)

	// by User
	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{UserID: 1})
	assert.NoError(t, err)
	assert.Len(t, times, 3)
	assert.Equal(t, int64(400), times[0].Time)

	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{UserID: 3})
	assert.NoError(t, err)
	assert.Len(t, times, 0)

	// by Repo
	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{RepositoryID: 2})
	assert.NoError(t, err)
	assert.Len(t, times, 3)
	assert.Equal(t, int64(1), times[0].Time)
	issue, err := issues_model.GetIssueByID(db.DefaultContext, times[0].IssueID)
	assert.NoError(t, err)
	assert.Equal(t, issue.RepoID, int64(2))

	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{RepositoryID: 1})
	assert.NoError(t, err)
	assert.Len(t, times, 5)

	times, err = issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{RepositoryID: 10})
	assert.NoError(t, err)
	assert.Len(t, times, 0)
}

func TestTotalTimes(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	total, err := issues_model.TotalTimes(&issues_model.FindTrackedTimesOptions{IssueID: 1})
	assert.NoError(t, err)
	assert.Len(t, total, 1)
	for user, time := range total {
		assert.Equal(t, int64(1), user.ID)
		assert.Equal(t, "6 minutes 40 seconds", time)
	}

	total, err = issues_model.TotalTimes(&issues_model.FindTrackedTimesOptions{IssueID: 2})
	assert.NoError(t, err)
	assert.Len(t, total, 2)
	for user, time := range total {
		if user.ID == 2 {
			assert.Equal(t, "1 hour 1 minute", time)
		} else if user.ID == 1 {
			assert.Equal(t, "20 seconds", time)
		} else {
			assert.Error(t, assert.AnError)
		}
	}

	total, err = issues_model.TotalTimes(&issues_model.FindTrackedTimesOptions{IssueID: 5})
	assert.NoError(t, err)
	assert.Len(t, total, 1)
	for user, time := range total {
		assert.Equal(t, int64(2), user.ID)
		assert.Equal(t, "1 second", time)
	}

	total, err = issues_model.TotalTimes(&issues_model.FindTrackedTimesOptions{IssueID: 4})
	assert.NoError(t, err)
	assert.Len(t, total, 2)
}
