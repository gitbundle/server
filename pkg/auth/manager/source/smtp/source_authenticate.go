// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package smtp

import (
	"errors"
	"net/smtp"
	"net/textproto"
	"strings"

	"github.com/gitbundle/modules/util"
	auth_model "github.com/gitbundle/server/pkg/auth"
	"github.com/gitbundle/server/pkg/mailer"
	user_model "github.com/gitbundle/server/pkg/user"
)

// Authenticate queries if the provided login/password is authenticates against the SMTP server
// Users will be autoregistered as required
func (source *Source) Authenticate(user *user_model.User, userName, password string) (*user_model.User, error) {
	// Verify allowed domains.
	if len(source.AllowedDomains) > 0 {
		idx := strings.Index(userName, "@")
		if idx == -1 {
			return nil, user_model.ErrUserNotExist{Name: userName}
		} else if !util.IsStringInSlice(userName[idx+1:], strings.Split(source.AllowedDomains, ","), true) {
			return nil, user_model.ErrUserNotExist{Name: userName}
		}
	}

	var auth smtp.Auth
	switch source.Auth {
	case PlainAuthentication:
		auth = smtp.PlainAuth("", userName, password, source.Host)
	case LoginAuthentication:
		auth = &loginAuthenticator{userName, password}
	case CRAMMD5Authentication:
		auth = smtp.CRAMMD5Auth(userName, password)
	default:
		return nil, errors.New("unsupported SMTP auth type")
	}

	if err := Authenticate(auth, source); err != nil {
		// Check standard error format first,
		// then fallback to worse case.
		tperr, ok := err.(*textproto.Error)
		if (ok && tperr.Code == 535) ||
			strings.Contains(err.Error(), "Username and Password not accepted") {
			return nil, user_model.ErrUserNotExist{Name: userName}
		}
		if (ok && tperr.Code == 534) ||
			strings.Contains(err.Error(), "Application-specific password required") {
			return nil, user_model.ErrUserNotExist{Name: userName}
		}
		return nil, err
	}

	if user != nil {
		return user, nil
	}

	username := userName
	idx := strings.Index(userName, "@")
	if idx > -1 {
		username = userName[:idx]
	}

	user = &user_model.User{
		LowerName:   strings.ToLower(username),
		Name:        strings.ToLower(username),
		Email:       userName,
		Passwd:      password,
		LoginType:   auth_model.SMTP,
		LoginSource: source.authSource.ID,
		LoginName:   userName,
	}
	overwriteDefault := &user_model.CreateUserOverwriteOptions{
		IsActive: util.OptionalBoolTrue,
	}

	if err := user_model.CreateUser(user, overwriteDefault); err != nil {
		return user, err
	}

	mailer.SendRegisterNotifyMail(user)

	return user, nil
}

// IsSkipLocalTwoFA returns if this source should skip local 2fa for password authentication
func (source *Source) IsSkipLocalTwoFA() bool {
	return source.SkipLocalTwoFA
}