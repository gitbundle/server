// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package packages

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gitbundle/modules/setting"
	user_model "github.com/gitbundle/server/pkg/user"

	"github.com/golang-jwt/jwt/v4"
)

type packageClaims struct {
	jwt.RegisteredClaims
	UserID int64
}

func CreateAuthorizationToken(u *user_model.User) (string, error) {
	now := time.Now()

	claims := packageClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID: u.ID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(setting.SecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ParseAuthorizationToken(req *http.Request) (int64, error) {
	parts := strings.SplitN(req.Header.Get("Authorization"), " ", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("no token")
	}

	token, err := jwt.ParseWithClaims(parts[1], &packageClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(setting.SecretKey), nil
	})
	if err != nil {
		return 0, err
	}

	c, ok := token.Claims.(*packageClaims)
	if !token.Valid || !ok {
		return 0, fmt.Errorf("invalid token claim")
	}

	return c.UserID, nil
}
