// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package issue

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gitbundle/modules/references"
	"github.com/gitbundle/server/pkg/action"
	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	access_model "github.com/gitbundle/server/pkg/perm/access"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/repo/repository"
	user_model "github.com/gitbundle/server/pkg/user"
)

const (
	secondsByMinute = float64(time.Minute / time.Second) // seconds in a minute
	secondsByHour   = 60 * secondsByMinute               // seconds in an hour
	secondsByDay    = 8 * secondsByHour                  // seconds in a day
	secondsByWeek   = 5 * secondsByDay                   // seconds in a week
	secondsByMonth  = 4 * secondsByWeek                  // seconds in a month
)

var reDuration = regexp.MustCompile(`(?i)^(?:(\d+([\.,]\d+)?)(?:mo))?(?:(\d+([\.,]\d+)?)(?:w))?(?:(\d+([\.,]\d+)?)(?:d))?(?:(\d+([\.,]\d+)?)(?:h))?(?:(\d+([\.,]\d+)?)(?:m))?$`)

// timeLogToAmount parses time log string and returns amount in seconds
func timeLogToAmount(str string) int64 {
	matches := reDuration.FindAllStringSubmatch(str, -1)
	if len(matches) == 0 {
		return 0
	}

	match := matches[0]

	var a int64

	// months
	if len(match[1]) > 0 {
		mo, _ := strconv.ParseFloat(strings.Replace(match[1], ",", ".", 1), 64)
		a += int64(mo * secondsByMonth)
	}

	// weeks
	if len(match[3]) > 0 {
		w, _ := strconv.ParseFloat(strings.Replace(match[3], ",", ".", 1), 64)
		a += int64(w * secondsByWeek)
	}

	// days
	if len(match[5]) > 0 {
		d, _ := strconv.ParseFloat(strings.Replace(match[5], ",", ".", 1), 64)
		a += int64(d * secondsByDay)
	}

	// hours
	if len(match[7]) > 0 {
		h, _ := strconv.ParseFloat(strings.Replace(match[7], ",", ".", 1), 64)
		a += int64(h * secondsByHour)
	}

	// minutes
	if len(match[9]) > 0 {
		d, _ := strconv.ParseFloat(strings.Replace(match[9], ",", ".", 1), 64)
		a += int64(d * secondsByMinute)
	}

	return a
}

func issueAddTime(issue *issues_model.Issue, doer *user_model.User, time time.Time, timeLog string) error {
	amount := timeLogToAmount(timeLog)
	if amount == 0 {
		return nil
	}

	_, err := issues_model.AddTime(doer, issue, amount, time)
	return err
}

// getIssueFromRef returns the issue referenced by a ref. Returns a nil *Issue
// if the provided ref references a non-existent issue.
func getIssueFromRef(repo *repo_model.Repository, index int64) (*issues_model.Issue, error) {
	issue, err := issues_model.GetIssueByIndex(repo.ID, index)
	if err != nil {
		if issues_model.IsErrIssueNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return issue, nil
}

// UpdateIssuesCommit checks if issues are manipulated by commit message.
func UpdateIssuesCommit(doer *user_model.User, repo *repo_model.Repository, commits []*repository.PushCommit, branchName string) error {
	// Commits are appended in the reverse order.
	for i := len(commits) - 1; i >= 0; i-- {
		c := commits[i]

		type markKey struct {
			ID     int64
			Action references.XRefAction
		}

		refMarked := make(map[markKey]bool)
		var refRepo *repo_model.Repository
		var refIssue *issues_model.Issue
		var err error
		for _, ref := range references.FindAllIssueReferences(c.Message) {

			// issue is from another repo
			if len(ref.Owner) > 0 && len(ref.Name) > 0 {
				refRepo, err = action.GetRepositoryFromMatch(ref.Owner, ref.Name)
				if err != nil {
					continue
				}
			} else {
				refRepo = repo
			}
			if refIssue, err = getIssueFromRef(refRepo, ref.Index); err != nil {
				return err
			}
			if refIssue == nil {
				continue
			}

			perm, err := access_model.GetUserRepoPermission(db.DefaultContext, refRepo, doer)
			if err != nil {
				return err
			}

			key := markKey{ID: refIssue.ID, Action: ref.Action}
			if refMarked[key] {
				continue
			}
			refMarked[key] = true

			// FIXME: this kind of condition is all over the code, it should be consolidated in a single place
			canclose := perm.IsAdmin() || perm.IsOwner() || perm.CanWriteIssuesOrPulls(refIssue.IsPull) || refIssue.PosterID == doer.ID
			cancomment := canclose || perm.CanReadIssuesOrPulls(refIssue.IsPull)

			// Don't proceed if the user can't comment
			if !cancomment {
				continue
			}

			message := fmt.Sprintf(`<a href="%s/commit/%s">%s</a>`, html.EscapeString(repo.Link()), html.EscapeString(url.PathEscape(c.Sha1)), html.EscapeString(strings.SplitN(c.Message, "\n", 2)[0]))
			if err = issues_model.CreateRefComment(doer, refRepo, refIssue, message, c.Sha1); err != nil {
				return err
			}

			// Only issues can be closed/reopened this way, and user needs the correct permissions
			if refIssue.IsPull || !canclose {
				continue
			}

			// Only process closing/reopening keywords
			if ref.Action != references.XRefActionCloses && ref.Action != references.XRefActionReopens {
				continue
			}

			if !repo.CloseIssuesViaCommitInAnyBranch {
				// If the issue was specified to be in a particular branch, don't allow commits in other branches to close it
				if refIssue.Ref != "" {
					if branchName != refIssue.Ref {
						continue
					}
					// Otherwise, only process commits to the default branch
				} else if branchName != repo.DefaultBranch {
					continue
				}
			}
			close := ref.Action == references.XRefActionCloses
			if close && len(ref.TimeLog) > 0 {
				if err := issueAddTime(refIssue, doer, c.Timestamp, ref.TimeLog); err != nil {
					return err
				}
			}
			if close != refIssue.IsClosed {
				refIssue.Repo = refRepo
				if err := ChangeStatus(refIssue, doer, close); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
