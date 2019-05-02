package api

import (
	"encoding/json"
	"io"
)

// outputJSON is a helper method to write the given data as JSON to the given writer.
//
// It only logs an error if one occurs, rather than returning, since there is no point in trying
// to send a new status code back to the client once the body has started sending.
func outputJSON(c *Context, w io.Writer, data interface{}) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(data)
	if err != nil {
		c.Logger.WithError(err).Error("failed to encode result")
	}
}
