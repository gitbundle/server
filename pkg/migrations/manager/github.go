// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package migrations

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gitbundle/api/pkg/structs"
	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/proxy"
	"github.com/gitbundle/modules/util"
	"github.com/gitbundle/server/pkg/migrations/manager/schemas"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

var (
	_ schemas.Downloader        = &GithubDownloaderV3{}
	_ schemas.DownloaderFactory = &GithubDownloaderV3Factory{}
	// GithubLimitRateRemaining limit to wait for new rate to apply
	GithubLimitRateRemaining = 0
)

func init() {
	RegisterDownloaderFactory(&GithubDownloaderV3Factory{})
}

// GithubDownloaderV3Factory defines a github downloader v3 factory
type GithubDownloaderV3Factory struct{}

// New returns a Downloader related to this factory according MigrateOptions
func (f *GithubDownloaderV3Factory) New(ctx context.Context, opts schemas.MigrateOptions) (schemas.Downloader, error) {
	u, err := url.Parse(opts.CloneAddr)
	if err != nil {
		return nil, err
	}

	baseURL := u.Scheme + "://" + u.Host
	fields := strings.Split(u.Path, "/")
	oldOwner := fields[1]
	oldName := strings.TrimSuffix(fields[2], ".git")

	log.Trace("Create github downloader: %s/%s", oldOwner, oldName)

	return NewGithubDownloaderV3(ctx, baseURL, opts.AuthUsername, opts.AuthPassword, opts.AuthToken, oldOwner, oldName), nil
}

// GitServiceType returns the type of git service
func (f *GithubDownloaderV3Factory) GitServiceType() structs.GitServiceType {
	return structs.GithubService
}

// GithubDownloaderV3 implements a Downloader interface to get repository information
// from github via APIv3
type GithubDownloaderV3 struct {
	schemas.NullDownloader
	ctx           context.Context
	clients       []*github.Client
	repoOwner     string
	repoName      string
	userName      string
	password      string
	rates         []*github.Rate
	curClientIdx  int
	maxPerPage    int
	SkipReactions bool
}

// NewGithubDownloaderV3 creates a github Downloader via github v3 API
func NewGithubDownloaderV3(ctx context.Context, baseURL, userName, password, token, repoOwner, repoName string) *GithubDownloaderV3 {
	downloader := GithubDownloaderV3{
		userName:   userName,
		password:   password,
		ctx:        ctx,
		repoOwner:  repoOwner,
		repoName:   repoName,
		maxPerPage: 100,
	}

	if token != "" {
		tokens := strings.Split(token, ",")
		for _, token := range tokens {
			token = strings.TrimSpace(token)
			ts := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: token},
			)
			client := &http.Client{
				Transport: &oauth2.Transport{
					Base:   NewMigrationHTTPTransport(),
					Source: oauth2.ReuseTokenSource(nil, ts),
				},
			}

			downloader.addClient(client, baseURL)
		}
	} else {
		transport := NewMigrationHTTPTransport()
		transport.Proxy = func(req *http.Request) (*url.URL, error) {
			req.SetBasicAuth(userName, password)
			return proxy.Proxy()(req)
		}
		client := &http.Client{
			Transport: transport,
		}
		downloader.addClient(client, baseURL)
	}
	return &downloader
}

func (g *GithubDownloaderV3) addClient(client *http.Client, baseURL string) {
	githubClient := github.NewClient(client)
	if baseURL != "https://github.com" {
		githubClient, _ = github.NewEnterpriseClient(baseURL, baseURL, client)
	}
	g.clients = append(g.clients, githubClient)
	g.rates = append(g.rates, nil)
}

// SetContext set context
func (g *GithubDownloaderV3) SetContext(ctx context.Context) {
	g.ctx = ctx
}

