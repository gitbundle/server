// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gitbundle/server/pkg/fixtures"
	"github.com/gitbundle/server/pkg/unittest"
)

// To generate derivative fixtures, execute the following from GitBundle's repository base dir:
// go run -tags 'sqlite sqlite_unlock_notify' contrib/fixtures/fixture_generation.go [fixture...]

var (
	generators = []struct {
		gen  func() (string, error)
		name string
	}{
		{
			fixtures.GetYamlFixturesAccess, "access",
		},
	}
	fixturesDir string
)

func main() {
	pathToGiteaRoot := "."
	fixturesDir = filepath.Join(pathToGiteaRoot, "pkg", "fixtures")
	if err := unittest.CreateTestEngine(unittest.FixturesOptions{
		Dir: fixturesDir,
	}); err != nil {
		fmt.Printf("CreateTestEngine: %+v", err)
		os.Exit(1)
	}
	if err := unittest.PrepareTestDatabase(); err != nil {
		fmt.Printf("PrepareTestDatabase: %+v\n", err)
		os.Exit(1)
	}
	if len(os.Args) == 0 {
		for _, r := range os.Args {
			if err := generate(r); err != nil {
				fmt.Printf("generate '%s': %+v\n", r, err)
				os.Exit(1)
			}
		}
	} else {
		for _, g := range generators {
			if err := generate(g.name); err != nil {
				fmt.Printf("generate '%s': %+v\n", g.name, err)
				os.Exit(1)
			}
		}
	}
}

func generate(name string) error {
	for _, g := range generators {
		if g.name == name {
			data, err := g.gen()
			if err != nil {
				return err
			}
			path := filepath.Join(fixturesDir, name+".yml")
			if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
				return fmt.Errorf("%s: %+v", path, err)
			}
			fmt.Printf("%s created.\n", path)
			return nil
		}
	}

	return fmt.Errorf("generator not found")
}
