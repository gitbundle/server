// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package db

import (
	"fmt"
	"strconv"

	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/setting"

	"xorm.io/xorm"
	"xorm.io/xorm/schemas"
)

// ConvertUtf8ToUtf8mb4 converts database and tables from utf8 to utf8mb4 if it's mysql and set ROW_FORMAT=dynamic
func ConvertUtf8ToUtf8mb4() error {
	if x.Dialect().URI().DBType != schemas.MYSQL {
		return nil
	}

	_, err := x.Exec(fmt.Sprintf("ALTER DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci", setting.Database.Name))
	if err != nil {
		return err
	}

	tables, err := x.DBMetas()
	if err != nil {
		return err
	}
	for _, table := range tables {
		if _, err := x.Exec(fmt.Sprintf("ALTER TABLE `%s` ROW_FORMAT=dynamic;", table.Name)); err != nil {
			return err
		}

		if _, err := x.Exec(fmt.Sprintf("ALTER TABLE `%s` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;", table.Name)); err != nil {
			return err
		}
	}

	return nil
}

// Cell2Int64 converts a xorm.Cell type to int64,
// and handles possible irregular cases.
func Cell2Int64(val xorm.Cell) int64 {
	switch (*val).(type) {
	case []uint8:
		log.Trace("Cell2Int64 ([]uint8): %v", *val)

		v, _ := strconv.ParseInt(string((*val).([]uint8)), 10, 64)
		return v
	}
	return (*val).(int64)
}