func (g *GithubDownloaderV3) waitAndPickClient() {
	var recentIdx int
	var maxRemaining int
	for i := 0; i < len(g.clients); i++ {
		if g.rates[i] != nil && g.rates[i].Remaining > maxRemaining {
			maxRemaining = g.rates[i].Remaining
			recentIdx = i
		}
	}
	g.curClientIdx = recentIdx // if no max remain, it will always pick the first client.

	for g.rates[g.curClientIdx] != nil && g.rates[g.curClientIdx].Remaining <= GithubLimitRateRemaining {
		timer := time.NewTimer(time.Until(g.rates[g.curClientIdx].Reset.Time))
		select {
		case <-g.ctx.Done():
			util.StopTimer(timer)
			return
		case <-timer.C:
		}

		err := g.RefreshRate()
		if err != nil {
			log.Error("g.getClient().RateLimits: %s", err)
		}
	}
}

// RefreshRate update the current rate (doesn't count in rate limit)
func (g *GithubDownloaderV3) RefreshRate() error {
	rates, _, err := g.getClient().RateLimits(g.ctx)
	if err != nil {
		// if rate limit is not enabled, ignore it
		if strings.Contains(err.Error(), "404") {
			g.setRate(nil)
			return nil
		}
		return err
	}

	g.setRate(rates.GetCore())
	return nil
}

func (g *GithubDownloaderV3) getClient() *github.Client {
	return g.clients[g.curClientIdx]
}

func (g *GithubDownloaderV3) setRate(rate *github.Rate) {
	g.rates[g.curClientIdx] = rate
}

// GetRepoInfo returns a repository information
func (g *GithubDownloaderV3) GetRepoInfo() (*schemas.Repository, error) {
	g.waitAndPickClient()
	gr, resp, err := g.getClient().Repositories.Get(g.ctx, g.repoOwner, g.repoName)
	if err != nil {
		return nil, err
	}
	g.setRate(&resp.Rate)

	// convert github repo to stand Repo
	return &schemas.Repository{
		Owner:         g.repoOwner,
		Name:          gr.GetName(),
		IsPrivate:     gr.GetPrivate(),
		Description:   gr.GetDescription(),
		OriginalURL:   gr.GetHTMLURL(),
		CloneURL:      gr.GetCloneURL(),
		DefaultBranch: gr.GetDefaultBranch(),
	}, nil
}

// GetTopics return github topics
func (g *GithubDownloaderV3) GetTopics() ([]string, error) {
	g.waitAndPickClient()
	r, resp, err := g.getClient().Repositories.Get(g.ctx, g.repoOwner, g.repoName)
	if err != nil {
		return nil, err
	}
	g.setRate(&resp.Rate)
	return r.Topics, nil
}

// GetMilestones returns milestones
func (g *GithubDownloaderV3) GetMilestones() ([]*schemas.Milestone, error) {
	perPage := g.maxPerPage
	milestones := make([]*schemas.Milestone, 0, perPage)
	for i := 1; ; i++ {
		g.waitAndPickClient()
		ms, resp, err := g.getClient().Issues.ListMilestones(g.ctx, g.repoOwner, g.repoName,
			&github.MilestoneListOptions{
				State: "all",
				ListOptions: github.ListOptions{
					Page:    i,
					PerPage: perPage,
				},
			})
		if err != nil {
			return nil, err
		}
		g.setRate(&resp.Rate)

		for _, m := range ms {
			state := "open"
			if m.State != nil {
				state = *m.State
			}
			milestones = append(milestones, &schemas.Milestone{
				Title:       m.GetTitle(),
				Description: m.GetDescription(),
				Deadline:    m.DueOn,
				State:       state,
				Created:     m.GetCreatedAt(),
				Updated:     m.UpdatedAt,
				Closed:      m.ClosedAt,
			})
		}
		if len(ms) < perPage {
			break
		}
	}
	return milestones, nil
}

func convertGithubLabel(label *github.Label) *schemas.Label {
	return &schemas.Label{
		Name:        label.GetName(),
		Color:       label.GetColor(),
		Description: label.GetDescription(),
	}
}

