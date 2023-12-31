// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package migrations

import (
	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

func migrateUserPasswordSalt(x *xorm.Engine) error {
	dbType := x.Dialect().URI().DBType
	// For SQLITE, the max length doesn't matter.
	if dbType == schemas.SQLITE {
		return nil
	}

	if err := modifyColumn(x, "user", &schemas.Column{
		Name: "rands",
		SQLType: schemas.SQLType{
			Name: "VARCHAR",
		},
		Length: 32,
		// MySQL will like us again.
		Nullable:       true,
		DefaultIsEmpty: true,
	}); err != nil {
		return err
	}

	return modifyColumn(x, "user", &schemas.Column{
		Name: "salt",
		SQLType: schemas.SQLType{
			Name: "VARCHAR",
		},
		Length:         32,
		Nullable:       true,
		DefaultIsEmpty: true,
	})
}
