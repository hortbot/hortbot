// Package apiclient provides helpers for hortbot's API clients.
package apiclient

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/carlmjohnson/requests"
)

// Error is an HTTP error response type, returned by an API.
type Error struct {
	API        string
	StatusCode int
	Err        error
	secrets    []string
}

func (e *Error) Error() string {
	api := e.API
	if api == "" {
		api = "client"
	}

	if e.Err == nil {
		return fmt.Sprintf("%s: unexpected status: %d", api, e.StatusCode)
	}

	s := fmt.Sprintf("%s: %s", api, e.Err.Error())
	for i, secret := range e.secrets {
		replacement := fmt.Sprintf("REDACTED%d", i)
		s = strings.ReplaceAll(s, secret, replacement)
		s = strings.ReplaceAll(s, url.QueryEscape(secret), replacement)
	}

	return s
}

func (e *Error) Unwrap() error {
	return e.Err
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

func WrapRequestErr(apiName string, err error, secrets []string) error {
	if err == nil {
		return nil
	}
	if re, ok := errors.AsType[*requests.ResponseError](err); ok {
		return &Error{API: apiName, Err: err, StatusCode: re.StatusCode, secrets: secrets}
	}
	return &Error{API: apiName, Err: err, secrets: secrets}
}

func NewStatusError(apiName string, code int) error {
	return &Error{API: apiName, StatusCode: code}
}

func NewNonStatusError(apiName string, err error) error {
	return &Error{API: apiName, Err: err}
}

func AsError(err error) (*Error, bool) {
	return errors.AsType[*Error](err)
}
