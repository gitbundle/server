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

	"github.com/gitbundle/modules/setting"
)

// UpdatePublicKeyInRepo update public key and if necessary deploy key updates
func UpdatePublicKeyInRepo(ctx context.Context, keyID, repoID int64) error {
	// Ask for running deliver hook and test pull request tasks.
	reqURL := setting.LocalURL + fmt.Sprintf("api/internal/ssh/%d/update/%d", keyID, repoID)
	resp, err := newInternalRequest(ctx, reqURL, "POST").Response()
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// All 2XX status codes are accepted and others will return an error
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("Failed to update public key: %s", decodeJSONError(resp).Err)
	}
	return nil
}

// AuthorizedPublicKeyByContent searches content as prefix (leak e-mail part)
// and returns public key found.
func AuthorizedPublicKeyByContent(ctx context.Context, content string) (string, error) {
	// Ask for running deliver hook and test pull request tasks.
	reqURL := setting.LocalURL + "api/internal/ssh/authorized_keys"
	req := newInternalRequest(ctx, reqURL, "POST")
	req.Param("content", content)
	resp, err := req.Response()
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	// All 2XX status codes are accepted and others will return an error
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Failed to update public key: %s", decodeJSONError(resp).Err)
	}
	bs, err := io.ReadAll(resp.Body)

	return string(bs), err
}
