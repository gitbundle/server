// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package integrations

import (
	"net/http"
	"testing"

	"github.com/gitbundle/modules/setting"

	"github.com/stretchr/testify/assert"
)

func TestSettingShowUserEmailExplore(t *testing.T) {
	defer prepareTestEnv(t)()

	showUserEmail := setting.UI.ShowUserEmail
	setting.UI.ShowUserEmail = true

	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/explore/users")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assert.Contains(t,
		htmlDoc.doc.Find(".ui.user.list").Text(),
		"user4@example.com",
	)

	setting.UI.ShowUserEmail = false

	req = NewRequest(t, "GET", "/explore/users")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	assert.NotContains(t,
		htmlDoc.doc.Find(".ui.user.list").Text(),
		"user4@example.com",
	)

	setting.UI.ShowUserEmail = showUserEmail
}

func TestSettingShowUserEmailProfile(t *testing.T) {
	defer prepareTestEnv(t)()

	showUserEmail := setting.UI.ShowUserEmail
	setting.UI.ShowUserEmail = true

	session := loginUser(t, "user2")
	req := NewRequest(t, "GET", "/user2")
	resp := session.MakeRequest(t, req, http.StatusOK)
	htmlDoc := NewHTMLParser(t, resp.Body)
	assert.Contains(t,
		htmlDoc.doc.Find(".user.profile").Text(),
		"user2@example.com",
	)

	setting.UI.ShowUserEmail = false

	req = NewRequest(t, "GET", "/user2")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	// Should contain since this user owns the profile page
	assert.Contains(t,
		htmlDoc.doc.Find(".user.profile").Text(),
		"user2@example.com",
	)

	setting.UI.ShowUserEmail = showUserEmail

	session = loginUser(t, "user4")
	req = NewRequest(t, "GET", "/user2")
	resp = session.MakeRequest(t, req, http.StatusOK)
	htmlDoc = NewHTMLParser(t, resp.Body)
	assert.NotContains(t,
		htmlDoc.doc.Find(".user.profile").Text(),
		"user2@example.com",
	)
}

func TestSettingLandingPage(t *testing.T) {
	defer prepareTestEnv(t)()

	landingPage := setting.LandingPageURL

	setting.LandingPageURL = setting.LandingPageHome
	req := NewRequest(t, "GET", "/")
	MakeRequest(t, req, http.StatusOK)

	setting.LandingPageURL = setting.LandingPageExplore
	req = NewRequest(t, "GET", "/")
	resp := MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/explore", resp.Header().Get("Location"))

	setting.LandingPageURL = setting.LandingPageOrganizations
	req = NewRequest(t, "GET", "/")
	resp = MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/explore/organizations", resp.Header().Get("Location"))

	setting.LandingPageURL = setting.LandingPageLogin
	req = NewRequest(t, "GET", "/")
	resp = MakeRequest(t, req, http.StatusSeeOther)
	assert.Equal(t, "/user/login", resp.Header().Get("Location"))

	setting.LandingPageURL = landingPage
}
