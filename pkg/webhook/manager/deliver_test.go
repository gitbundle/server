// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package webhook_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/gitbundle/modules/setting"

	"github.com/stretchr/testify/assert"
)

func TestWebhookProxy(t *testing.T) {
	setting.Webhook.ProxyURL = "http://localhost:8080"
	setting.Webhook.ProxyURLFixed, _ = url.Parse(setting.Webhook.ProxyURL)
	setting.Webhook.ProxyHosts = []string{"*.discordapp.com", "discordapp.com"}

	kases := map[string]string{
		"https://discordapp.com/api/webhooks/xxxxxxxxx/xxxxxxxxxxxxxxxxxxx": "http://localhost:8080",
		"http://s.discordapp.com/assets/xxxxxx":                             "http://localhost:8080",
		"http://github.com/a/b":                                             "",
	}

	for reqURL, proxyURL := range kases {
		req, err := http.NewRequest("POST", reqURL, nil)
		assert.NoError(t, err)

		u, err := webhookProxy()(req)
		assert.NoError(t, err)
		if proxyURL == "" {
			assert.Nil(t, u)
		} else {
			assert.EqualValues(t, proxyURL, u.String())
		}
	}
}
