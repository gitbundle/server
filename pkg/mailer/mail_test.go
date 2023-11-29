// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package mailer_test

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"strings"
	"testing"
	texttmpl "text/template"

	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/action"
	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/mailer"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

const subjectTpl = `
{{.SubjectPrefix}}[{{.Repo}}] @{{.Doer.Name}} #{{.Issue.Index}} - {{.Issue.Title}}
`

const bodyTpl = `
<!DOCTYPE html>
<html>
<head>
	<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
	<title>{{.Subject}}</title>
</head>

<body>
	<p>{{.Body}}</p>
	<p>
		---
		<br>
		<a href="{{.Link}}">View it on GitBundle</a>.
	</p>
</body>
</html>
`

func prepareMailerTest(t *testing.T) (doer *user_model.User, repo *repo_model.Repository, issue *issues_model.Issue, comment *issues_model.Comment) {
	assert.NoError(t, unittest.PrepareTestDatabase())
	mailService := setting.Mailer{
		From: "test@gitea.com",
	}

	setting.MailService = &mailService
	setting.Domain = "localhost"

	doer = unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)
	repo = unittest.AssertExistsAndLoadBean(t, &repo_model.Repository{ID: 1, Owner: doer}).(*repo_model.Repository)
	issue = unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 1, Repo: repo, Poster: doer}).(*issues_model.Issue)
	assert.NoError(t, issue.LoadRepo(db.DefaultContext))
	comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 2, Issue: issue}).(*issues_model.Comment)
	return
}

func TestComposeIssueCommentMessage(t *testing.T) {
	doer, _, issue, comment := prepareMailerTest(t)

	stpl := texttmpl.Must(texttmpl.New("issue/comment").Parse(subjectTpl))
	btpl := template.Must(template.New("issue/comment").Parse(bodyTpl))
	mailer.InitMailRender(stpl, btpl)

	recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}, {Name: "Test2", Email: "test2@gitea.com"}}
	msgs, err := composeIssueCommentMessages(&mailCommentContext{
		Context: context.TODO(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: action.ActionCommentIssue,
		Content: "test body", Comment: comment,
	}, "en-US", recipients, false, "issue comment")
	assert.NoError(t, err)
	assert.Len(t, msgs, 2)
	gomailMsg := msgs[0].ToMessage()
	mailto := gomailMsg.GetHeader("To")
	subject := gomailMsg.GetHeader("Subject")
	messageID := gomailMsg.GetHeader("Message-ID")
	inReplyTo := gomailMsg.GetHeader("In-Reply-To")
	references := gomailMsg.GetHeader("References")

	assert.Len(t, mailto, 1, "exactly one recipient is expected in the To field")
	assert.Equal(t, "Re: ", subject[0][:4], "Comment reply subject should contain Re:")
	assert.Equal(t, "Re: [user2/repo1] @user2 #1 - issue1", subject[0])
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", inReplyTo[0], "In-Reply-To header doesn't match")
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", references[0], "References header doesn't match")
	assert.Equal(t, "<user2/repo1/issues/1/comment/2@localhost>", messageID[0], "Message-ID header doesn't match")
}

func TestComposeIssueMessage(t *testing.T) {
	doer, _, issue, _ := prepareMailerTest(t)

	stpl := texttmpl.Must(texttmpl.New("issue/new").Parse(subjectTpl))
	btpl := template.Must(template.New("issue/new").Parse(bodyTpl))
	mailer.InitMailRender(stpl, btpl)

	recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}, {Name: "Test2", Email: "test2@gitea.com"}}
	msgs, err := composeIssueCommentMessages(&mailCommentContext{
		Context: context.TODO(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: action.ActionCreateIssue,
		Content: "test body",
	}, "en-US", recipients, false, "issue create")
	assert.NoError(t, err)
	assert.Len(t, msgs, 2)

	gomailMsg := msgs[0].ToMessage()
	mailto := gomailMsg.GetHeader("To")
	subject := gomailMsg.GetHeader("Subject")
	messageID := gomailMsg.GetHeader("Message-ID")
	inReplyTo := gomailMsg.GetHeader("In-Reply-To")
	references := gomailMsg.GetHeader("References")

	assert.Len(t, mailto, 1, "exactly one recipient is expected in the To field")
	assert.Equal(t, "[user2/repo1] @user2 #1 - issue1", subject[0])
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", inReplyTo[0], "In-Reply-To header doesn't match")
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", references[0], "References header doesn't match")
	assert.Equal(t, "<user2/repo1/issues/1@localhost>", messageID[0], "Message-ID header doesn't match")
}

