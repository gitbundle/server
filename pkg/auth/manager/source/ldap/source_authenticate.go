// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package ldap

import (
	"fmt"
	"strings"

	"github.com/gitbundle/modules/util"
	asymkey_model "github.com/gitbundle/server/pkg/asymkey"
	"github.com/gitbundle/server/pkg/auth"
	"github.com/gitbundle/server/pkg/db"
	"github.com/gitbundle/server/pkg/mailer"
	"github.com/gitbundle/server/pkg/organization"
	user_model "github.com/gitbundle/server/pkg/user"
	user_service "github.com/gitbundle/server/pkg/user/manager"
)

// Authenticate queries if login/password is valid against the LDAP directory pool,
// and create a local user if success when enabled.
func (source *Source) Authenticate(user *user_model.User, userName, password string) (*user_model.User, error) {
	loginName := userName
	if user != nil {
		loginName = user.LoginName
	}
	sr := source.SearchEntry(loginName, password, source.authSource.Type == auth.DLDAP)
	if sr == nil {
		// User not in LDAP, do nothing
		return nil, user_model.ErrUserNotExist{Name: loginName}
	}

	isAttributeSSHPublicKeySet := len(strings.TrimSpace(source.AttributeSSHPublicKey)) > 0

	// Update User admin flag if exist
	if isExist, err := user_model.IsUserExist(db.DefaultContext, 0, sr.Username); err != nil {
		return nil, err
	} else if isExist {
		if user == nil {
			user, err = user_model.GetUserByName(db.DefaultContext, sr.Username)
			if err != nil {
				return nil, err
			}
		}
		if user != nil && !user.ProhibitLogin {
			cols := make([]string, 0)
			if len(source.AdminFilter) > 0 && user.IsAdmin != sr.IsAdmin {
				// Change existing admin flag only if AdminFilter option is set
				user.IsAdmin = sr.IsAdmin
				cols = append(cols, "is_admin")
			}
			if !user.IsAdmin && len(source.RestrictedFilter) > 0 && user.IsRestricted != sr.IsRestricted {
				// Change existing restricted flag only if RestrictedFilter option is set
				user.IsRestricted = sr.IsRestricted
				cols = append(cols, "is_restricted")
			}
			if len(cols) > 0 {
				err = user_model.UpdateUserCols(db.DefaultContext, user, cols...)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	if user != nil {
		if source.GroupsEnabled && (source.GroupTeamMap != "" || source.GroupTeamMapRemoval) {
			orgCache := make(map[string]*organization.Organization)
			teamCache := make(map[string]*organization.Team)
			source.SyncLdapGroupsToTeams(user, sr.LdapTeamAdd, sr.LdapTeamRemove, orgCache, teamCache)
		}
		if isAttributeSSHPublicKeySet && asymkey_model.SynchronizePublicKeys(user, source.authSource, sr.SSHPublicKey) {
			return user, asymkey_model.RewriteAllPublicKeys()
		}
		return user, nil
	}

	// Fallback.
	if len(sr.Username) == 0 {
		sr.Username = userName
	}

	if len(sr.Mail) == 0 {
		sr.Mail = fmt.Sprintf("%s@localhost", sr.Username)
	}

	user = &user_model.User{
		LowerName:   strings.ToLower(sr.Username),
		Name:        sr.Username,
		FullName:    composeFullName(sr.Name, sr.Surname, sr.Username),
		Email:       sr.Mail,
		LoginType:   source.authSource.Type,
		LoginSource: source.authSource.ID,
		LoginName:   userName,
		IsAdmin:     sr.IsAdmin,
	}
	overwriteDefault := &user_model.CreateUserOverwriteOptions{
		IsRestricted: util.OptionalBoolOf(sr.IsRestricted),
		IsActive:     util.OptionalBoolTrue,
	}

	err := user_model.CreateUser(user, overwriteDefault)
	if err != nil {
		return user, err
	}

	mailer.SendRegisterNotifyMail(user)

	if isAttributeSSHPublicKeySet && asymkey_model.AddPublicKeysBySource(user, source.authSource, sr.SSHPublicKey) {
		err = asymkey_model.RewriteAllPublicKeys()
	}
	if err == nil && len(source.AttributeAvatar) > 0 {
		_ = user_service.UploadAvatar(user, sr.Avatar)
	}
	if source.GroupsEnabled && (source.GroupTeamMap != "" || source.GroupTeamMapRemoval) {
		orgCache := make(map[string]*organization.Organization)
		teamCache := make(map[string]*organization.Team)
		source.SyncLdapGroupsToTeams(user, sr.LdapTeamAdd, sr.LdapTeamRemove, orgCache, teamCache)
	}

	return user, err
}

// IsSkipLocalTwoFA returns if this source should skip local 2fa for password authentication
func (source *Source) IsSkipLocalTwoFA() bool {
	return source.SkipLocalTwoFA
}
