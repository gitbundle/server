// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package notify

import (
	"fmt"
	"net/http"

	"github.com/gitbundle/server/pkg/context"
	"github.com/gitbundle/server/pkg/convert"
	"github.com/gitbundle/server/pkg/db"
	issues_model "github.com/gitbundle/server/pkg/issues"
	"github.com/gitbundle/server/pkg/repo/repoman"
)

// GetThread get notification by ID
func GetThread(ctx *context.APIContext) {
	// swagger:operation GET /notifications/threads/{id} notification notifyGetThread
	// ---
	// summary: Get notification thread by ID
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: id of notification thread
	//   type: string
	//   required: true
	// responses:
	//   "200":
	//     "$ref": "#/responses/NotificationThread"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	n := getThread(ctx)
	if n == nil {
		return
	}
	if err := n.LoadAttributes(); err != nil && !issues_model.IsErrCommentNotExist(err) {
		ctx.InternalServerError(err)
		return
	}

	ctx.JSON(http.StatusOK, convert.ToNotificationThread(n))
}

// ReadThread mark notification as read by ID
func ReadThread(ctx *context.APIContext) {
	// swagger:operation PATCH /notifications/threads/{id} notification notifyReadThread
	// ---
	// summary: Mark notification thread as read by ID
	// consumes:
	// - application/json
	// produces:
	// - application/json
	// parameters:
	// - name: id
	//   in: path
	//   description: id of notification thread
	//   type: string
	//   required: true
	// - name: to-status
	//   in: query
	//   description: Status to mark notifications as
	//   type: string
	//   default: read
	//   required: false
	// responses:
	//   "205":
	//     "$ref": "#/responses/NotificationThread"
	//   "403":
	//     "$ref": "#/responses/forbidden"
	//   "404":
	//     "$ref": "#/responses/notFound"

	n := getThread(ctx)
	if n == nil {
		return
	}

	targetStatus := statusStringToNotificationStatus(ctx.FormString("to-status"))
	if targetStatus == 0 {
		targetStatus = repoman.NotificationStatusRead
	}

	notif, err := repoman.SetNotificationStatus(n.ID, ctx.Doer, targetStatus)
	if err != nil {
		ctx.InternalServerError(err)
		return
	}
	if err = notif.LoadAttributes(); err != nil && !issues_model.IsErrCommentNotExist(err) {
		ctx.InternalServerError(err)
		return
	}
	ctx.JSON(http.StatusResetContent, convert.ToNotificationThread(notif))
}

func getThread(ctx *context.APIContext) *repoman.Notification {
	n, err := repoman.GetNotificationByID(ctx.ParamsInt64(":id"))
	if err != nil {
		if db.IsErrNotExist(err) {
			ctx.Error(http.StatusNotFound, "GetNotificationByID", err)
		} else {
			ctx.InternalServerError(err)
		}
		return nil
	}
	if n.UserID != ctx.Doer.ID && !ctx.Doer.IsAdmin {
		ctx.Error(http.StatusForbidden, "GetNotificationByID", fmt.Errorf("only user itself and admin are allowed to read/change this thread %d", n.ID))
		return nil
	}
	return n
}
