// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package migrations

import (
	"github.com/gitbundle/modules/timeutil"

	"xorm.io/xorm"
)

func addTimeStamps(x *xorm.Engine) error {
	// this will add timestamps where it is useful to have

	// Star represents a starred repo by an user.
	type Star struct {
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	}
	if err := x.Sync2(new(Star)); err != nil {
		return err
	}

	// Label represents a label of repository for issues.
	type Label struct {
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
		UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
	}
	if err := x.Sync2(new(Label)); err != nil {
		return err
	}

	// Follow represents relations of user and his/her followers.
	type Follow struct {
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
	}
	if err := x.Sync2(new(Follow)); err != nil {
		return err
	}

	// Watch is connection request for receiving repository notification.
	type Watch struct {
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
		UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
	}
	if err := x.Sync2(new(Watch)); err != nil {
		return err
	}

	// Collaboration represent the relation between an individual and a repository.
	type Collaboration struct {
		CreatedUnix timeutil.TimeStamp `xorm:"INDEX created"`
		UpdatedUnix timeutil.TimeStamp `xorm:"INDEX updated"`
	}
	return x.Sync2(new(Collaboration))
}
