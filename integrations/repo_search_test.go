// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package integrations

import (
	"net/http"
	"testing"

	"github.com/gitbundle/modules/setting"
	code_indexer "github.com/gitbundle/server/pkg/indexer/code"
	repo_model "github.com/gitbundle/server/pkg/repo"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func resultFilenames(t testing.TB, doc *HTMLDoc) []string {
	filenameSelections := doc.doc.Find(".repository.search").Find(".repo-search-result").Find(".header").Find("span.file")
	result := make([]string, filenameSelections.Length())
	filenameSelections.Each(func(i int, selection *goquery.Selection) {
		result[i] = selection.Text()
	})
	return result
}

func TestSearchRepo(t *testing.T) {
	defer prepareTestEnv(t)()

	repo, err := repo_model.GetRepositoryByOwnerAndName("user2", "repo1")
	assert.NoError(t, err)

	executeIndexer(t, repo, code_indexer.UpdateRepoIndexer)

	testSearch(t, "/user2/repo1/search?q=Description&page=1", []string{"README.md"})

	setting.Indexer.IncludePatterns = setting.IndexerGlobFromString("**.txt")
	setting.Indexer.ExcludePatterns = setting.IndexerGlobFromString("**/y/**")

	repo, err = repo_model.GetRepositoryByOwnerAndName("user2", "glob")
	assert.NoError(t, err)

	executeIndexer(t, repo, code_indexer.UpdateRepoIndexer)

	testSearch(t, "/user2/glob/search?q=loren&page=1", []string{"a.txt"})
	testSearch(t, "/user2/glob/search?q=file3&page=1", []string{"x/b.txt"})
	testSearch(t, "/user2/glob/search?q=file4&page=1", []string{})
	testSearch(t, "/user2/glob/search?q=file5&page=1", []string{})
}

func testSearch(t *testing.T, url string, expected []string) {
	req := NewRequestf(t, "GET", url)
	resp := MakeRequest(t, req, http.StatusOK)

	filenames := resultFilenames(t, NewHTMLParser(t, resp.Body))
	assert.EqualValues(t, expected, filenames)
}

func executeIndexer(t *testing.T, repo *repo_model.Repository, op func(*repo_model.Repository)) {
	op(repo)
}
