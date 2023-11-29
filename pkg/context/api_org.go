// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package context

import "github.com/gitbundle/server/pkg/organization"

// APIOrganization contains organization and team
type APIOrganization struct {
	Organization *organization.Organization
	Team         *organization.Team
}
