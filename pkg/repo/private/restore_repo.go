// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package private

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gitbundle/modules/json"
	"github.com/gitbundle/modules/setting"
)

// RestoreParams structure holds a data for restore repository
type RestoreParams struct {
	RepoDir    string
	OwnerName  string
	RepoName   string
	Units      []string
	Validation bool
}

// RestoreRepo calls the internal RestoreRepo function
func RestoreRepo(ctx context.Context, repoDir, ownerName, repoName string, units []string, validation bool) (int, string) {
	reqURL := setting.LocalURL + "api/internal/restore_repo"

	req := newInternalRequest(ctx, reqURL, "POST")
	req.SetTimeout(3*time.Second, 0) // since the request will spend much time, don't timeout
	req = req.Header("Content-Type", "application/json")
	jsonBytes, _ := json.Marshal(RestoreParams{
		RepoDir:    repoDir,
		OwnerName:  ownerName,
		RepoName:   repoName,
		Units:      units,
		Validation: validation,
	})
	req.Body(jsonBytes)
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v, could you confirm it's running?", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ret := struct {
			Err string `json:"err"`
		}{}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return http.StatusInternalServerError, fmt.Sprintf("Response body error: %v", err.Error())
		}
		if err := json.Unmarshal(body, &ret); err != nil {
			return http.StatusInternalServerError, fmt.Sprintf("Response body Unmarshal error: %v", err.Error())
		}
		return http.StatusInternalServerError, ret.Err
	}

	return http.StatusOK, fmt.Sprintf("Restore repo %s/%s successfully", ownerName, repoName)
}