func TestTemplateSelection(t *testing.T) {
	doer, repo, issue, comment := prepareMailerTest(t)
	recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}}

	stpl := texttmpl.Must(texttmpl.New("issue/default").Parse("issue/default/subject"))
	texttmpl.Must(stpl.New("issue/new").Parse("issue/new/subject"))
	texttmpl.Must(stpl.New("pull/comment").Parse("pull/comment/subject"))
	texttmpl.Must(stpl.New("issue/close").Parse("")) // Must default to fallback subject

	btpl := template.Must(template.New("issue/default").Parse("issue/default/body"))
	template.Must(btpl.New("issue/new").Parse("issue/new/body"))
	template.Must(btpl.New("pull/comment").Parse("pull/comment/body"))
	template.Must(btpl.New("issue/close").Parse("issue/close/body"))

	mailer.InitMailRender(stpl, btpl)

	expect := func(t *testing.T, msg *mailer.Message, expSubject, expBody string) {
		subject := msg.ToMessage().GetHeader("Subject")
		msgbuf := new(bytes.Buffer)
		_, _ = msg.ToMessage().WriteTo(msgbuf)
		wholemsg := msgbuf.String()
		assert.Equal(t, []string{expSubject}, subject)
		assert.Contains(t, wholemsg, expBody)
	}

	msg := testComposeIssueCommentMessage(t, &mailCommentContext{
		Context: context.TODO(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: action.ActionCreateIssue,
		Content: "test body",
	}, recipients, false, "TestTemplateSelection")
	expect(t, msg, "issue/new/subject", "issue/new/body")

	msg = testComposeIssueCommentMessage(t, &mailCommentContext{
		Context: context.TODO(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: action.ActionCommentIssue,
		Content: "test body", Comment: comment,
	}, recipients, false, "TestTemplateSelection")
	expect(t, msg, "issue/default/subject", "issue/default/body")

	pull := unittest.AssertExistsAndLoadBean(t, &issues_model.Issue{ID: 2, Repo: repo, Poster: doer}).(*issues_model.Issue)
	comment = unittest.AssertExistsAndLoadBean(t, &issues_model.Comment{ID: 4, Issue: pull}).(*issues_model.Comment)
	msg = testComposeIssueCommentMessage(t, &mailCommentContext{
		Context: context.TODO(), // TODO: use a correct context
		Issue:   pull, Doer: doer, ActionType: action.ActionCommentPull,
		Content: "test body", Comment: comment,
	}, recipients, false, "TestTemplateSelection")
	expect(t, msg, "pull/comment/subject", "pull/comment/body")

	msg = testComposeIssueCommentMessage(t, &mailCommentContext{
		Context: context.TODO(), // TODO: use a correct context
		Issue:   issue, Doer: doer, ActionType: action.ActionCloseIssue,
		Content: "test body", Comment: comment,
	}, recipients, false, "TestTemplateSelection")
	expect(t, msg, "Re: [user2/repo1] issue1 (#1)", "issue/close/body")
}

func TestTemplateServices(t *testing.T) {
	doer, _, issue, comment := prepareMailerTest(t)
	assert.NoError(t, issue.LoadRepo(db.DefaultContext))

	expect := func(t *testing.T, issue *issues_model.Issue, comment *issues_model.Comment, doer *user_model.User,
		actionType action.ActionType, fromMention bool, tplSubject, tplBody, expSubject, expBody string,
	) {
		stpl := texttmpl.Must(texttmpl.New("issue/default").Parse(tplSubject))
		btpl := template.Must(template.New("issue/default").Parse(tplBody))
		mailer.InitMailRender(stpl, btpl)

		recipients := []*user_model.User{{Name: "Test", Email: "test@gitea.com"}}
		msg := testComposeIssueCommentMessage(t, &mailCommentContext{
			Context: context.TODO(), // TODO: use a correct context
			Issue:   issue, Doer: doer, ActionType: actionType,
			Content: "test body", Comment: comment,
		}, recipients, fromMention, "TestTemplateServices")

		subject := msg.ToMessage().GetHeader("Subject")
		msgbuf := new(bytes.Buffer)
		_, _ = msg.ToMessage().WriteTo(msgbuf)
		wholemsg := msgbuf.String()

		assert.Equal(t, []string{expSubject}, subject)
		assert.Contains(t, wholemsg, "\r\n"+expBody+"\r\n")
	}

	expect(t, issue, comment, doer, action.ActionCommentIssue, false,
		"{{.SubjectPrefix}}[{{.Repo}}]: @{{.Doer.Name}} commented on #{{.Issue.Index}} - {{.Issue.Title}}",
		"//{{.ActionType}},{{.ActionName}},{{if .IsMention}}norender{{end}}//",
		"Re: [user2/repo1]: @user2 commented on #1 - issue1",
		"//issue,comment,//")

	expect(t, issue, comment, doer, action.ActionCommentIssue, true,
		"{{if .IsMention}}must render{{end}}",
		"//subject is: {{.Subject}}//",
		"must render",
		"//subject is: must render//")

	expect(t, issue, comment, doer, action.ActionCommentIssue, true,
		"{{.FallbackSubject}}",
		"//{{.SubjectPrefix}}//",
		"Re: [user2/repo1] issue1 (#1)",
		"//Re: //")
}

func testComposeIssueCommentMessage(t *testing.T, ctx *mailCommentContext, recipients []*user_model.User, fromMention bool, info string) *mailer.Message {
	msgs, err := composeIssueCommentMessages(ctx, "en-US", recipients, fromMention, info)
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)
	return msgs[0]
}

