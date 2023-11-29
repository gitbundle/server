// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package asymkey

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/process"
	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/modules/util"
	"github.com/gitbundle/server/pkg/db"

	"golang.org/x/crypto/ssh"
)

// ___________.__                                         .__        __
// \_   _____/|__| ____    ____   ________________________|__| _____/  |_
//  |    __)  |  |/    \  / ___\_/ __ \_  __ \____ \_  __ \  |/    \   __\
//  |     \   |  |   |  \/ /_/  >  ___/|  | \/  |_> >  | \/  |   |  \  |
//  \___  /   |__|___|  /\___  / \___  >__|  |   __/|__|  |__|___|  /__|
//      \/            \//_____/      \/      |__|                 \/
//
// This file contains functions for fingerprinting SSH keys
//
// The database is used in checkKeyFingerprint however most of these functions probably belong in a module

// checkKeyFingerprint only checks if key fingerprint has been used as public key,
// it is OK to use same key as deploy key for multiple repositories/users.
func checkKeyFingerprint(ctx context.Context, fingerprint string) error {
	has, err := db.GetByBean(ctx, &PublicKey{
		Fingerprint: fingerprint,
	})
	if err != nil {
		return err
	} else if has {
		return ErrKeyAlreadyExist{0, fingerprint, ""}
	}
	return nil
}

func calcFingerprintSSHKeygen(publicKeyContent string) (string, error) {
	// Calculate fingerprint.
	tmpPath, err := writeTmpKeyFile(publicKeyContent)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := util.Remove(tmpPath); err != nil {
			log.Warn("Unable to remove temporary key file: %s: Error: %v", tmpPath, err)
		}
	}()
	stdout, stderr, err := process.GetManager().Exec("AddPublicKey", "ssh-keygen", "-lf", tmpPath)
	if err != nil {
		if strings.Contains(stderr, "is not a public key file") {
			return "", ErrKeyUnableVerify{stderr}
		}
		return "", fmt.Errorf("'ssh-keygen -lf %s' failed with error '%s': %s", tmpPath, err, stderr)
	} else if len(stdout) < 2 {
		return "", errors.New("not enough output for calculating fingerprint: " + stdout)
	}
	return strings.Split(stdout, " ")[1], nil
}

func calcFingerprintNative(publicKeyContent string) (string, error) {
	// Calculate fingerprint.
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicKeyContent))
	if err != nil {
		return "", err
	}
	return ssh.FingerprintSHA256(pk), nil
}

// CalcFingerprint calculate public key's fingerprint
func CalcFingerprint(publicKeyContent string) (string, error) {
	// Call the method based on configuration
	var (
		fnName, fp string
		err        error
	)
	if setting.SSH.StartBuiltinServer {
		fnName = "calcFingerprintNative"
		fp, err = calcFingerprintNative(publicKeyContent)
	} else {
		fnName = "calcFingerprintSSHKeygen"
		fp, err = calcFingerprintSSHKeygen(publicKeyContent)
	}
	if err != nil {
		if IsErrKeyUnableVerify(err) {
			log.Info("%s", publicKeyContent)
			return "", err
		}
		return "", fmt.Errorf("%s: %v", fnName, err)
	}
	return fp, nil
}
