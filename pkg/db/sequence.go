// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package db

import (
	"fmt"
	"regexp"

	"github.com/gitbundle/modules/setting"
)

// CountBadSequences looks for broken sequences from recreate-table mistakes
func CountBadSequences() (int64, error) {
	if !setting.Database.UsePostgreSQL {
		return 0, nil
	}

	sess := x.NewSession()
	defer sess.Close()

	var sequences []string
	schema := x.Dialect().URI().Schema

	sess.Engine().SetSchema("")
	if err := sess.Table("information_schema.sequences").Cols("sequence_name").Where("sequence_name LIKE 'tmp_recreate__%_id_seq%' AND sequence_catalog = ?", setting.Database.Name).Find(&sequences); err != nil {
		return 0, err
	}
	sess.Engine().SetSchema(schema)

	return int64(len(sequences)), nil
}

// FixBadSequences fixes for broken sequences from recreate-table mistakes
func FixBadSequences() error {
	if !setting.Database.UsePostgreSQL {
		return nil
	}

	sess := x.NewSession()
	defer sess.Close()
	if err := sess.Begin(); err != nil {
		return err
	}

	var sequences []string
	schema := sess.Engine().Dialect().URI().Schema

	sess.Engine().SetSchema("")
	if err := sess.Table("information_schema.sequences").Cols("sequence_name").Where("sequence_name LIKE 'tmp_recreate__%_id_seq%' AND sequence_catalog = ?", setting.Database.Name).Find(&sequences); err != nil {
		return err
	}
	sess.Engine().SetSchema(schema)

	sequenceRegexp := regexp.MustCompile(`tmp_recreate__(\w+)_id_seq.*`)

	for _, sequence := range sequences {
		tableName := sequenceRegexp.FindStringSubmatch(sequence)[1]
		newSequenceName := tableName + "_id_seq"
		if _, err := sess.Exec(fmt.Sprintf("ALTER SEQUENCE `%s` RENAME TO `%s`", sequence, newSequenceName)); err != nil {
			return err
		}
		if _, err := sess.Exec(fmt.Sprintf("SELECT setval('%s', COALESCE((SELECT MAX(id)+1 FROM `%s`), 1), false)", newSequenceName, tableName)); err != nil {
			return err
		}
	}

	return sess.Commit()
}
