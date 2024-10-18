// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type contextHandlerFunc func(c *Context, w http.ResponseWriter, r *http.Request)

type contextHandler struct {
	context     *Context
	handler     contextHandlerFunc
	handlerName string
}

func (h contextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ww := NewWrappedWriter(w)
	context := h.context.Clone()
	context.RequestID = model.NewID()

	userID := ""
	if r.Context().Value(ContextKeyUserID{}) != nil {
		userID = r.Context().Value(ContextKeyUserID{}).(string)
	}

	context.Logger = context.Logger.WithFields(log.Fields{
		"handler": h.handlerName,
		"method":  r.Method,
		"path":    r.URL.Path,
		"request": context.RequestID,
		"user_id": userID,
	})

	context.Logger.Debug("Handling Request")

	h.handler(context, ww, r)

	elapsed := float64(time.Since(start)) / float64(time.Second)
	context.Metrics.ObserveAPIEndpointDuration(h.handlerName, r.Method, ww.StatusCode(), elapsed)
	context.Metrics.IncrementAPIRequest()
}

func newContextHandler(context *Context, handler contextHandlerFunc) *contextHandler {
	// Obtain the handler function name to be used for API metrics.
	splitFuncName := strings.Split((runtime.FuncForPC(reflect.ValueOf(handler).Pointer()).Name()), ".")

	return &contextHandler{
		context:     context,
		handler:     handler,
		handlerName: splitFuncName[len(splitFuncName)-1],
	}
}
