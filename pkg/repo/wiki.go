// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package repo

import (
	"path/filepath"
	"strings"

	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/util"
	user_model "github.com/gitbundle/server/pkg/user"
)

// WikiCloneLink returns clone URLs of repository wiki.
func (repo *Repository) WikiCloneLink() *CloneLink {
	return repo.cloneLink(true)
}

// WikiPath returns wiki data path by given user and repository name.
func WikiPath(userName, repoName string) string {
	return filepath.Join(user_model.UserPath(userName), strings.ToLower(repoName)+".wiki.git")
}

// WikiPath returns wiki data path for given repository.
func (repo *Repository) WikiPath() string {
	return WikiPath(repo.OwnerName, repo.Name)
}

// HasWiki returns true if repository has wiki.
func (repo *Repository) HasWiki() bool {
	isDir, err := util.IsDir(repo.WikiPath())
	if err != nil {
		log.Error("Unable to check if %s is a directory: %v", repo.WikiPath(), err)
	}
	return isDir
}
