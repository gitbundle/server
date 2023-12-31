// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package convert

import (
	"net/url"
	"time"

	api "github.com/gitbundle/api/pkg/structs"
	"github.com/gitbundle/modules/git"
	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/util"
	"github.com/gitbundle/server/pkg/git/gitdiff"
	repo_model "github.com/gitbundle/server/pkg/repo"
	user_model "github.com/gitbundle/server/pkg/user"
)

// ToCommitUser convert a git.Signature to an api.CommitUser
func ToCommitUser(sig *git.Signature) *api.CommitUser {
	return &api.CommitUser{
		Identity: api.Identity{
			Name:  sig.Name,
			Email: sig.Email,
		},
		Date: sig.When.UTC().Format(time.RFC3339),
	}
}

// ToCommitMeta convert a git.Tag to an api.CommitMeta
func ToCommitMeta(repo *repo_model.Repository, tag *git.Tag) *api.CommitMeta {
	return &api.CommitMeta{
		SHA:     tag.Object.String(),
		URL:     util.URLJoin(repo.APIURL(), "git/commits", tag.ID.String()),
		Created: tag.Tagger.When,
	}
}

// ToPayloadCommit convert a git.Commit to api.PayloadCommit
func ToPayloadCommit(repo *repo_model.Repository, c *git.Commit) *api.PayloadCommit {
	authorUsername := ""
	if author, err := user_model.GetUserByEmail(c.Author.Email); err == nil {
		authorUsername = author.Name
	} else if !user_model.IsErrUserNotExist(err) {
		log.Error("GetUserByEmail: %v", err)
	}

	committerUsername := ""
	if committer, err := user_model.GetUserByEmail(c.Committer.Email); err == nil {
		committerUsername = committer.Name
	} else if !user_model.IsErrUserNotExist(err) {
		log.Error("GetUserByEmail: %v", err)
	}

	return &api.PayloadCommit{
		ID:      c.ID.String(),
		Message: c.Message(),
		URL:     util.URLJoin(repo.HTMLURL(), "commit", c.ID.String()),
		Author: &api.PayloadUser{
			Name:     c.Author.Name,
			Email:    c.Author.Email,
			UserName: authorUsername,
		},
		Committer: &api.PayloadUser{
			Name:     c.Committer.Name,
			Email:    c.Committer.Email,
			UserName: committerUsername,
		},
		Timestamp:    c.Author.When,
		Verification: ToVerification(c),
	}
}

// ToCommit convert a git.Commit to api.Commit
func ToCommit(repo *repo_model.Repository, gitRepo *git.Repository, commit *git.Commit, userCache map[string]*user_model.User) (*api.Commit, error) {
	var apiAuthor, apiCommitter *api.User

	// Retrieve author and committer information

	var cacheAuthor *user_model.User
	var ok bool
	if userCache == nil {
		cacheAuthor = (*user_model.User)(nil)
		ok = false
	} else {
		cacheAuthor, ok = userCache[commit.Author.Email]
	}

	if ok {
		apiAuthor = ToUser(cacheAuthor, nil)
	} else {
		author, err := user_model.GetUserByEmail(commit.Author.Email)
		if err != nil && !user_model.IsErrUserNotExist(err) {
			return nil, err
		} else if err == nil {
			apiAuthor = ToUser(author, nil)
			if userCache != nil {
				userCache[commit.Author.Email] = author
			}
		}
	}

	var cacheCommitter *user_model.User
	if userCache == nil {
		cacheCommitter = (*user_model.User)(nil)
		ok = false
	} else {
		cacheCommitter, ok = userCache[commit.Committer.Email]
	}

	if ok {
		apiCommitter = ToUser(cacheCommitter, nil)
	} else {
		committer, err := user_model.GetUserByEmail(commit.Committer.Email)
		if err != nil && !user_model.IsErrUserNotExist(err) {
			return nil, err
		} else if err == nil {
			apiCommitter = ToUser(committer, nil)
			if userCache != nil {
				userCache[commit.Committer.Email] = committer
			}
		}
	}

	// Retrieve parent(s) of the commit
	apiParents := make([]*api.CommitMeta, commit.ParentCount())
	for i := 0; i < commit.ParentCount(); i++ {
		sha, _ := commit.ParentID(i)
		apiParents[i] = &api.CommitMeta{
			URL: repo.APIURL() + "/git/commits/" + url.PathEscape(sha.String()),
			SHA: sha.String(),
		}
	}

	// Retrieve files affected by the commit
	fileStatus, err := git.GetCommitFileStatus(gitRepo.Ctx, repo.RepoPath(), commit.ID.String())
	if err != nil {
		return nil, err
	}
	affectedFileList := make([]*api.CommitAffectedFiles, 0, len(fileStatus.Added)+len(fileStatus.Removed)+len(fileStatus.Modified))
	for _, files := range [][]string{fileStatus.Added, fileStatus.Removed, fileStatus.Modified} {
		for _, filename := range files {
			affectedFileList = append(affectedFileList, &api.CommitAffectedFiles{
				Filename: filename,
			})
		}
	}

	diff, err := gitdiff.GetDiff(gitRepo, &gitdiff.DiffOptions{
		AfterCommitID: commit.ID.String(),
	})
	if err != nil {
		return nil, err
	}

	return &api.Commit{
		CommitMeta: &api.CommitMeta{
			URL:     repo.APIURL() + "/git/commits/" + url.PathEscape(commit.ID.String()),
			SHA:     commit.ID.String(),
			Created: commit.Committer.When,
		},
		HTMLURL: repo.HTMLURL() + "/commit/" + url.PathEscape(commit.ID.String()),
		RepoCommit: &api.RepoCommit{
			URL: repo.APIURL() + "/git/commits/" + url.PathEscape(commit.ID.String()),
			Author: &api.CommitUser{
				Identity: api.Identity{
					Name:  commit.Author.Name,
					Email: commit.Author.Email,
				},
				Date: commit.Author.When.Format(time.RFC3339),
			},
			Committer: &api.CommitUser{
				Identity: api.Identity{
					Name:  commit.Committer.Name,
					Email: commit.Committer.Email,
				},
				Date: commit.Committer.When.Format(time.RFC3339),
			},
			Message: commit.Message(),
			Tree: &api.CommitMeta{
				URL:     repo.APIURL() + "/git/trees/" + url.PathEscape(commit.ID.String()),
				SHA:     commit.ID.String(),
				Created: commit.Committer.When,
			},
			Verification: ToVerification(commit),
		},
		Author:    apiAuthor,
		Committer: apiCommitter,
		Parents:   apiParents,
		Files:     affectedFileList,
		Stats: &api.CommitStats{
			Total:     diff.TotalAddition + diff.TotalDeletion,
			Additions: diff.TotalAddition,
			Deletions: diff.TotalDeletion,
		},
	}, nil
}