func TestGenerateAdditionalHeaders(t *testing.T) {
	doer, _, issue, _ := prepareMailerTest(t)

	ctx := &mailCommentContext{Context: context.TODO() /* TODO: use a correct context */, Issue: issue, Doer: doer}
	recipient := &user_model.User{Name: "Test", Email: "test@gitea.com"}

	headers := generateAdditionalHeaders(ctx, "dummy-reason", recipient)

	expected := map[string]string{
		"List-ID":                   "user2/repo1 <repo1.user2.localhost>",
		"List-Archive":              "<https://try.gitea.io/user2/repo1>",
		"List-Unsubscribe":          "https://try.gitea.io/user2/repo1/issues/1",
		"X-Gitea-Reason":            "dummy-reason",
		"X-Gitea-Sender":            "< U<se>r Tw<o > ><",
		"X-Gitea-Recipient":         "Test",
		"X-Gitea-Recipient-Address": "test@gitea.com",
		"X-Gitea-Repository":        "repo1",
		"X-Gitea-Repository-Path":   "user2/repo1",
		"X-Gitea-Repository-Link":   "https://try.gitea.io/user2/repo1",
		"X-Gitea-Issue-ID":          "1",
		"X-Gitea-Issue-Link":        "https://try.gitea.io/user2/repo1/issues/1",
	}

	for key, value := range expected {
		if assert.Contains(t, headers, key) {
			assert.Equal(t, value, headers[key])
		}
	}
}

func Test_createReference(t *testing.T) {
	_, _, issue, comment := prepareMailerTest(t)
	_, _, pullIssue, _ := prepareMailerTest(t)
	pullIssue.IsPull = true

	type args struct {
		issue      *issues_model.Issue
		comment    *issues_model.Comment
		actionType action.ActionType
	}
	tests := []struct {
		name   string
		args   args
		prefix string
		suffix string
	}{
		{
			name: "Open Issue",
			args: args{
				issue:      issue,
				actionType: action.ActionCreateIssue,
			},
			prefix: fmt.Sprintf("%s/issues/%d@%s", issue.Repo.FullName(), issue.Index, setting.Domain),
		},
		{
			name: "Open Pull",
			args: args{
				issue:      pullIssue,
				actionType: action.ActionCreatePullRequest,
			},
			prefix: fmt.Sprintf("%s/pulls/%d@%s", issue.Repo.FullName(), issue.Index, setting.Domain),
		},
		{
			name: "Comment Issue",
			args: args{
				issue:      issue,
				comment:    comment,
				actionType: action.ActionCommentIssue,
			},
			prefix: fmt.Sprintf("%s/issues/%d/comment/%d@%s", issue.Repo.FullName(), issue.Index, comment.ID, setting.Domain),
		},
		{
			name: "Comment Pull",
			args: args{
				issue:      pullIssue,
				comment:    comment,
				actionType: action.ActionCommentPull,
			},
			prefix: fmt.Sprintf("%s/pulls/%d/comment/%d@%s", issue.Repo.FullName(), issue.Index, comment.ID, setting.Domain),
		},
		{
			name: "Close Issue",
			args: args{
				issue:      issue,
				actionType: action.ActionCloseIssue,
			},
			prefix: fmt.Sprintf("%s/issues/%d/close/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Close Pull",
			args: args{
				issue:      pullIssue,
				actionType: action.ActionClosePullRequest,
			},
			prefix: fmt.Sprintf("%s/pulls/%d/close/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Reopen Issue",
			args: args{
				issue:      issue,
				actionType: action.ActionReopenIssue,
			},
			prefix: fmt.Sprintf("%s/issues/%d/reopen/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Reopen Pull",
			args: args{
				issue:      pullIssue,
				actionType: action.ActionReopenPullRequest,
			},
			prefix: fmt.Sprintf("%s/pulls/%d/reopen/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Merge Pull",
			args: args{
				issue:      pullIssue,
				actionType: action.ActionMergePullRequest,
			},
			prefix: fmt.Sprintf("%s/pulls/%d/merge/", issue.Repo.FullName(), issue.Index),
		},
		{
			name: "Ready Pull",
			args: args{
				issue:      pullIssue,
				actionType: action.ActionPullRequestReadyForReview,
			},
			prefix: fmt.Sprintf("%s/pulls/%d/ready/", issue.Repo.FullName(), issue.Index),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createReference(tt.args.issue, tt.args.comment, tt.args.actionType)
			if !strings.HasPrefix(got, tt.prefix) {
				t.Errorf("createReference() = %v, want %v", got, tt.prefix)
			}
			if !strings.HasSuffix(got, tt.suffix) {
				t.Errorf("createReference() = %v, want %v", got, tt.prefix)
			}
		})
	}
}
