// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package repo

import "github.com/gitbundle/server/pkg/db"

// MergeStyle represents the approach to merge commits into base branch.
type MergeStyle string

const (
	// MergeStyleMerge create merge commit
	MergeStyleMerge MergeStyle = "merge"
	// MergeStyleRebase rebase before merging
	MergeStyleRebase MergeStyle = "rebase"
	// MergeStyleRebaseMerge rebase before merging with merge commit (--no-ff)
	MergeStyleRebaseMerge MergeStyle = "rebase-merge"
	// MergeStyleSquash squash commits into single commit before merging
	MergeStyleSquash MergeStyle = "squash"
	// MergeStyleManuallyMerged pr has been merged manually, just mark it as merged directly
	MergeStyleManuallyMerged MergeStyle = "manually-merged"
	// MergeStyleRebaseUpdate not a merge style, used to update pull head by rebase
	MergeStyleRebaseUpdate MergeStyle = "rebase-update-only"
)

// UpdateDefaultBranch updates the default branch
func UpdateDefaultBranch(repo *Repository) error {
	_, err := db.GetEngine(db.DefaultContext).ID(repo.ID).Cols("default_branch").Update(repo)
	return err
}
