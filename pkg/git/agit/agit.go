// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package agit

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gitbundle/modules/git"
	"github.com/gitbundle/modules/log"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/notification"
	pull_service "github.com/gitbundle/server/pkg/pull/manager"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/repo/private"
	user_model "github.com/gitbundle/server/pkg/user"
)

// ProcReceive handle proc receive work
func ProcReceive(ctx context.Context, repo *repo_model.Repository, gitRepo *git.Repository, opts *private.HookOptions) ([]private.HookProcReceiveRefResult, error) {
	// TODO: Add more options?
	var (
		topicBranch string
		title       string
		description string
		forcePush   bool
	)

	results := make([]private.HookProcReceiveRefResult, 0, len(opts.OldCommitIDs))

	ownerName := repo.OwnerName
	repoName := repo.Name

	topicBranch = opts.GitPushOptions["topic"]
	_, forcePush = opts.GitPushOptions["force-push"]

	for i := range opts.OldCommitIDs {
		if opts.NewCommitIDs[i] == git.EmptySHA {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "Can't delete not exist branch",
			})
			continue
		}

		if !strings.HasPrefix(opts.RefFullNames[i], git.PullRequestPrefix) {
			results = append(results, private.HookProcReceiveRefResult{
				IsNotMatched: true,
				OriginalRef:  opts.RefFullNames[i],
			})
			continue
		}

		baseBranchName := opts.RefFullNames[i][len(git.PullRequestPrefix):]
		curentTopicBranch := ""
		if !gitRepo.IsBranchExist(baseBranchName) {
			// try match refs/for/<target-branch>/<topic-branch>
			for p, v := range baseBranchName {
				if v == '/' && gitRepo.IsBranchExist(baseBranchName[:p]) && p != len(baseBranchName)-1 {
					curentTopicBranch = baseBranchName[p+1:]
					baseBranchName = baseBranchName[:p]
					break
				}
			}
		}

		if len(topicBranch) == 0 && len(curentTopicBranch) == 0 {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "topic-branch is not set",
			})
			continue
		}

		headBranch := ""
		userName := strings.ToLower(opts.UserName)

		if len(curentTopicBranch) == 0 {
			curentTopicBranch = topicBranch
		}

		// because different user maybe want to use same topic,
		// So it's better to make sure the topic branch name
		// has user name prefix
		if !strings.HasPrefix(curentTopicBranch, userName+"/") {
			headBranch = userName + "/" + curentTopicBranch
		} else {
			headBranch = curentTopicBranch
		}

		pr, err := issues_model.GetUnmergedPullRequest(repo.ID, repo.ID, headBranch, baseBranchName, issues_model.PullRequestFlowAGit)
		if err != nil {
			if !issues_model.IsErrPullRequestNotExist(err) {
				return nil, fmt.Errorf("Failed to get unmerged agit flow pull request in repository: %s/%s Error: %v", ownerName, repoName, err)
			}

			// create a new pull request
			if len(title) == 0 {
				has := false
				title, has = opts.GitPushOptions["title"]
				if !has || len(title) == 0 {
					commit, err := gitRepo.GetCommit(opts.NewCommitIDs[i])
					if err != nil {
						return nil, fmt.Errorf("Failed to get commit %s in repository: %s/%s Error: %v", opts.NewCommitIDs[i], ownerName, repoName, err)
					}
					title = strings.Split(commit.CommitMessage, "\n")[0]
				}
				description = opts.GitPushOptions["description"]
			}

			pusher, err := user_model.GetUserByID(opts.UserID)
			if err != nil {
				return nil, fmt.Errorf("Failed to get user. Error: %v", err)
			}

			prIssue := &issues_model.Issue{
				RepoID:   repo.ID,
				Title:    title,
				PosterID: pusher.ID,
				Poster:   pusher,
				IsPull:   true,
				Content:  description,
			}

			pr := &issues_model.PullRequest{
				HeadRepoID:   repo.ID,
				BaseRepoID:   repo.ID,
				HeadBranch:   headBranch,
				HeadCommitID: opts.NewCommitIDs[i],
				BaseBranch:   baseBranchName,
				HeadRepo:     repo,
				BaseRepo:     repo,
				MergeBase:    "",
				Type:         issues_model.PullRequestGitea,
				Flow:         issues_model.PullRequestFlowAGit,
			}

			if err := pull_service.NewPullRequest(ctx, repo, prIssue, []int64{}, []string{}, pr, []int64{}); err != nil {
				return nil, err
			}

			log.Trace("Pull request created: %d/%d", repo.ID, prIssue.ID)

			results = append(results, private.HookProcReceiveRefResult{
				Ref:         pr.GetGitRefName(),
				OriginalRef: opts.RefFullNames[i],
				OldOID:      git.EmptySHA,
				NewOID:      opts.NewCommitIDs[i],
			})
			continue
		}

		// update exist pull request
		if err := pr.LoadBaseRepoCtx(ctx); err != nil {
			return nil, fmt.Errorf("Unable to load base repository for PR[%d] Error: %v", pr.ID, err)
		}

		oldCommitID, err := gitRepo.GetRefCommitID(pr.GetGitRefName())
		if err != nil {
			return nil, fmt.Errorf("Unable to get ref commit id in base repository for PR[%d] Error: %v", pr.ID, err)
		}

		if oldCommitID == opts.NewCommitIDs[i] {
			results = append(results, private.HookProcReceiveRefResult{
				OriginalRef: opts.RefFullNames[i],
				OldOID:      opts.OldCommitIDs[i],
				NewOID:      opts.NewCommitIDs[i],
				Err:         "new commit is same with old commit",
			})
			continue
		}

		if !forcePush {
			output, _, err := git.NewCommand(ctx, "rev-list", "--max-count=1", oldCommitID, "^"+opts.NewCommitIDs[i]).RunStdString(&git.RunOpts{Dir: repo.RepoPath(), Env: os.Environ()})
			if err != nil {
				return nil, fmt.Errorf("Fail to detect force push: %v", err)
			} else if len(output) > 0 {
				results = append(results, private.HookProcReceiveRefResult{
					OriginalRef: opts.RefFullNames[i],
					OldOID:      opts.OldCommitIDs[i],
					NewOID:      opts.NewCommitIDs[i],
					Err:         "request `force-push` push option",
				})
				continue
			}
		}

		pr.HeadCommitID = opts.NewCommitIDs[i]
		if err = pull_service.UpdateRef(ctx, pr); err != nil {
			return nil, fmt.Errorf("Failed to update pull ref. Error: %v", err)
		}

		pull_service.AddToTaskQueue(pr)
		pusher, err := user_model.GetUserByID(opts.UserID)
		if err != nil {
			return nil, fmt.Errorf("Failed to get user. Error: %v", err)
		}
		err = pr.LoadIssue()
		if err != nil {
			return nil, fmt.Errorf("Failed to load pull issue. Error: %v", err)
		}
		comment, err := issues_model.CreatePushPullComment(ctx, pusher, pr, oldCommitID, opts.NewCommitIDs[i])
		if err == nil && comment != nil {
			notification.NotifyPullRequestPushCommits(pusher, pr, comment)
		}
		notification.NotifyPullRequestSynchronized(pusher, pr)
		isForcePush := comment != nil && comment.IsForcePush

		results = append(results, private.HookProcReceiveRefResult{
			OldOID:      oldCommitID,
			NewOID:      opts.NewCommitIDs[i],
			Ref:         pr.GetGitRefName(),
			OriginalRef: opts.RefFullNames[i],
			IsForcePush: isForcePush,
		})
	}

	return results, nil
}

// UserNameChanged handle user name change for agit flow pull
func UserNameChanged(user *user_model.User, newName string) error {
	pulls, err := issues_model.GetAllUnmergedAgitPullRequestByPoster(user.ID)
	if err != nil {
		return err
	}

	newName = strings.ToLower(newName)

	for _, pull := range pulls {
		pull.HeadBranch = strings.TrimPrefix(pull.HeadBranch, user.LowerName+"/")
		pull.HeadBranch = newName + "/" + pull.HeadBranch
		if err = pull.UpdateCols("head_branch"); err != nil {
			return err
		}
	}

	return nil
}
