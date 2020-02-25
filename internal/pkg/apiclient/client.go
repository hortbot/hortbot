// Package apiclient provides helpers for hortbot's API clients.
package apiclient

import (
	"fmt"
	"net/http"
)

// Error is an HTTP error response type, returned by an API.
type Error struct {
	API        string
	StatusCode int
	Err        error
}

func (e *Error) Error() string {
	api := e.API
	if api == "" {
		api = "client"
	}

	if e.Err == nil {
		return fmt.Sprintf("%s: status code %d", api, e.StatusCode)
	}

	return fmt.Sprintf("%s: %s", api, e.Err)
}

// IsNotFound returns true if the error is a not found.
func (e *Error) IsNotFound() bool {
	return e.StatusCode == http.StatusNotFound
}

// IsServerError returns true if the error is a server error.
func (e *Error) IsServerError() bool {
	return e.StatusCode >= 500
}

// IsNotPermitted returns true if the client was not permitted to access a resource.
func (e *Error) IsNotPermitted() bool {
	return e.StatusCode == http.StatusUnauthorized || e.StatusCode == http.StatusForbidden
}

// IsOK return true if the provided status code is okay (2xx).
func IsOK(code int) bool {
	return code >= 200 && code < 300
}
