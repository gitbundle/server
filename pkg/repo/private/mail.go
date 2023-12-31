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

	"github.com/gitbundle/modules/json"
	"github.com/gitbundle/modules/setting"
)

// Email structure holds a data for sending general emails
type Email struct {
	Subject string
	Message string
	To      []string
}

// SendEmail calls the internal SendEmail function
//
// It accepts a list of usernames.
// If DB contains these users it will send the email to them.
//
// If to list == nil its supposed to send an email to every
// user present in DB
func SendEmail(ctx context.Context, subject, message string, to []string) (int, string) {
	reqURL := setting.LocalURL + "api/internal/mail/send"

	req := newInternalRequest(ctx, reqURL, "POST")
	req = req.Header("Content-Type", "application/json")
	jsonBytes, _ := json.Marshal(Email{
		Subject: subject,
		Message: message,
		To:      to,
	})
	req.Body(jsonBytes)
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Response body error: %v", err.Error())
	}

	users := fmt.Sprintf("%d", len(to))
	if len(to) == 0 {
		users = "all"
	}

	return http.StatusOK, fmt.Sprintf("Sent %s email(s) to %s users", body, users)
}
