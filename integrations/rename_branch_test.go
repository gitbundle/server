// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package integrations

import (
	"net/http"
	"testing"

	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/unittest"

	"github.com/stretchr/testify/assert"
)

func TestRenameBranch(t *testing.T) {
	// get branch setting page
	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/user2/repo1/settings/branches")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)

	postData := map[string]string{
		"_csrf": htmlDoc.GetCSRF(),
		"from":  "master",
		"to":    "main",
	}
	req = NewRequestWithValues(t, "POST", "/user2/repo1/settings/rename_branch", postData)
	session.MakeRequest(t, req, http.StatusSeeOther)

	// check new branch link
	req = NewRequestWithValues(t, "GET", "/user2/repo1/src/branch/main/README.md", postData)
	session.MakeRequest(t, req, http.StatusOK)

	// check old branch link
	req = NewRequestWithValues(t, "GET", "/user2/repo1/src/branch/master/README.md", postData)
	resp = session.MakeRequest(t, req, http.StatusSeeOther)
	location := resp.HeaderMap.Get("Location")
	assert.Equal(t, "/user2/repo1/src/branch/main/README.md", location)

	// check db
	repo1 := unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1}).(*repo_model.Repository)
	assert.Equal(t, "main", repo1.DefaultBranch)
}
