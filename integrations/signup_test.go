// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package integrations

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/modules/translation/i18n"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestSignup(t *testing.T) {
	defer prepareTestEnv(t)()

	setting.Service.EnableCaptcha = false

	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "exampleUser",
		"email":     "exampleUser@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)

	// should be able to view new user's page
	req = NewRequest(t, "GET", "/exampleUser")
	MakeRequest(t, req, http.StatusOK)
}

func TestSignupAsRestricted(t *testing.T) {
	defer prepareTestEnv(t)()

	setting.Service.EnableCaptcha = false
	setting.Service.DefaultUserIsRestricted = true

	req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
		"user_name": "restrictedUser",
		"email":     "restrictedUser@example.com",
		"password":  "examplePassword!1",
		"retype":    "examplePassword!1",
	})
	MakeRequest(t, req, http.StatusSeeOther)

	// should be able to view new user's page
	req = NewRequest(t, "GET", "/restrictedUser")
	MakeRequest(t, req, http.StatusOK)

	user2 := unittest.AssertExistsAndLoadBean(t, &user_model.User{Name: "restrictedUser"}).(*user_model.User)
	assert.True(t, user2.IsRestricted)
}

func TestSignupEmail(t *testing.T) {
	defer prepareTestEnv(t)()

	setting.Service.EnableCaptcha = false

	tests := []struct {
		email      string
		wantStatus int
		wantMsg    string
	}{
		{"exampleUser@example.com\r\n", http.StatusOK, i18n.Tr("en", "form.email_invalid")},
		{"exampleUser@example.com\r", http.StatusOK, i18n.Tr("en", "form.email_invalid")},
		{"exampleUser@example.com\n", http.StatusOK, i18n.Tr("en", "form.email_invalid")},
		{"exampleUser@example.com", http.StatusSeeOther, ""},
	}

	for i, test := range tests {
		req := NewRequestWithValues(t, "POST", "/user/sign_up", map[string]string{
			"user_name": fmt.Sprintf("exampleUser%d", i),
			"email":     test.email,
			"password":  "examplePassword!1",
			"retype":    "examplePassword!1",
		})
		resp := MakeRequest(t, req, test.wantStatus)
		if test.wantMsg != "" {
			htmlDoc := NewHTMLParser(t, resp.Body)
			assert.Equal(t,
				test.wantMsg,
				strings.TrimSpace(htmlDoc.doc.Find(".ui.message").Text()),
			)
		}
	}
}
