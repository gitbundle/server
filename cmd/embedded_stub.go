// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

//go:build !bindata

package cmd

import (
	"fmt"
	"os"

	"github.com/urfave/cli"
)

// Cmdembedded represents the available extract sub-command.
var (
	Cmdembedded = cli.Command{
		Name:        "embedded",
		Usage:       "Extract embedded resources",
		Description: "A command for extracting embedded resources, like templates and images",
		Action:      extractorNotImplemented,
	}
)

func extractorNotImplemented(c *cli.Context) error {
	err := fmt.Errorf("Sorry: the 'embedded' subcommand is not available in builds without bindata")
	fmt.Fprintf(os.Stderr, "%s\n", err)
	return err
}
