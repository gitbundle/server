// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package user_heatmap_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/gitbundle/modules/json"
	"github.com/gitbundle/modules/timeutil"
	"github.com/gitbundle/server/pkg/action"
	"github.com/gitbundle/server/pkg/db"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"
	"github.com/gitbundle/server/pkg/user_heatmap"

	"github.com/stretchr/testify/assert"
)

func TestGetUserHeatmapDataByUser(t *testing.T) {
	testCases := []struct {
		desc        string
		userID      int64
		doerID      int64
		CountResult int
		JSONResult  string
	}{
		{
			"self looks at action in private repo",
			2, 2, 1, `[{"timestamp":1603227600,"contributions":1}]`,
		},
		{
			"admin looks at action in private repo",
			2, 1, 1, `[{"timestamp":1603227600,"contributions":1}]`,
		},
		{
			"other user looks at action in private repo",
			2, 3, 0, `[]`,
		},
		{
			"nobody looks at action in private repo",
			2, 0, 0, `[]`,
		},
		{
			"collaborator looks at action in private repo",
			16, 15, 1, `[{"timestamp":1603267200,"contributions":1}]`,
		},
		{
			"no action action not performed by target user",
			3, 3, 0, `[]`,
		},
		{
			"multiple actions performed with two grouped together",
			10, 10, 3, `[{"timestamp":1603009800,"contributions":1},{"timestamp":1603010700,"contributions":2}]`,
		},
	}
	// Prepare
	assert.NoError(t, unittest.PrepareTestDatabase())

	// Mock time
	timeutil.Set(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC))
	defer timeutil.Unset()

	for _, tc := range testCases {
		user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: tc.userID}).(*user_model.User)

		doer := &user_model.User{ID: tc.doerID}
		_, err := unittest.LoadBeanIfExists(doer)
		assert.NoError(t, err)
		if tc.doerID == 0 {
			doer = nil
		}

		// get the action for comparison
		actions, err := action.GetFeeds(db.DefaultContext, action.GetFeedsOptions{
			RequestedUser:   user,
			Actor:           doer,
			IncludePrivate:  true,
			OnlyPerformedBy: true,
			IncludeDeleted:  true,
		})
		assert.NoError(t, err)

		// Get the heatmap and compare
		heatmap, err := user_heatmap.GetUserHeatmapDataByUser(user, doer)
		var contributions int
		for _, hm := range heatmap {
			contributions += int(hm.Contributions)
		}
		assert.NoError(t, err)
		assert.Len(t, actions, contributions, "invalid action count: did the test data became too old?")
		assert.Equal(t, tc.CountResult, contributions, fmt.Sprintf("testcase '%s'", tc.desc))

		// Test JSON rendering
		jsonData, err := json.Marshal(heatmap)
		assert.NoError(t, err)
		assert.Equal(t, tc.JSONResult, string(jsonData))
	}
}
