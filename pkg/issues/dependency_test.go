// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package issues_test

import (
	"testing"

	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestCreateIssueDependency(t *testing.T) {
	// Prepare
	assert.NoError(t, unittest.PrepareTestDatabase())

	user1, err := user_model.GetUserByID(1)
	assert.NoError(t, err)

	issue1, err := issues_model.GetIssueByID(db.DefaultContext, 1)
	assert.NoError(t, err)

	issue2, err := issues_model.GetIssueByID(db.DefaultContext, 2)
	assert.NoError(t, err)

	// Create a dependency and check if it was successful
	err = issues_model.CreateIssueDependency(user1, issue1, issue2)
	assert.NoError(t, err)

	// Do it again to see if it will check if the dependency already exists
	err = issues_model.CreateIssueDependency(user1, issue1, issue2)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrDependencyExists(err))

	// Check for circular dependencies
	err = issues_model.CreateIssueDependency(user1, issue2, issue1)
	assert.Error(t, err)
	assert.True(t, issues_model.IsErrCircularDependency(err))

	_ = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{Type: issues_model.CommentTypeAddDependency, PosterID: user1.ID, IssueID: issue1.ID})

	// Check if dependencies left is correct
	left, err := issues_model.IssueNoDependenciesLeft(db.DefaultContext, issue1)
	assert.NoError(t, err)
	assert.False(t, left)

	// Close #2 and check again
	_, err = issues_model.ChangeIssueStatus(db.DefaultContext, issue2, user1, true)
	assert.NoError(t, err)

	left, err = issues_model.IssueNoDependenciesLeft(db.DefaultContext, issue1)
	assert.NoError(t, err)
	assert.True(t, left)

	// Test removing the dependency
	err = issues_model.RemoveIssueDependency(user1, issue1, issue2, issues_model.DependencyTypeBlockedBy)
	assert.NoError(t, err)
}
