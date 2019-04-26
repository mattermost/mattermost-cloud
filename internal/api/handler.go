package api

import "net/http"

type contextHandlerFunc func(c *Context, w http.ResponseWriter, r *http.Request)

type contextHandler struct {
	context *Context
	handler contextHandlerFunc
}

func (h contextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	context := h.context.Clone()
	context.Logger = context.Logger.WithField("path", r.URL.Path)

	h.handler(context, w, r)
}

func newContextHandler(context *Context, handler contextHandlerFunc) *contextHandler {
	return &contextHandler{
		context: context,
		handler: handler,
	}
}
