// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/context"
	user_model "github.com/gitbundle/server/pkg/user"
)

// https://datatracker.ietf.org/doc/html/draft-ietf-appsawg-webfinger-14#section-4.4

type webfingerJRD struct {
	Subject    string                 `json:"subject,omitempty"`
	Aliases    []string               `json:"aliases,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	Links      []*webfingerLink       `json:"links,omitempty"`
}

type webfingerLink struct {
	Rel        string                 `json:"rel,omitempty"`
	Type       string                 `json:"type,omitempty"`
	Href       string                 `json:"href,omitempty"`
	Titles     map[string]string      `json:"titles,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// WebfingerQuery returns informations about a resource
// https://datatracker.ietf.org/doc/html/rfc7565
func WebfingerQuery(ctx *context.Context) {
	appURL, _ := url.Parse(setting.AppURL)

	resource, err := url.Parse(ctx.FormTrim("resource"))
	if err != nil {
		ctx.Error(http.StatusBadRequest)
		return
	}

	var u *user_model.User

	switch resource.Scheme {
	case "acct":
		// allow only the current host
		parts := strings.SplitN(resource.Opaque, "@", 2)
		if len(parts) != 2 {
			ctx.Error(http.StatusBadRequest)
			return
		}
		if parts[1] != appURL.Host {
			ctx.Error(http.StatusBadRequest)
			return
		}

		u, err = user_model.GetUserByName(ctx, parts[0])
	case "mailto":
		u, err = user_model.GetUserByEmailContext(ctx, resource.Opaque)
		if u != nil && u.KeepEmailPrivate {
			err = user_model.ErrUserNotExist{}
		}
	default:
		ctx.Error(http.StatusBadRequest)
		return
	}
	if err != nil {
		if user_model.IsErrUserNotExist(err) {
			ctx.Error(http.StatusNotFound)
		} else {
			log.Error("Error getting user: %s Error: %v", resource.Opaque, err)
			ctx.Error(http.StatusInternalServerError)
		}
		return
	}

	if !user_model.IsUserVisibleToViewer(ctx, u, ctx.Doer) {
		ctx.Error(http.StatusNotFound)
		return
	}

	aliases := []string{
		u.HTMLURL(),
	}
	if !u.KeepEmailPrivate {
		aliases = append(aliases, fmt.Sprintf("mailto:%s", u.Email))
	}

	links := []*webfingerLink{
		{
			Rel:  "http://webfinger.net/rel/profile-page",
			Type: "text/html",
			Href: u.HTMLURL(),
		},
		{
			Rel:  "http://webfinger.net/rel/avatar",
			Href: u.AvatarLink(),
		},
	}

	ctx.JSON(http.StatusOK, &webfingerJRD{
		Subject: fmt.Sprintf("acct:%s@%s", url.QueryEscape(u.Name), appURL.Host),
		Aliases: aliases,
		Links:   links,
	})
}
