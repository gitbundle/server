// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package files

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/gitbundle/modules/git"
	"github.com/gitbundle/modules/lfs"
	"github.com/gitbundle/modules/setting"
	git_model "github.com/gitbundle/server/pkg/git"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/upload"
	user_model "github.com/gitbundle/server/pkg/user"
)

// UploadRepoFileOptions contains the uploaded repository file options
type UploadRepoFileOptions struct {
	LastCommitID string
	OldBranch    string
	NewBranch    string
	TreePath     string
	Message      string
	Files        []string // In UUID format.
	Signoff      bool
}

type uploadInfo struct {
	upload        *upload.Upload
	lfsMetaObject *git_model.LFSMetaObject
}

func cleanUpAfterFailure(infos *[]uploadInfo, t *TemporaryUploadRepository, original error) error {
	for _, info := range *infos {
		if info.lfsMetaObject == nil {
			continue
		}
		if !info.lfsMetaObject.Existing {
			if _, err := git_model.RemoveLFSMetaObjectByOid(t.repo.ID, info.lfsMetaObject.Oid); err != nil {
				original = fmt.Errorf("%v, %v", original, err)
			}
		}
	}
	return original
}

// UploadRepoFiles uploads files to the given repository
func UploadRepoFiles(ctx context.Context, repo *repo_model.Repository, doer *user_model.User, opts *UploadRepoFileOptions) error {
	if len(opts.Files) == 0 {
		return nil
	}

	uploads, err := upload.GetUploadsByUUIDs(opts.Files)
	if err != nil {
		return fmt.Errorf("GetUploadsByUUIDs [uuids: %v]: %v", opts.Files, err)
	}

	names := make([]string, len(uploads))
	infos := make([]uploadInfo, len(uploads))
	for i, upload := range uploads {
		// Check file is not lfs locked, will return nil if lock setting not enabled
		filepath := path.Join(opts.TreePath, upload.Name)
		lfsLock, err := git_model.GetTreePathLock(repo.ID, filepath)
		if err != nil {
			return err
		}
		if lfsLock != nil && lfsLock.OwnerID != doer.ID {
			u, err := user_model.GetUserByID(lfsLock.OwnerID)
			if err != nil {
				return err
			}
			return git_model.ErrLFSFileLocked{RepoID: repo.ID, Path: filepath, UserName: u.Name}
		}

		names[i] = upload.Name
		infos[i] = uploadInfo{upload: upload}
	}

	t, err := NewTemporaryUploadRepository(ctx, repo)
	if err != nil {
		return err
	}
	defer t.Close()
	if err := t.Clone(opts.OldBranch); err != nil {
		return err
	}
	if err := t.SetDefaultIndex(); err != nil {
		return err
	}

	var filename2attribute2info map[string]map[string]string
	if setting.LFS.StartServer {
		filename2attribute2info, err = t.gitRepo.CheckAttribute(git.CheckAttributeOpts{
			Attributes: []string{"filter"},
			Filenames:  names,
			CachedOnly: true,
		})
		if err != nil {
			return err
		}
	}

	// Copy uploaded files into repository.
	for i := range infos {
		if err := copyUploadedLFSFileIntoRepository(&infos[i], filename2attribute2info, t, opts.TreePath); err != nil {
			return err
		}
	}

	// Now write the tree
	treeHash, err := t.WriteTree()
	if err != nil {
		return err
	}

	// make author and committer the doer
	author := doer
	committer := doer

	// Now commit the tree
	commitHash, err := t.CommitTree(opts.LastCommitID, author, committer, treeHash, opts.Message, opts.Signoff)
	if err != nil {
		return err
	}

	// Now deal with LFS objects
	for i := range infos {
		if infos[i].lfsMetaObject == nil {
			continue
		}
		infos[i].lfsMetaObject, err = git_model.NewLFSMetaObject(infos[i].lfsMetaObject)
		if err != nil {
			// OK Now we need to cleanup
			return cleanUpAfterFailure(&infos, t, err)
		}
		// Don't move the files yet - we need to ensure that
		// everything can be inserted first
	}

	// OK now we can insert the data into the store - there's no way to clean up the store
	// once it's in there, it's in there.
	contentStore := lfs.NewContentStore()
	for _, info := range infos {
		if err := uploadToLFSContentStore(info, contentStore); err != nil {
			return cleanUpAfterFailure(&infos, t, err)
		}
	}

	// Then push this tree to NewBranch
	if err := t.Push(doer, commitHash, opts.NewBranch); err != nil {
		return err
	}

	return upload.DeleteUploads(uploads...)
}

func copyUploadedLFSFileIntoRepository(info *uploadInfo, filename2attribute2info map[string]map[string]string, t *TemporaryUploadRepository, treePath string) error {
	file, err := os.Open(info.upload.LocalPath())
	if err != nil {
		return err
	}
	defer file.Close()

	var objectHash string
	if setting.LFS.StartServer && filename2attribute2info[info.upload.Name] != nil && filename2attribute2info[info.upload.Name]["filter"] == "lfs" {
		// Handle LFS
		// FIXME: Inefficient! this should probably happen in upload.Upload
		pointer, err := lfs.GeneratePointer(file)
		if err != nil {
			return err
		}

		info.lfsMetaObject = &git_model.LFSMetaObject{Pointer: pointer, RepositoryID: t.repo.ID}

		if objectHash, err = t.HashObject(strings.NewReader(pointer.StringContent())); err != nil {
			return err
		}
	} else if objectHash, err = t.HashObject(file); err != nil {
		return err
	}

	// Add the object to the index
	return t.AddObjectToIndex("100644", objectHash, path.Join(treePath, info.upload.Name))
}

func uploadToLFSContentStore(info uploadInfo, contentStore *lfs.ContentStore) error {
	if info.lfsMetaObject == nil {
		return nil
	}
	exist, err := contentStore.Exists(info.lfsMetaObject.Pointer)
	if err != nil {
		return err
	}
	if !exist {
		file, err := os.Open(info.upload.LocalPath())
		if err != nil {
			return err
		}

		defer file.Close()
		// FIXME: Put regenerates the hash and copies the file over.
		// I guess this strictly ensures the soundness of the store but this is inefficient.
		if err := contentStore.Put(info.lfsMetaObject.Pointer, file); err != nil {
			// OK Now we need to cleanup
			// Can't clean up the store, once uploaded there they're there.
			return err
		}
	}
	return nil
}
