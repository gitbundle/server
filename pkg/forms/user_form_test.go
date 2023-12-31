// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package forms

import (
	"testing"

	"github.com/gitbundle/modules/setting"

	"github.com/stretchr/testify/assert"
)

func TestRegisterForm_IsDomainAllowed_Empty(t *testing.T) {
	_ = setting.Service

	setting.Service.EmailDomainWhitelist = []string{}

	form := RegisterForm{}

	assert.True(t, form.IsEmailDomainAllowed())
}

func TestRegisterForm_IsDomainAllowed_InvalidEmail(t *testing.T) {
	_ = setting.Service

	setting.Service.EmailDomainWhitelist = []string{"gitea.io"}

	tt := []struct {
		email string
	}{
		{"securitygieqqq"},
		{"hdudhdd"},
	}

	for _, v := range tt {
		form := RegisterForm{Email: v.email}

		assert.False(t, form.IsEmailDomainAllowed())
	}
}

func TestRegisterForm_IsDomainAllowed_WhitelistedEmail(t *testing.T) {
	_ = setting.Service

	setting.Service.EmailDomainWhitelist = []string{"gitea.io"}

	tt := []struct {
		email string
		valid bool
	}{
		{"security@gitea.io", true},
		{"security@gITea.io", true},
		{"hdudhdd", false},
		{"seee@example.com", false},
	}

	for _, v := range tt {
		form := RegisterForm{Email: v.email}

		assert.Equal(t, v.valid, form.IsEmailDomainAllowed())
	}
}

func TestRegisterForm_IsDomainAllowed_BlocklistedEmail(t *testing.T) {
	_ = setting.Service

	setting.Service.EmailDomainWhitelist = []string{}
	setting.Service.EmailDomainBlocklist = []string{"gitea.io"}

	tt := []struct {
		email string
		valid bool
	}{
		{"security@gitea.io", false},
		{"security@gitea.example", true},
		{"hdudhdd", true},
	}

	for _, v := range tt {
		form := RegisterForm{Email: v.email}

		assert.Equal(t, v.valid, form.IsEmailDomainAllowed())
	}
}
