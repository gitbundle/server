// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

// a middleware that generates and validates CSRF tokens.

package context

import (
	"encoding/base32"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/modules/util"
	"github.com/gitbundle/server/pkg/web/middleware"
)

// CSRFProtector represents a CSRF protector and is used to get the current token and validate the token.
type CSRFProtector interface {
	// GetHeaderName returns HTTP header to search for token.
	GetHeaderName() string
	// GetFormName returns form value to search for token.
	GetFormName() string
	// GetToken returns the token.
	GetToken() string
	// Validate validates the token in http context.
	Validate(ctx *Context)
}

type csrfProtector struct {
	// Header name value for setting and getting csrf token.
	Header string
	// Form name value for setting and getting csrf token.
	Form string
	// Cookie name value for setting and getting csrf token.
	Cookie string
	// Cookie domain
	CookieDomain string
	// Cookie path
	CookiePath string
	// Cookie HttpOnly flag value used for the csrf token.
	CookieHTTPOnly bool
	// Token generated to pass via header, cookie, or hidden form value.
	Token string
	// This value must be unique per user.
	ID string
	// Secret used along with the unique id above to generate the Token.
	Secret string
}

// GetHeaderName returns the name of the HTTP header for csrf token.
func (c *csrfProtector) GetHeaderName() string {
	return c.Header
}

// GetFormName returns the name of the form value for csrf token.
func (c *csrfProtector) GetFormName() string {
	return c.Form
}

// GetToken returns the current token. This is typically used
// to populate a hidden form in an HTML template.
func (c *csrfProtector) GetToken() string {
	return c.Token
}

// CsrfOptions maintains options to manage behavior of Generate.
type CsrfOptions struct {
	// The global secret value used to generate Tokens.
	Secret string
	// HTTP header used to set and get token.
	Header string
	// Form value used to set and get token.
	Form string
	// Cookie value used to set and get token.
	Cookie string
	// Cookie domain.
	CookieDomain string
	// Cookie path.
	CookiePath     string
	CookieHTTPOnly bool
	// SameSite set the cookie SameSite type
	SameSite http.SameSite
	// Key used for getting the unique ID per user.
	SessionKey string
	// oldSessionKey saves old value corresponding to SessionKey.
	oldSessionKey string
	// If true, send token via X-Csrf-Token header.
	SetHeader bool
	// If true, send token via _csrf cookie.
	SetCookie bool
	// Set the Secure flag to true on the cookie.
	Secure bool
	// Disallow Origin appear in request header.
	Origin bool
	// Cookie lifetime. Default is 0
	CookieLifeTime int
}

func prepareDefaultCsrfOptions(opt CsrfOptions) CsrfOptions {
	if opt.Secret == "" {
		randBytes, err := util.CryptoRandomBytes(8)
		if err != nil {
			// this panic can be handled by the recover() in http handlers
			panic(fmt.Errorf("failed to generate random bytes: %w", err))
		}
		opt.Secret = base32.StdEncoding.EncodeToString(randBytes)
	}
	if opt.Header == "" {
		opt.Header = "X-Csrf-Token"
	}
	if opt.Form == "" {
		opt.Form = "_csrf"
	}
	if opt.Cookie == "" {
		opt.Cookie = "_csrf"
	}
	if opt.CookiePath == "" {
		opt.CookiePath = "/"
	}
	if opt.SessionKey == "" {
		opt.SessionKey = "uid"
	}
	opt.oldSessionKey = "_old_" + opt.SessionKey
	return opt
}

// PrepareCSRFProtector returns a CSRFProtector to be used for every request.
// Additionally, depending on options set, generated tokens will be sent via Header and/or Cookie.
func PrepareCSRFProtector(opt CsrfOptions, ctx *Context) CSRFProtector {
	opt = prepareDefaultCsrfOptions(opt)
	x := &csrfProtector{
		Secret:         opt.Secret,
		Header:         opt.Header,
		Form:           opt.Form,
		Cookie:         opt.Cookie,
		CookieDomain:   opt.CookieDomain,
		CookiePath:     opt.CookiePath,
		CookieHTTPOnly: opt.CookieHTTPOnly,
	}

	if opt.Origin && len(ctx.Req.Header.Get("Origin")) > 0 {
		return x
	}

	x.ID = "0"
	uidAny := ctx.Session.Get(opt.SessionKey)
	if uidAny != nil {
		switch uidVal := uidAny.(type) {
		case string:
			x.ID = uidVal
		case int64:
			x.ID = strconv.FormatInt(uidVal, 10)
		default:
			log.Error("invalid uid type in session: %T", uidAny)
		}
	}

	oldUID := ctx.Session.Get(opt.oldSessionKey)
	uidChanged := oldUID == nil || oldUID.(string) != x.ID
	cookieToken := ctx.GetCookie(opt.Cookie)

	needsNew := true
	if uidChanged {
		_ = ctx.Session.Set(opt.oldSessionKey, x.ID)
	} else if cookieToken != "" {
		// If cookie token presents, re-use existing unexpired token, else generate a new one.
		if issueTime, ok := ParseCsrfToken(cookieToken); ok {
			dur := time.Since(issueTime) // issueTime is not a monotonic-clock, the server time may change a lot to an early time.
			if dur >= -CsrfTokenRegenerationInterval && dur <= CsrfTokenRegenerationInterval {
				x.Token = cookieToken
				needsNew = false
			}
		}
	}

	if needsNew {
		// FIXME: actionId.
		x.Token = GenerateCsrfToken(x.Secret, x.ID, "POST", time.Now())
		if opt.SetCookie {
			var expires interface{}
			if opt.CookieLifeTime == 0 {
				expires = time.Now().Add(CsrfTokenTimeout)
			}
			middleware.SetCookie(ctx.Resp, opt.Cookie, x.Token,
				opt.CookieLifeTime,
				opt.CookiePath,
				opt.CookieDomain,
				opt.Secure,
				opt.CookieHTTPOnly,
				expires,
				middleware.SameSite(opt.SameSite),
			)
		}
	}

	if opt.SetHeader {
		ctx.Resp.Header().Add(opt.Header, x.Token)
	}
	return x
}

func (c *csrfProtector) validateToken(ctx *Context, token string) {
	if !ValidCsrfToken(token, c.Secret, c.ID, "POST", time.Now()) {
		middleware.DeleteCSRFCookie(ctx.Resp)
		if middleware.IsAPIPath(ctx.Req) {
			// currently, there should be no access to the APIPath with CSRF token. because templates shouldn't use the `/api/` endpoints.
			http.Error(ctx.Resp, "Invalid CSRF token.", http.StatusBadRequest)
		} else {
			ctx.Flash.Error(ctx.Tr("error.invalid_csrf"))
			ctx.Redirect(setting.AppSubURL + "/")
		}
	}
}

// Validate should be used as a per route middleware. It attempts to get a token from an "X-Csrf-Token"
// HTTP header and then a "_csrf" form value. If one of these is found, the token will be validated.
// If this validation fails, custom Error is sent in the reply.
// If neither a header nor form value is found, http.StatusBadRequest is sent.
func (c *csrfProtector) Validate(ctx *Context) {
	if token := ctx.Req.Header.Get(c.GetHeaderName()); token != "" {
		c.validateToken(ctx, token)
		return
	}
	if token := ctx.Req.FormValue(c.GetFormName()); token != "" {
		c.validateToken(ctx, token)
		return
	}
	c.validateToken(ctx, "") // no csrf token, use an empty token to respond error
}
