// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package appstate

import (
	"context"

	"github.com/gitbundle/server/pkg/db"
)

// AppState represents a state record in database
// if one day we would make GitBundle run as a cluster,
// we can introduce a new field `Scope` here to store different states for different nodes
type AppState struct {
	ID       string `xorm:"pk varchar(200)"`
	Revision int64
	Content  string `xorm:"LONGTEXT"`
}

func init() {
	db.RegisterModel(new(AppState))
}

// SaveAppStateContent saves the app state item to database
func SaveAppStateContent(key, content string) error {
	return db.WithTx(func(ctx context.Context) error {
		eng := db.GetEngine(ctx)
		// try to update existing row
		res, err := eng.Exec("UPDATE app_state SET revision=revision+1, content=? WHERE id=?", content, key)
		if err != nil {
			return err
		}
		rows, _ := res.RowsAffected()
		if rows != 0 {
			// the existing row is updated, so we can return
			return nil
		}
		// if no existing row, insert a new row
		_, err = eng.Insert(&AppState{ID: key, Content: content})
		return err
	})
}

// GetAppStateContent gets an app state from database
func GetAppStateContent(key string) (content string, err error) {
	e := db.GetEngine(db.DefaultContext)
	appState := &AppState{ID: key}
	has, err := e.Get(appState)
	if err != nil {
		return "", err
	} else if !has {
		return "", nil
	}
	return appState.Content, nil
}
