// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package nuget

import (
	"fmt"
)

type linkBuilder struct {
	Base string
}

// GetRegistrationIndexURL builds the registration index url
func (l *linkBuilder) GetRegistrationIndexURL(id string) string {
	return fmt.Sprintf("%s/registration/%s/index.json", l.Base, id)
}

// GetRegistrationLeafURL builds the registration leaf url
func (l *linkBuilder) GetRegistrationLeafURL(id, version string) string {
	return fmt.Sprintf("%s/registration/%s/%s.json", l.Base, id, version)
}

// GetPackageDownloadURL builds the download url
func (l *linkBuilder) GetPackageDownloadURL(id, version string) string {
	return fmt.Sprintf("%s/package/%s/%s/%s.%s.nupkg", l.Base, id, version, id, version)
}