// GetLabels returns labels
func (g *GithubDownloaderV3) GetLabels() ([]*schemas.Label, error) {
	perPage := g.maxPerPage
	labels := make([]*schemas.Label, 0, perPage)
	for i := 1; ; i++ {
		g.waitAndPickClient()
		ls, resp, err := g.getClient().Issues.ListLabels(g.ctx, g.repoOwner, g.repoName,
			&github.ListOptions{
				Page:    i,
				PerPage: perPage,
			})
		if err != nil {
			return nil, err
		}
		g.setRate(&resp.Rate)

		for _, label := range ls {
			labels = append(labels, convertGithubLabel(label))
		}
		if len(ls) < perPage {
			break
		}
	}
	return labels, nil
}

func (g *GithubDownloaderV3) convertGithubRelease(rel *github.RepositoryRelease) *schemas.Release {
	r := &schemas.Release{
		Name:            rel.GetName(),
		TagName:         rel.GetTagName(),
		TargetCommitish: rel.GetTargetCommitish(),
		Draft:           rel.GetDraft(),
		Prerelease:      rel.GetPrerelease(),
		Created:         rel.GetCreatedAt().Time,
		PublisherID:     rel.GetAuthor().GetID(),
		PublisherName:   rel.GetAuthor().GetLogin(),
		PublisherEmail:  rel.GetAuthor().GetEmail(),
		Body:            rel.GetBody(),
	}

	if rel.PublishedAt != nil {
		r.Published = rel.PublishedAt.Time
	}

	httpClient := NewMigrationHTTPClient()

	for _, asset := range rel.Assets {
		assetID := *asset.ID // Don't optimize this, for closure we need a local variable
		r.Assets = append(r.Assets, &schemas.ReleaseAsset{
			ID:            asset.GetID(),
			Name:          asset.GetName(),
			ContentType:   asset.ContentType,
			Size:          asset.Size,
			DownloadCount: asset.DownloadCount,
			Created:       asset.CreatedAt.Time,
			Updated:       asset.UpdatedAt.Time,
			DownloadFunc: func() (io.ReadCloser, error) {
				g.waitAndPickClient()
				asset, redirectURL, err := g.getClient().Repositories.DownloadReleaseAsset(g.ctx, g.repoOwner, g.repoName, assetID, nil)
				if err != nil {
					return nil, err
				}
				if err := g.RefreshRate(); err != nil {
					log.Error("g.getClient().RateLimits: %s", err)
				}
				if asset == nil {
					if redirectURL != "" {
						g.waitAndPickClient()
						req, err := http.NewRequestWithContext(g.ctx, "GET", redirectURL, nil)
						if err != nil {
							return nil, err
						}
						resp, err := httpClient.Do(req)
						err1 := g.RefreshRate()
						if err1 != nil {
							log.Error("g.getClient().RateLimits: %s", err1)
						}
						if err != nil {
							return nil, err
						}
						return resp.Body, nil
					}
					return nil, fmt.Errorf("No release asset found for %d", assetID)
				}
				return asset, nil
			},
		})
	}
	return r
}

// GetReleases returns releases
func (g *GithubDownloaderV3) GetReleases() ([]*schemas.Release, error) {
	perPage := g.maxPerPage
	releases := make([]*schemas.Release, 0, perPage)
	for i := 1; ; i++ {
		g.waitAndPickClient()
		ls, resp, err := g.getClient().Repositories.ListReleases(g.ctx, g.repoOwner, g.repoName,
			&github.ListOptions{
				Page:    i,
				PerPage: perPage,
			})
		if err != nil {
			return nil, err
		}
		g.setRate(&resp.Rate)

		for _, release := range ls {
			releases = append(releases, g.convertGithubRelease(release))
		}
		if len(ls) < perPage {
			break
		}
	}
	return releases, nil
}

