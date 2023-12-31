// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package integrations

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gitbundle/modules/packages/maven"
	"github.com/gitbundle/server/pkg/db"
	"github.com/gitbundle/server/pkg/packages"
	"github.com/gitbundle/server/pkg/unittest"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestPackageMaven(t *testing.T) {
	defer prepareTestEnv(t)()
	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2}).(*user_model.User)

	groupID := "com.gitea"
	artifactID := "test-project"
	packageName := groupID + "-" + artifactID
	packageVersion := "1.0.1"
	packageDescription := "Test Description"

	root := fmt.Sprintf("/api/packages/%s/maven/%s/%s", user.Name, strings.ReplaceAll(groupID, ".", "/"), artifactID)
	filename := fmt.Sprintf("%s-%s.jar", packageName, packageVersion)

	putFile := func(t *testing.T, path, content string, expectedStatus int) {
		req := NewRequestWithBody(t, "PUT", root+path, strings.NewReader(content))
		req = AddBasicAuthHeader(req, user.Name)
		MakeRequest(t, req, expectedStatus)
	}

	t.Run("Upload", func(t *testing.T) {
		defer PrintCurrentTest(t)()

		putFile(t, fmt.Sprintf("/%s/%s", packageVersion, filename), "test", http.StatusCreated)
		putFile(t, "/maven-metadata.xml", "test", http.StatusOK)

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeMaven)
		assert.NoError(t, err)
		assert.Len(t, pvs, 1)

		pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
		assert.NoError(t, err)
		assert.Nil(t, pd.SemVer)
		assert.Nil(t, pd.Metadata)
		assert.Equal(t, packageName, pd.Package.Name)
		assert.Equal(t, packageVersion, pd.Version.Version)

		pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
		assert.NoError(t, err)
		assert.Len(t, pfs, 1)
		assert.Equal(t, filename, pfs[0].Name)
		assert.False(t, pfs[0].IsLead)

		pb, err := packages.GetBlobByID(db.DefaultContext, pfs[0].BlobID)
		assert.NoError(t, err)
		assert.Equal(t, int64(4), pb.Size)
	})

	t.Run("UploadExists", func(t *testing.T) {
		defer PrintCurrentTest(t)()

		putFile(t, fmt.Sprintf("/%s/%s", packageVersion, filename), "test", http.StatusBadRequest)
	})

	t.Run("Download", func(t *testing.T) {
		defer PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/%s/%s", root, packageVersion, filename))
		req = AddBasicAuthHeader(req, user.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, []byte("test"), resp.Body.Bytes())

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeMaven)
		assert.NoError(t, err)
		assert.Len(t, pvs, 1)
		assert.Equal(t, int64(0), pvs[0].DownloadCount)
	})

	t.Run("UploadVerifySHA1", func(t *testing.T) {
		defer PrintCurrentTest(t)()

		t.Run("Missmatch", func(t *testing.T) {
			defer PrintCurrentTest(t)()

			putFile(t, fmt.Sprintf("/%s/%s.sha1", packageVersion, filename), "test", http.StatusBadRequest)
		})
		t.Run("Valid", func(t *testing.T) {
			defer PrintCurrentTest(t)()

			putFile(t, fmt.Sprintf("/%s/%s.sha1", packageVersion, filename), "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3", http.StatusOK)
		})
	})

	pomContent := `<?xml version="1.0"?>
<project xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
  <groupId>` + groupID + `</groupId>
  <artifactId>` + artifactID + `</artifactId>
  <version>` + packageVersion + `</version>
  <description>` + packageDescription + `</description>
</project>`

	t.Run("UploadPOM", func(t *testing.T) {
		defer PrintCurrentTest(t)()

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeMaven)
		assert.NoError(t, err)
		assert.Len(t, pvs, 1)

		pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
		assert.NoError(t, err)
		assert.Nil(t, pd.Metadata)

		putFile(t, fmt.Sprintf("/%s/%s.pom", packageVersion, filename), pomContent, http.StatusCreated)

		pvs, err = packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeMaven)
		assert.NoError(t, err)
		assert.Len(t, pvs, 1)

		pd, err = packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
		assert.NoError(t, err)
		assert.IsType(t, &maven.Metadata{}, pd.Metadata)
		assert.Equal(t, packageDescription, pd.Metadata.(*maven.Metadata).Description)

		pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
		assert.NoError(t, err)
		assert.Len(t, pfs, 2)
		i := 0
		if strings.HasSuffix(pfs[1].Name, ".pom") {
			i = 1
		}
		assert.Equal(t, filename+".pom", pfs[i].Name)
		assert.True(t, pfs[i].IsLead)
	})

	t.Run("DownloadPOM", func(t *testing.T) {
		defer PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/%s/%s.pom", root, packageVersion, filename))
		req = AddBasicAuthHeader(req, user.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, []byte(pomContent), resp.Body.Bytes())

		pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeMaven)
		assert.NoError(t, err)
		assert.Len(t, pvs, 1)
		assert.Equal(t, int64(1), pvs[0].DownloadCount)
	})

	t.Run("DownloadChecksums", func(t *testing.T) {
		defer PrintCurrentTest(t)()

		req := NewRequest(t, "GET", fmt.Sprintf("%s/1.2.3/%s", root, filename))
		req = AddBasicAuthHeader(req, user.Name)
		MakeRequest(t, req, http.StatusNotFound)

		for key, checksum := range map[string]string{
			"md5":    "098f6bcd4621d373cade4e832627b4f6",
			"sha1":   "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3",
			"sha256": "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
			"sha512": "ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732c304cc5fa9ad8e6f57f50028a8ff",
		} {
			req := NewRequest(t, "GET", fmt.Sprintf("%s/%s/%s.%s", root, packageVersion, filename, key))
			req = AddBasicAuthHeader(req, user.Name)
			resp := MakeRequest(t, req, http.StatusOK)

			assert.Equal(t, checksum, resp.Body.String())
		}
	})

	t.Run("DownloadMetadata", func(t *testing.T) {
		defer PrintCurrentTest(t)()

		req := NewRequest(t, "GET", root+"/maven-metadata.xml")
		req = AddBasicAuthHeader(req, user.Name)
		resp := MakeRequest(t, req, http.StatusOK)

		expectedMetadata := `<?xml version="1.0" encoding="UTF-8"?>` + "\n<metadata><groupId>com.gitea</groupId><artifactId>test-project</artifactId><versioning><release>1.0.1</release><latest>1.0.1</latest><versions><version>1.0.1</version></versions></versioning></metadata>"
		assert.Equal(t, expectedMetadata, resp.Body.String())

		for key, checksum := range map[string]string{
			"md5":    "6bee0cebaaa686d658adf3e7e16371a0",
			"sha1":   "8696abce499fe84d9ea93e5492abe7147e195b6c",
			"sha256": "3f48322f81c4b2c3bb8649ae1e5c9801476162b520e1c2734ac06b2c06143208",
			"sha512": "cb075aa2e2ef1a83cdc14dd1e08c505b72d633399b39e73a21f00f0deecb39a3e2c79f157c1163f8a3854828750706e0dec3a0f5e4778e91f8ec2cf351a855f2",
		} {
			req := NewRequest(t, "GET", fmt.Sprintf("%s/maven-metadata.xml.%s", root, key))
			req = AddBasicAuthHeader(req, user.Name)
			resp := MakeRequest(t, req, http.StatusOK)

			assert.Equal(t, checksum, resp.Body.String())
		}
	})
}
