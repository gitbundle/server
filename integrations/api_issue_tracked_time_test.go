// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package integrations

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	api "github.com/gitbundle/api/pkg/structs"
	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestAPIGetTrackedTimes(t *testing.T) {
	defer prepareTestEnv(t)()

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
	issue2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2}).(*issues_model.Issue)
	assert.NoError(t, issue2.LoadRepo(db.DefaultContext))

	session := loginUser(t, user2.Name)
	token := getTokenForLoggedInUser(t, session)

	req := NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/%d/times?token=%s", user2.Name, issue2.Repo.Name, issue2.Index, token)
	resp := session.MakeRequest(t, req, http.StatusOK)
	var apiTimes api.TrackedTimeList
	DecodeJSON(t, resp, &apiTimes)
	expect, err := issues_model.GetTrackedTimes(db.DefaultContext, &issues_model.FindTrackedTimesOptions{IssueID: issue2.ID})
	assert.NoError(t, err)
	assert.Len(t, apiTimes, 3)

	for i, time := range expect {
		assert.Equal(t, time.ID, apiTimes[i].ID)
		assert.EqualValues(t, issue2.Title, apiTimes[i].Issue.Title)
		assert.EqualValues(t, issue2.ID, apiTimes[i].IssueID)
		assert.Equal(t, time.Created.Unix(), apiTimes[i].Created.Unix())
		assert.Equal(t, time.Time, apiTimes[i].Time)
		user, err := user_model.GetUserByID(time.UserID)
		assert.NoError(t, err)
		assert.Equal(t, user.Name, apiTimes[i].UserName)
	}

	// test filter
	since := "2000-01-01T00%3A00%3A02%2B00%3A00"  // 946684802
	before := "2000-01-01T00%3A00%3A12%2B00%3A00" // 946684812

	req = NewRequestf(t, "GET", "/api/v1/repos/%s/%s/issues/%d/times?since=%s&before=%s&token=%s", user2.Name, issue2.Repo.Name, issue2.Index, since, before, token)
	resp = session.MakeRequest(t, req, http.StatusOK)
	var filterAPITimes api.TrackedTimeList
	DecodeJSON(t, resp, &filterAPITimes)
	assert.Len(t, filterAPITimes, 2)
	assert.Equal(t, int64(3), filterAPITimes[0].ID)
	assert.Equal(t, int64(6), filterAPITimes[1].ID)
}

func TestAPIDeleteTrackedTime(t *testing.T) {
	defer prepareTestEnv(t)()

	time6 := unittest.AssertExistsAndLoadBean(t, &issues_model.TrackedTime{ID: 6}).(*issues_model.TrackedTime)
	issue2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2}).(*issues_model.Issue)
	assert.NoError(t, issue2.LoadRepo(db.DefaultContext))
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)

	session := loginUser(t, user2.Name)
	token := getTokenForLoggedInUser(t, session)

	// Deletion not allowed
	req := NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s/issues/%d/times/%d?token=%s", user2.Name, issue2.Repo.Name, issue2.Index, time6.ID, token)
	session.MakeRequest(t, req, http.StatusForbidden)

	time3 := unittest.AssertExistsAndLoadBean(t, &issues_model.TrackedTime{ID: 3}).(*issues_model.TrackedTime)
	req = NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s/issues/%d/times/%d?token=%s", user2.Name, issue2.Repo.Name, issue2.Index, time3.ID, token)
	session.MakeRequest(t, req, http.StatusNoContent)
	// Delete non existing time
	session.MakeRequest(t, req, http.StatusNotFound)

	// Reset time of user 2 on issue 2
	trackedSeconds, err := issues_model.GetTrackedSeconds(db.DefaultContext, issues_model.FindTrackedTimesOptions{IssueID: 2, UserID: 2})
	assert.NoError(t, err)
	assert.Equal(t, int64(3661), trackedSeconds)

	req = NewRequestf(t, "DELETE", "/api/v1/repos/%s/%s/issues/%d/times?token=%s", user2.Name, issue2.Repo.Name, issue2.Index, token)
	session.MakeRequest(t, req, http.StatusNoContent)
	session.MakeRequest(t, req, http.StatusNotFound)

	trackedSeconds, err = issues_model.GetTrackedSeconds(db.DefaultContext, issues_model.FindTrackedTimesOptions{IssueID: 2, UserID: 2})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), trackedSeconds)
}

func TestAPIAddTrackedTimes(t *testing.T) {
	defer prepareTestEnv(t)()

	issue2 := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2}).(*issues_model.Issue)
	assert.NoError(t, issue2.LoadRepo(db.DefaultContext))
	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
	admin := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 1}).(*user_model.User)

	session := loginUser(t, admin.Name)
	token := getTokenForLoggedInUser(t, session)

	urlStr := fmt.Sprintf("/api/v1/repos/%s/%s/issues/%d/times?token=%s", user2.Name, issue2.Repo.Name, issue2.Index, token)

	req := NewRequestWithJSON(t, "POST", urlStr, &api.AddTimeOption{
		Time:    33,
		User:    user2.Name,
		Created: time.Unix(947688818, 0),
	})
	resp := session.MakeRequest(t, req, http.StatusOK)
	var apiNewTime api.TrackedTime
	DecodeJSON(t, resp, &apiNewTime)

	assert.EqualValues(t, 33, apiNewTime.Time)
	assert.EqualValues(t, user2.ID, apiNewTime.UserID)
	assert.EqualValues(t, 947688818, apiNewTime.Created.Unix())
}