// GetIssues returns issues according start and limit
func (g *GithubDownloaderV3) GetIssues(page, perPage int) ([]*schemas.Issue, bool, error) {
	if perPage > g.maxPerPage {
		perPage = g.maxPerPage
	}
	opt := &github.IssueListByRepoOptions{
		Sort:      "created",
		Direction: "asc",
		State:     "all",
		ListOptions: github.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
	}

	allIssues := make([]*schemas.Issue, 0, perPage)
	g.waitAndPickClient()
	issues, resp, err := g.getClient().Issues.ListByRepo(g.ctx, g.repoOwner, g.repoName, opt)
	if err != nil {
		return nil, false, fmt.Errorf("error while listing repos: %v", err)
	}
	log.Trace("Request get issues %d/%d, but in fact get %d", perPage, page, len(issues))
	g.setRate(&resp.Rate)
	for _, issue := range issues {
		if issue.IsPullRequest() {
			continue
		}

		labels := make([]*schemas.Label, 0, len(issue.Labels))
		for _, l := range issue.Labels {
			labels = append(labels, convertGithubLabel(l))
		}

		// get reactions
		var reactions []*schemas.Reaction
		if !g.SkipReactions {
			for i := 1; ; i++ {
				g.waitAndPickClient()
				res, resp, err := g.getClient().Reactions.ListIssueReactions(g.ctx, g.repoOwner, g.repoName, issue.GetNumber(), &github.ListOptions{
					Page:    i,
					PerPage: perPage,
				})
				if err != nil {
					return nil, false, err
				}
				g.setRate(&resp.Rate)
				if len(res) == 0 {
					break
				}
				for _, reaction := range res {
					reactions = append(reactions, &schemas.Reaction{
						UserID:   reaction.User.GetID(),
						UserName: reaction.User.GetLogin(),
						Content:  reaction.GetContent(),
					})
				}
			}
		}

		var assignees []string
		for i := range issue.Assignees {
			assignees = append(assignees, issue.Assignees[i].GetLogin())
		}

		allIssues = append(allIssues, &schemas.Issue{
			Title:        *issue.Title,
			Number:       int64(*issue.Number),
			PosterID:     issue.GetUser().GetID(),
			PosterName:   issue.GetUser().GetLogin(),
			PosterEmail:  issue.GetUser().GetEmail(),
			Content:      issue.GetBody(),
			Milestone:    issue.GetMilestone().GetTitle(),
			State:        issue.GetState(),
			Created:      issue.GetCreatedAt(),
			Updated:      issue.GetUpdatedAt(),
			Labels:       labels,
			Reactions:    reactions,
			Closed:       issue.ClosedAt,
			IsLocked:     issue.GetLocked(),
			Assignees:    assignees,
			ForeignIndex: int64(*issue.Number),
		})
	}

	return allIssues, len(issues) < perPage, nil
}

// SupportGetRepoComments return true if it supports get repo comments
func (g *GithubDownloaderV3) SupportGetRepoComments() bool {
	return true
}

// GetComments returns comments according issueNumber
func (g *GithubDownloaderV3) GetComments(commentable schemas.Commentable) ([]*schemas.Comment, bool, error) {
	comments, err := g.getComments(commentable)
	return comments, false, err
}

