// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package middleware

import (
	"net/http"

	"github.com/gitbundle/modules/setting"
	"github.com/gitbundle/modules/translation"
	"github.com/gitbundle/modules/translation/i18n"

	"golang.org/x/text/language"
)

// Locale handle locale
func Locale(resp http.ResponseWriter, req *http.Request) translation.Locale {
	// 1. Check URL arguments.
	lang := req.URL.Query().Get("lang")
	changeLang := lang != ""

	// 2. Get language information from cookies.
	if len(lang) == 0 {
		ck, _ := req.Cookie("lang")
		if ck != nil {
			lang = ck.Value
		}
	}

	// Check again in case someone changes the supported language list.
	if lang != "" && !i18n.DefaultLocales.HasLang(lang) {
		lang = ""
		changeLang = false
	}

	// 3. Get language information from 'Accept-Language'.
	// The first element in the list is chosen to be the default language automatically.
	if len(lang) == 0 {
		tags, _, _ := language.ParseAcceptLanguage(req.Header.Get("Accept-Language"))
		tag := translation.Match(tags...)
		lang = tag.String()
	}

	if changeLang {
		SetLocaleCookie(resp, lang, 1<<31-1)
	}

	return translation.NewLocale(lang)
}

// SetLocaleCookie convenience function to set the locale cookie consistently
func SetLocaleCookie(resp http.ResponseWriter, lang string, expiry int) {
	SetCookie(resp, "lang", lang, expiry,
		setting.AppSubURL,
		setting.SessionConfig.Domain,
		setting.SessionConfig.Secure,
		true,
		SameSite(setting.SessionConfig.SameSite))
}

// DeleteLocaleCookie convenience function to delete the locale cookie consistently
// Setting the lang cookie will trigger the middleware to reset the language ot previous state.
func DeleteLocaleCookie(resp http.ResponseWriter) {
	SetCookie(resp, "lang", "",
		-1,
		setting.AppSubURL,
		setting.SessionConfig.Domain,
		setting.SessionConfig.Secure,
		true,
		SameSite(setting.SessionConfig.SameSite))
}
