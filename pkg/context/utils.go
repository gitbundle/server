// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package context

import (
	"net/url"
	"strings"
	"time"
)

// GetQueryBeforeSince return parsed time (unix format) from URL query's before and since
func GetQueryBeforeSince(ctx *Context) (before, since int64, err error) {
	qCreatedBefore, err := prepareQueryArg(ctx, "before")
	if err != nil {
		return 0, 0, err
	}

	qCreatedSince, err := prepareQueryArg(ctx, "since")
	if err != nil {
		return 0, 0, err
	}

	before, err = parseTime(qCreatedBefore)
	if err != nil {
		return 0, 0, err
	}

	since, err = parseTime(qCreatedSince)
	if err != nil {
		return 0, 0, err
	}
	return before, since, nil
}

// parseTime parse time and return unix timestamp
func parseTime(value string) (int64, error) {
	if len(value) != 0 {
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			return 0, err
		}
		if !t.IsZero() {
			return t.Unix(), nil
		}
	}
	return 0, nil
}

// prepareQueryArg unescape and trim a query arg
func prepareQueryArg(ctx *Context, name string) (value string, err error) {
	value, err = url.PathUnescape(ctx.FormString(name))
	value = strings.TrimSpace(value)
	return
}
