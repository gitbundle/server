// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package install

import (
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/db"

	"xorm.io/xorm"
)

func getXORMEngine() *xorm.Engine {
	return db.DefaultContext.(*db.Context).Engine().(*xorm.Engine)
}

// CheckDatabaseConnection checks the database connection
func CheckDatabaseConnection() error {
	e := db.GetEngine(db.DefaultContext)
	_, err := e.Exec("SELECT 1")
	return err
}

// GetMigrationVersion gets the database migration version
func GetMigrationVersion() (int64, error) {
	var installedDbVersion int64
	x := getXORMEngine()
	exist, err := x.IsTableExist("version")
	if err != nil {
		return 0, err
	}
	if !exist {
		return 0, nil
	}
	_, err = x.Table("version").Cols("version").Get(&installedDbVersion)
	if err != nil {
		return 0, err
	}
	return installedDbVersion, nil
}

// HasPostInstallationUsers checks whether there are users after installation
func HasPostInstallationUsers() (bool, error) {
	x := getXORMEngine()
	exist, err := x.IsTableExist("user")
	if err != nil {
		return false, err
	}
	if !exist {
		return false, nil
	}

	// if there are 2 or more users in database, we consider there are users created after installation
	threshold := 2
	if !setting.IsProd {
		// to debug easily, with non-prod RUN_MODE, we only check the count to 1
		threshold = 1
	}
	res, err := x.Table("user").Cols("id").Limit(threshold).Query()
	if err != nil {
		return false, err
	}
	return len(res) >= threshold, nil
}
