// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package pull_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/gitbundle/modules/queue"
	issues_model "github.com/gitbundle/server/pkg/issues"
	pull "github.com/gitbundle/server/pkg/pull/manager"
	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestPullRequest_AddToTaskQueue(t *testing.T) {
	assert.NoError(t, unittest.PrepareTestDatabase())

	idChan := make(chan int64, 10)

	q, err := queue.NewChannelUniqueQueue(func(data ...queue.Data) []queue.Data {
		for _, datum := range data {
			id, _ := strconv.ParseInt(datum.(string), 10, 64)
			idChan <- id
		}
		return nil
	}, queue.ChannelUniqueQueueConfiguration{
		WorkerPoolConfiguration: queue.WorkerPoolConfiguration{
			QueueLength: 10,
			BatchLength: 1,
			Name:        "temporary-queue",
		},
		Workers: 1,
	}, "")
	assert.NoError(t, err)

	queueShutdown := []func(){}
	queueTerminate := []func(){}

	prPatchCheckerQueue = q.(queue.UniqueQueue)

	pr := unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}).(*issues_model.PullRequest)
	pull.AddToTaskQueue(pr)

	assert.Eventually(t, func() bool {
		pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}).(*issues_model.PullRequest)
		return pr.Status == issues_model.PullRequestStatusChecking
	}, 1*time.Second, 100*time.Millisecond)

	has, err := prPatchCheckerQueue.Has(strconv.FormatInt(pr.ID, 10))
	assert.True(t, has)
	assert.NoError(t, err)

	prPatchCheckerQueue.Run(func(shutdown func()) {
		queueShutdown = append(queueShutdown, shutdown)
	}, func(terminate func()) {
		queueTerminate = append(queueTerminate, terminate)
	})

	select {
	case id := <-idChan:
		assert.EqualValues(t, pr.ID, id)
	case <-time.After(time.Second):
		assert.Fail(t, "Timeout: nothing was added to pullRequestQueue")
	}

	has, err = prPatchCheckerQueue.Has(strconv.FormatInt(pr.ID, 10))
	assert.False(t, has)
	assert.NoError(t, err)

	pr = unittest.AssertExistsAndLoadBean(t, &issues_model.PullRequest{ID: 2}).(*issues_model.PullRequest)
	assert.Equal(t, issues_model.PullRequestStatusChecking, pr.Status)

	for _, callback := range queueShutdown {
		callback()
	}
	for _, callback := range queueTerminate {
		callback()
	}

	prPatchCheckerQueue = nil
}
