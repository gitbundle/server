// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package cmd

import (
	"net"
	"net/http"
	"net/http/fcgi"
	"strings"

	"github.com/gitbundle/modules/graceful"
	"github.com/gitbundle/modules/log"
	"github.com/gitbundle/modules/setting"
)

func runHTTP(network, listenAddr, name string, m http.Handler) error {
	return graceful.HTTPListenAndServe(network, listenAddr, name, m)
}

// NoHTTPRedirector tells our cleanup routine that we will not be using a fallback http redirector
func NoHTTPRedirector() {
	graceful.GetManager().InformCleanup()
}

// NoMainListener tells our cleanup routine that we will not be using a possibly provided listener
// for our main HTTP/HTTPS service
func NoMainListener() {
	graceful.GetManager().InformCleanup()
}

// NoInstallListener tells our cleanup routine that we will not be using a possibly provided listener
// for our install HTTP/HTTPS service
func NoInstallListener() {
	graceful.GetManager().InformCleanup()
}

func runFCGI(network, listenAddr, name string, m http.Handler) error {
	// This needs to handle stdin as fcgi point
	fcgiServer := graceful.NewServer(network, listenAddr, name)

	err := fcgiServer.ListenAndServe(func(listener net.Listener) error {
		return fcgi.Serve(listener, http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
			if setting.AppSubURL != "" {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, setting.AppSubURL)
			}
			m.ServeHTTP(resp, req)
		}))
	})
	if err != nil {
		log.Fatal("Failed to start FCGI main server: %v", err)
	}
	log.Info("FCGI Listener: %s Closed", listenAddr)
	return err
}
