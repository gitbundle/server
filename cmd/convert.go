// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"

	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/server/pkg/db"

	"github.com/urfave/cli"
)

// CmdConvert represents the available convert sub-command.
var CmdConvert = cli.Command{
	Name:        "convert",
	Usage:       "Convert the database",
	Description: "A command to convert an existing MySQL database from utf8 to utf8mb4",
	Action:      runConvert,
}

func runConvert(ctx *cli.Context) error {
	stdCtx, cancel := installSignals()
	defer cancel()

	if err := initDB(stdCtx); err != nil {
		return err
	}

	log.Info("AppPath: %s", setting.AppPath)
	log.Info("AppWorkPath: %s", setting.AppWorkPath)
	log.Info("Custom path: %s", setting.CustomPath)
	log.Info("Log path: %s", setting.LogRootPath)
	log.Info("Configuration file: %s", setting.CustomConf)

	if !setting.Database.UseMySQL {
		fmt.Println("This command can only be used with a MySQL database")
		return nil
	}

	if err := db.ConvertUtf8ToUtf8mb4(); err != nil {
		log.Fatal("Failed to convert database from utf8 to utf8mb4: %v", err)
		return err
	}

	fmt.Println("Converted successfully, please confirm your database's character set is now utf8mb4")

	return nil
}
