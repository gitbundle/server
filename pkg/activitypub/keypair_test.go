// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package activitypub

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeygen(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	assert.NoError(t, err)

	assert.NotEmpty(t, priv)
	assert.NotEmpty(t, pub)

	assert.Regexp(t, regexp.MustCompile("^-----BEGIN RSA PRIVATE KEY-----.*"), priv)
	assert.Regexp(t, regexp.MustCompile("^-----BEGIN PUBLIC KEY-----.*"), pub)
}

func TestSignUsingKeys(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	assert.NoError(t, err)

	privPem, _ := pem.Decode([]byte(priv))
	if privPem == nil || privPem.Type != "RSA PRIVATE KEY" {
		t.Fatal("key is wrong type")
	}

	privParsed, err := x509.ParsePKCS1PrivateKey(privPem.Bytes)
	assert.NoError(t, err)

	pubPem, _ := pem.Decode([]byte(pub))
	if pubPem == nil || pubPem.Type != "PUBLIC KEY" {
		t.Fatal("key failed to decode")
	}

	pubParsed, err := x509.ParsePKIXPublicKey(pubPem.Bytes)
	assert.NoError(t, err)

	// Sign
	msg := "activity pub is great!"
	h := sha256.New()
	h.Write([]byte(msg))
	d := h.Sum(nil)
	sig, err := rsa.SignPKCS1v15(rand.Reader, privParsed, crypto.SHA256, d)
	assert.NoError(t, err)

	// Verify
	err = rsa.VerifyPKCS1v15(pubParsed.(*rsa.PublicKey), crypto.SHA256, d, sig)
	assert.NoError(t, err)
}
