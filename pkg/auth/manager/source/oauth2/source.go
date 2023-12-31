// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package oauth2

import (
	"github.com/gitbundle/modules/json"
	"github.com/gitbundle/server/pkg/auth"
)

// ________      _____          __  .__     ________
// \_____  \    /  _  \  __ ___/  |_|  |__  \_____  \
// /   |   \  /  /_\  \|  |  \   __\  |  \  /  ____/
// /    |    \/    |    \  |  /|  | |   Y  \/       \
// \_______  /\____|__  /____/ |__| |___|  /\_______ \
//         \/         \/                 \/         \/

// Source holds configuration for the OAuth2 login source.
type Source struct {
	Provider                      string
	ClientID                      string
	ClientSecret                  string
	OpenIDConnectAutoDiscoveryURL string
	CustomURLMapping              *CustomURLMapping
	IconURL                       string

	Scopes             []string
	RequiredClaimName  string
	RequiredClaimValue string
	GroupClaimName     string
	AdminGroup         string
	RestrictedGroup    string
	SkipLocalTwoFA     bool `json:",omitempty"`

	// reference to the authSource
	authSource *auth.Source
}

// FromDB fills up an OAuth2Config from serialized format.
func (source *Source) FromDB(bs []byte) error {
	return json.UnmarshalHandleDoubleEncode(bs, &source)
}

// ToDB exports an SMTPConfig to a serialized format.
func (source *Source) ToDB() ([]byte, error) {
	return json.Marshal(source)
}

// SetAuthSource sets the related AuthSource
func (source *Source) SetAuthSource(authSource *auth.Source) {
	source.authSource = authSource
}

func init() {
	auth.RegisterTypeConfig(auth.OAuth2, &Source{})
}
