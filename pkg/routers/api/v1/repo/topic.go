// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package repo

import (
	"net/http"
	"strings"

	api "github.com/gitbundle/api/pkg/structs"
	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/server/pkg/context"
	"github.com/gitbundle/server/pkg/convert"
	repo_model "github.com/gitbundle/server/pkg/repo"
	"github.com/gitbundle/server/pkg/routers/api/v1/utils"
	"github.com/gitbundle/server/pkg/web"
)

// ListTopics returns list of current topics for repo
func ListTopics(ctx *context.APIContext) {
	// swagger:operation GET /repos/{owner}/{repo}/topics repository repoListTopics
	// ---
	// summary: Get list of topics that a repository has
	// produces:
	//   - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: page
	//   in: query
	//   description: page number of results to return (1-based)
	//   type: integer
	// - name: limit
	//   in: query
	//   description: page size of results
	//   type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/TopicNames"

	opts := &repo_model.FindTopicOptions{
		ListOptions: utils.GetListOptions(ctx),
		RepoID:      ctx.Repo.Repository.ID,
	}

	topics, total, err := repo_model.FindTopics(opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	topicNames := make([]string, len(topics))
	for i, topic := range topics {
		topicNames[i] = topic.Name
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"topics": topicNames,
	})
}

// UpdateTopics updates repo with a new set of topics
func UpdateTopics(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/topics repository repoUpdateTopics
	// ---
	// summary: Replace list of topics for a repository
	// produces:
	//   - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: body
	//   in: body
	//   schema:
	//     "$ref": "#/definitions/RepoTopicOptions"
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "422":
	//     "$ref": "#/responses/invalidTopicsError"

	form := web.GetForm(ctx).(*api.RepoTopicOptions)
	topicNames := form.Topics
	validTopics, invalidTopics := repo_model.SanitizeAndValidateTopics(topicNames)

	if len(validTopics) > 25 {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"invalidTopics": nil,
			"message":       "Exceeding maximum number of topics per repo",
		})
		return
	}

	if len(invalidTopics) > 0 {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"invalidTopics": invalidTopics,
			"message":       "Topic names are invalid",
		})
		return
	}

	err := repo_model.SaveTopics(ctx.Repo.Repository.ID, validTopics...)
	if err != nil {
		log.Error("SaveTopics failed: %v", err)
		ctx.InternalServerError(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// AddTopic adds a topic name to a repo
func AddTopic(ctx *context.APIContext) {
	// swagger:operation PUT /repos/{owner}/{repo}/topics/{topic} repository repoAddTopic
	// ---
	// summary: Add a topic to a repository
	// produces:
	//   - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: topic
	//   in: path
	//   description: name of the topic to add
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "422":
	//     "$ref": "#/responses/invalidTopicsError"

	topicName := strings.TrimSpace(strings.ToLower(ctx.Params(":topic")))

	if !repo_model.ValidateTopic(topicName) {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"invalidTopics": topicName,
			"message":       "Topic name is invalid",
		})
		return
	}

	// Prevent adding more topics than allowed to repo
	count, err := repo_model.CountTopics(&repo_model.FindTopicOptions{
		RepoID: ctx.Repo.Repository.ID,
	})
	if err != nil {
		log.Error("CountTopics failed: %v", err)
		ctx.InternalServerError(err)
		return
	}
	if count >= 25 {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Exceeding maximum allowed topics per repo.",
		})
		return
	}

	_, err = repo_model.AddTopic(ctx.Repo.Repository.ID, topicName)
	if err != nil {
		log.Error("AddTopic failed: %v", err)
		ctx.InternalServerError(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// DeleteTopic removes topic name from repo
func DeleteTopic(ctx *context.APIContext) {
	// swagger:operation DELETE /repos/{owner}/{repo}/topics/{topic} repository repoDeleteTopic
	// ---
	// summary: Delete a topic from a repository
	// produces:
	//   - application/json
	// parameters:
	// - name: owner
	//   in: path
	//   description: owner of the repo
	//   type: string
	//   required: true
	// - name: repo
	//   in: path
	//   description: name of the repo
	//   type: string
	//   required: true
	// - name: topic
	//   in: path
	//   description: name of the topic to delete
	//   type: string
	//   required: true
	// responses:
	//   "204":
	//     "$ref": "#/responses/empty"
	//   "422":
	//     "$ref": "#/responses/invalidTopicsError"

	topicName := strings.TrimSpace(strings.ToLower(ctx.Params(":topic")))

	if !repo_model.ValidateTopic(topicName) {
		ctx.JSON(http.StatusUnprocessableEntity, map[string]interface{}{
			"invalidTopics": topicName,
			"message":       "Topic name is invalid",
		})
		return
	}

	topic, err := repo_model.DeleteTopic(ctx.Repo.Repository.ID, topicName)
	if err != nil {
		log.Error("DeleteTopic failed: %v", err)
		ctx.InternalServerError(err)
		return
	}

	if topic == nil {
		ctx.NotFound()
		return
	}

	ctx.Status(http.StatusNoContent)
}

// TopicSearch search for creating topic
func TopicSearch(ctx *context.APIContext) {
	// swagger:operation GET /topics/search repository topicSearch
	// ---
	// summary: search topics via keyword
	// produces:
	//   - application/json
	// parameters:
	//   - name: q
	//     in: query
	//     description: keywords to search
	//     required: true
	//     type: string
	//   - name: page
	//     in: query
	//     description: page number of results to return (1-based)
	//     type: integer
	//   - name: limit
	//     in: query
	//     description: page size of results
	//     type: integer
	// responses:
	//   "200":
	//     "$ref": "#/responses/TopicListResponse"
	//   "403":
	//     "$ref": "#/responses/forbidden"

	opts := &repo_model.FindTopicOptions{
		Keyword:     ctx.FormString("q"),
		ListOptions: utils.GetListOptions(ctx),
	}

	topics, total, err := repo_model.FindTopics(opts)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}

	topicResponses := make([]*api.TopicResponse, len(topics))
	for i, topic := range topics {
		topicResponses[i] = convert.ToTopicResponse(topic)
	}

	ctx.SetTotalCountHeader(total)
	ctx.JSON(http.StatusOK, map[string]interface{}{
		"topics": topicResponses,
	})
}
