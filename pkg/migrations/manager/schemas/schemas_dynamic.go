// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

//go:build !bindata

package schemas

import (
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

func openSchema(s string) (io.ReadCloser, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	basename := path.Base(u.Path)
	filename := basename
	//
	// Schema reference each other within the schemas directory but
	// the tests run in the parent directory.
	//
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		filename = filepath.Join("schemas", basename)
		//
		// Integration tests run from the git root directory, not the
		// directory in which the test source is located.
		//
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			filename = filepath.Join("modules/migration/schemas", basename)
		}
	}
	return os.Open(filename)
}
