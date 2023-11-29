// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package forms

import (
	"net/http"

	"github.com/gitbundle/server/pkg/context"
	"github.com/gitbundle/server/pkg/web/middleware"

	"gitea.com/go-chi/binding"
)

// SignInOpenIDForm form for signing in with OpenID
type SignInOpenIDForm struct {
	Openid   string `binding:"Required;MaxSize(256)"`
	Remember bool
}

// Validate validates the fields
func (f *SignInOpenIDForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// SignUpOpenIDForm form for signin up with OpenID
type SignUpOpenIDForm struct {
	UserName           string `binding:"Required;AlphaDashDot;MaxSize(40)"`
	Email              string `binding:"Required;Email;MaxSize(254)"`
	GRecaptchaResponse string `form:"g-recaptcha-response"`
	HcaptchaResponse   string `form:"h-captcha-response"`
}

// Validate validates the fields
func (f *SignUpOpenIDForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}

// ConnectOpenIDForm form for connecting an existing account to an OpenID URI
type ConnectOpenIDForm struct {
	UserName string `binding:"Required;MaxSize(254)"`
	Password string `binding:"Required;MaxSize(255)"`
}

// Validate validates the fields
func (f *ConnectOpenIDForm) Validate(req *http.Request, errs binding.Errors) binding.Errors {
	ctx := context.GetContext(req)
	return middleware.Validate(errs, ctx.Data, f, ctx.Locale)
}
