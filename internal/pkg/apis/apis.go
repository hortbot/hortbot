// Package apis provides helpers for hortbot's API clients.
package apis

import (
	"fmt"
	"net/http"
)

// Error is an HTTP error response type, returned by an API.
type Error struct {
	API        string
	StatusCode int
	Message    string
}

func (e *Error) Error() string {
	api := e.API
	if api == "" {
		api = "apis"
	}

	if e.Message == "" {
		return fmt.Sprintf("%s: status code %d", api, e.StatusCode)
	}

	return fmt.Sprintf("%s: status code %d: %s", api, e.StatusCode, e.Message)
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
