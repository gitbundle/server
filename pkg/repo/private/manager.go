// Copyright 2023 The GitBundle Inc. All rights reserved.
// Copyright 2016 The Gitea Authors. All rights reserved.
// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a CC BY-NC 4.0
// license that can be found in the LICENSE file.

package private

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gitbundle/modules/json"
	"github.com/gitbundle/modules/setting"
)

// Shutdown calls the internal shutdown function
func Shutdown(ctx context.Context) (int, string) {
	reqURL := setting.LocalURL + "api/internal/manager/shutdown"

	req := newInternalRequest(ctx, reqURL, "POST")
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, "Shutting down"
}

// Restart calls the internal restart function
func Restart(ctx context.Context) (int, string) {
	reqURL := setting.LocalURL + "api/internal/manager/restart"

	req := newInternalRequest(ctx, reqURL, "POST")
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, "Restarting"
}

// FlushOptions represents the options for the flush call
type FlushOptions struct {
	Timeout     time.Duration
	NonBlocking bool
}

// FlushQueues calls the internal flush-queues function
func FlushQueues(ctx context.Context, timeout time.Duration, nonBlocking bool) (int, string) {
	reqURL := setting.LocalURL + "api/internal/manager/flush-queues"

	req := newInternalRequest(ctx, reqURL, "POST")
	if timeout > 0 {
		req.SetTimeout(timeout+10*time.Second, timeout+10*time.Second)
	}
	req = req.Header("Content-Type", "application/json")
	jsonBytes, _ := json.Marshal(FlushOptions{
		Timeout:     timeout,
		NonBlocking: nonBlocking,
	})
	req.Body(jsonBytes)
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, "Flushed"
}

// PauseLogging pauses logging
func PauseLogging(ctx context.Context) (int, string) {
	reqURL := setting.LocalURL + "api/internal/manager/pause-logging"

	req := newInternalRequest(ctx, reqURL, "POST")
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, "Logging Paused"
}

// ResumeLogging resumes logging
func ResumeLogging(ctx context.Context) (int, string) {
	reqURL := setting.LocalURL + "api/internal/manager/resume-logging"

	req := newInternalRequest(ctx, reqURL, "POST")
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, "Logging Restarted"
}

// ReleaseReopenLogging releases and reopens logging files
func ReleaseReopenLogging(ctx context.Context) (int, string) {
	reqURL := setting.LocalURL + "api/internal/manager/release-and-reopen-logging"

	req := newInternalRequest(ctx, reqURL, "POST")
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, "Logging Restarted"
}

// LoggerOptions represents the options for the add logger call
type LoggerOptions struct {
	Group  string
	Name   string
	Mode   string
	Config map[string]interface{}
}

// AddLogger adds a logger
func AddLogger(ctx context.Context, group, name, mode string, config map[string]interface{}) (int, string) {
	reqURL := setting.LocalURL + "api/internal/manager/add-logger"

	req := newInternalRequest(ctx, reqURL, "POST")
	req = req.Header("Content-Type", "application/json")
	jsonBytes, _ := json.Marshal(LoggerOptions{
		Group:  group,
		Name:   name,
		Mode:   mode,
		Config: config,
	})
	req.Body(jsonBytes)
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, "Added"
}

// RemoveLogger removes a logger
func RemoveLogger(ctx context.Context, group, name string) (int, string) {
	reqURL := setting.LocalURL + fmt.Sprintf("api/internal/manager/remove-logger/%s/%s", url.PathEscape(group), url.PathEscape(name))

	req := newInternalRequest(ctx, reqURL, "POST")
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	return http.StatusOK, "Removed"
}

// Processes return the current processes from this gitea instance
func Processes(ctx context.Context, out io.Writer, flat, noSystem, stacktraces, json bool, cancel string) (int, string) {
	reqURL := setting.LocalURL + fmt.Sprintf("api/internal/manager/processes?flat=%t&no-system=%t&stacktraces=%t&json=%t&cancel-pid=%s", flat, noSystem, stacktraces, json, url.QueryEscape(cancel))

	req := newInternalRequest(ctx, reqURL, "GET")
	resp, err := req.Response()
	if err != nil {
		return http.StatusInternalServerError, fmt.Sprintf("Unable to contact gitea: %v", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, decodeJSONError(resp).Err
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return http.StatusInternalServerError, err.Error()
	}
	return http.StatusOK, ""
}