func (g *GithubDownloaderV3) getComments(commentable schemas.Commentable) ([]*schemas.Comment, error) {
	var (
		allComments = make([]*schemas.Comment, 0, g.maxPerPage)
		created     = "created"
		asc         = "asc"
	)
	opt := &github.IssueListCommentsOptions{
		Sort:      &created,
		Direction: &asc,
		ListOptions: github.ListOptions{
			PerPage: g.maxPerPage,
		},
	}
	for {
		g.waitAndPickClient()
		comments, resp, err := g.getClient().Issues.ListComments(g.ctx, g.repoOwner, g.repoName, int(commentable.GetForeignIndex()), opt)
		if err != nil {
			return nil, fmt.Errorf("error while listing repos: %v", err)
		}
		g.setRate(&resp.Rate)
		for _, comment := range comments {
			// get reactions
			var reactions []*schemas.Reaction
			if !g.SkipReactions {
				for i := 1; ; i++ {
					g.waitAndPickClient()
					res, resp, err := g.getClient().Reactions.ListIssueCommentReactions(g.ctx, g.repoOwner, g.repoName, comment.GetID(), &github.ListOptions{
						Page:    i,
						PerPage: g.maxPerPage,
					})
					if err != nil {
						return nil, err
					}
					g.setRate(&resp.Rate)
					if len(res) == 0 {
						break
					}
					for _, reaction := range res {
						reactions = append(reactions, &schemas.Reaction{
							UserID:   reaction.User.GetID(),
							UserName: reaction.User.GetLogin(),
							Content:  reaction.GetContent(),
						})
					}
				}
			}

			allComments = append(allComments, &schemas.Comment{
				IssueIndex:  commentable.GetLocalIndex(),
				Index:       comment.GetID(),
				PosterID:    comment.GetUser().GetID(),
				PosterName:  comment.GetUser().GetLogin(),
				PosterEmail: comment.GetUser().GetEmail(),
				Content:     comment.GetBody(),
				Created:     comment.GetCreatedAt(),
				Updated:     comment.GetUpdatedAt(),
				Reactions:   reactions,
			})
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allComments, nil
}

// GetAllComments returns repository comments according page and perPageSize
func (g *GithubDownloaderV3) GetAllComments(page, perPage int) ([]*schemas.Comment, bool, error) {
	var (
		allComments = make([]*schemas.Comment, 0, perPage)
		created     = "created"
		asc         = "asc"
	)
	if perPage > g.maxPerPage {
		perPage = g.maxPerPage
	}
	opt := &github.IssueListCommentsOptions{
		Sort:      &created,
		Direction: &asc,
		ListOptions: github.ListOptions{
			Page:    page,
			PerPage: perPage,
		},
	}

	g.waitAndPickClient()
	comments, resp, err := g.getClient().Issues.ListComments(g.ctx, g.repoOwner, g.repoName, 0, opt)
	if err != nil {
		return nil, false, fmt.Errorf("error while listing repos: %v", err)
	}
	isEnd := resp.NextPage == 0

	log.Trace("Request get comments %d/%d, but in fact get %d, next page is %d", perPage, page, len(comments), resp.NextPage)
	g.setRate(&resp.Rate)
	for _, comment := range comments {
		// get reactions
		var reactions []*schemas.Reaction
		if !g.SkipReactions {
			for i := 1; ; i++ {
				g.waitAndPickClient()
				res, resp, err := g.getClient().Reactions.ListIssueCommentReactions(g.ctx, g.repoOwner, g.repoName, comment.GetID(), &github.ListOptions{
					Page:    i,
					PerPage: g.maxPerPage,
				})
				if err != nil {
					return nil, false, err
				}
				g.setRate(&resp.Rate)
				if len(res) == 0 {
					break
				}
				for _, reaction := range res {
					reactions = append(reactions, &schemas.Reaction{
						UserID:   reaction.User.GetID(),
						UserName: reaction.User.GetLogin(),
						Content:  reaction.GetContent(),
					})
				}
			}
		}
		idx := strings.LastIndex(*comment.IssueURL, "/")
		issueIndex, _ := strconv.ParseInt((*comment.IssueURL)[idx+1:], 10, 64)
		allComments = append(allComments, &schemas.Comment{
			IssueIndex:  issueIndex,
			Index:       comment.GetID(),
			PosterID:    comment.GetUser().GetID(),
			PosterName:  comment.GetUser().GetLogin(),
			PosterEmail: comment.GetUser().GetEmail(),
			Content:     comment.GetBody(),
			Created:     comment.GetCreatedAt(),
			Updated:     comment.GetUpdatedAt(),
			Reactions:   reactions,
		})
	}

	return allComments, isEnd, nil
}

// GetPullRequests returns pull requests according page and perPage
func (g *GithubDownloaderV3) GetPullRequests(page, perPage int) ([]*schemas.PullRequest, bool, error) {
	if perPage > g.maxPerPage {
		perPage = g.maxPerPage
	}
	opt := &github.PullRequestListOptions{
		Sort:      "created",
		Direction: "asc",
		State:     "all",
		ListOptions: github.ListOptions{
			PerPage: perPage,
			Page:    page,
		},
	}
	allPRs := make([]*schemas.PullRequest, 0, perPage)
	g.waitAndPickClient()
	prs, resp, err := g.getClient().PullRequests.List(g.ctx, g.repoOwner, g.repoName, opt)
	if err != nil {
		return nil, false, fmt.Errorf("error while listing repos: %v", err)
	}
	log.Trace("Request get pull requests %d/%d, but in fact get %d", perPage, page, len(prs))
	g.setRate(&resp.Rate)
	for _, pr := range prs {
		labels := make([]*schemas.Label, 0, len(pr.Labels))
		for _, l := range pr.Labels {
			labels = append(labels, convertGithubLabel(l))
		}

		// get reactions
		var reactions []*schemas.Reaction
		if !g.SkipReactions {
			for i := 1; ; i++ {
				g.waitAndPickClient()
				res, resp, err := g.getClient().Reactions.ListIssueReactions(g.ctx, g.repoOwner, g.repoName, pr.GetNumber(), &github.ListOptions{
					Page:    i,
					PerPage: perPage,
				})
				if err != nil {
					return nil, false, err
				}
				g.setRate(&resp.Rate)
				if len(res) == 0 {
					break
				}
				for _, reaction := range res {
					reactions = append(reactions, &schemas.Reaction{
						UserID:   reaction.User.GetID(),
						UserName: reaction.User.GetLogin(),
						Content:  reaction.GetContent(),
					})
				}
			}
		}

		// download patch and saved as tmp file
		g.waitAndPickClient()

		allPRs = append(allPRs, &schemas.PullRequest{
			Title:          pr.GetTitle(),
			Number:         int64(pr.GetNumber()),
			PosterID:       pr.GetUser().GetID(),
			PosterName:     pr.GetUser().GetLogin(),
			PosterEmail:    pr.GetUser().GetEmail(),
			Content:        pr.GetBody(),
			Milestone:      pr.GetMilestone().GetTitle(),
			State:          pr.GetState(),
			Created:        pr.GetCreatedAt(),
			Updated:        pr.GetUpdatedAt(),
			Closed:         pr.ClosedAt,
			Labels:         labels,
			Merged:         pr.MergedAt != nil,
			MergeCommitSHA: pr.GetMergeCommitSHA(),
			MergedTime:     pr.MergedAt,
			IsLocked:       pr.ActiveLockReason != nil,
			Head: schemas.PullRequestBranch{
				Ref:       pr.GetHead().GetRef(),
				SHA:       pr.GetHead().GetSHA(),
				OwnerName: pr.GetHead().GetUser().GetLogin(),
				RepoName:  pr.GetHead().GetRepo().GetName(),
				CloneURL:  pr.GetHead().GetRepo().GetCloneURL(),
			},
			Base: schemas.PullRequestBranch{
				Ref:       pr.GetBase().GetRef(),
				SHA:       pr.GetBase().GetSHA(),
				RepoName:  pr.GetBase().GetRepo().GetName(),
				OwnerName: pr.GetBase().GetUser().GetLogin(),
			},
			PatchURL:     pr.GetPatchURL(),
			Reactions:    reactions,
			ForeignIndex: int64(*pr.Number),
		})
	}

	return allPRs, len(prs) < perPage, nil
}

func convertGithubReview(r *github.PullRequestReview) *schemas.Review {
	return &schemas.Review{
		ID:           r.GetID(),
		ReviewerID:   r.GetUser().GetID(),
		ReviewerName: r.GetUser().GetLogin(),
		CommitID:     r.GetCommitID(),
		Content:      r.GetBody(),
		CreatedAt:    r.GetSubmittedAt(),
		State:        r.GetState(),
	}
}

func (g *GithubDownloaderV3) convertGithubReviewComments(cs []*github.PullRequestComment) ([]*schemas.ReviewComment, error) {
	rcs := make([]*schemas.ReviewComment, 0, len(cs))
	for _, c := range cs {
		// get reactions
		var reactions []*schemas.Reaction
		if !g.SkipReactions {
			for i := 1; ; i++ {
				g.waitAndPickClient()
				res, resp, err := g.getClient().Reactions.ListPullRequestCommentReactions(g.ctx, g.repoOwner, g.repoName, c.GetID(), &github.ListOptions{
					Page:    i,
					PerPage: g.maxPerPage,
				})
				if err != nil {
					return nil, err
				}
				g.setRate(&resp.Rate)
				if len(res) == 0 {
					break
				}
				for _, reaction := range res {
					reactions = append(reactions, &schemas.Reaction{
						UserID:   reaction.User.GetID(),
						UserName: reaction.User.GetLogin(),
						Content:  reaction.GetContent(),
					})
				}
			}
		}

		rcs = append(rcs, &schemas.ReviewComment{
			ID:        c.GetID(),
			InReplyTo: c.GetInReplyTo(),
			Content:   c.GetBody(),
			TreePath:  c.GetPath(),
			DiffHunk:  c.GetDiffHunk(),
			Position:  c.GetPosition(),
			CommitID:  c.GetCommitID(),
			PosterID:  c.GetUser().GetID(),
			Reactions: reactions,
			CreatedAt: c.GetCreatedAt(),
			UpdatedAt: c.GetUpdatedAt(),
		})
	}
	return rcs, nil
}

// GetReviews returns pull requests review
func (g *GithubDownloaderV3) GetReviews(reviewable schemas.Reviewable) ([]*schemas.Review, error) {
	allReviews := make([]*schemas.Review, 0, g.maxPerPage)
	opt := &github.ListOptions{
		PerPage: g.maxPerPage,
	}
	// Get approve/request change reviews
	for {
		g.waitAndPickClient()
		reviews, resp, err := g.getClient().PullRequests.ListReviews(g.ctx, g.repoOwner, g.repoName, int(reviewable.GetForeignIndex()), opt)
		if err != nil {
			return nil, fmt.Errorf("error while listing repos: %v", err)
		}
		g.setRate(&resp.Rate)
		for _, review := range reviews {
			r := convertGithubReview(review)
			r.IssueIndex = reviewable.GetLocalIndex()
			// retrieve all review comments
			opt2 := &github.ListOptions{
				PerPage: g.maxPerPage,
			}
			for {
				g.waitAndPickClient()
				reviewComments, resp, err := g.getClient().PullRequests.ListReviewComments(g.ctx, g.repoOwner, g.repoName, int(reviewable.GetForeignIndex()), review.GetID(), opt2)
				if err != nil {
					return nil, fmt.Errorf("error while listing repos: %v", err)
				}
				g.setRate(&resp.Rate)

				cs, err := g.convertGithubReviewComments(reviewComments)
				if err != nil {
					return nil, err
				}
				r.Comments = append(r.Comments, cs...)
				if resp.NextPage == 0 {
					break
				}
				opt2.Page = resp.NextPage
			}
			allReviews = append(allReviews, r)
		}
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	// Get requested reviews
	for {
		g.waitAndPickClient()
		reviewers, resp, err := g.getClient().PullRequests.ListReviewers(g.ctx, g.repoOwner, g.repoName, int(reviewable.GetForeignIndex()), opt)
		if err != nil {
			return nil, fmt.Errorf("error while listing repos: %v", err)
		}
		g.setRate(&resp.Rate)
		for _, user := range reviewers.Users {
			r := &schemas.Review{
				ReviewerID:   user.GetID(),
				ReviewerName: user.GetLogin(),
				State:        schemas.ReviewStateRequestReview,
				IssueIndex:   reviewable.GetLocalIndex(),
			}
			allReviews = append(allReviews, r)
		}
		// TODO: Handle Team requests
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	return allReviews, nil
}
