package twitch

import (
	"context"
	"net/http"
	"strconv"

	"github.com/hortbot/hortbot/internal/pkg/jsonx"
	"golang.org/x/oauth2"
)

func statusToError(code int) error {
	if code >= 200 && code < 300 {
		return nil
	}

	switch code {
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrNotAuthorized
	}

	if code >= 500 {
		return ErrServerError
	}

	return ErrUnknown
}

func setToken(newToken **oauth2.Token) func(tok *oauth2.Token, err error) {
	return func(tok *oauth2.Token, err error) {
		if err == nil {
			*newToken = tok
		}
	}
}

func paginate[T any](ctx context.Context, cli *httpClient, url string, perPage int, limit int) (items []T, err error) {
	if perPage > 0 {
		url += "&first=" + strconv.Itoa(perPage)
	}
	cursor := ""

	doOne := func() error {
		url := url
		if cursor != "" {
			url += "&after=" + cursor
		}

		resp, err := cli.Get(ctx, url)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := statusToError(resp.StatusCode); err != nil {
			return err
		}

		var v struct {
			Data       []T `json:"data"`
			Pagination struct {
				Cursor string `json:"cursor"`
			} `json:"pagination"`
		}

		if err := jsonx.DecodeSingle(resp.Body, &v); err != nil {
			return ErrServerError
		}

		items = append(items, v.Data...)
		cursor = v.Pagination.Cursor

		return nil
	}

	prevLen := 0

	for {
		if err := doOne(); err != nil {
			return nil, err
		}

		if cursor == "" {
			break
		}

		// Sanity checks.
		if len(items) == prevLen || len(items) >= limit {
			break
		}

		prevLen = len(items)
	}

	return items, nil
}

func fetchList[T any](ctx context.Context, cli *httpClient, url string) ([]T, error) {
	resp, err := cli.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := &struct {
		Data []T `json:"data"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	if len(body.Data) == 0 {
		return nil, ErrNotFound
	}

	return body.Data, nil
}

func postAndDecodeList[T any](ctx context.Context, cli *httpClient, url string, v any) ([]T, error) {
	resp, err := cli.Post(ctx, url, v)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := statusToError(resp.StatusCode); err != nil {
		return nil, err
	}

	body := &struct {
		Data []T `json:"data"`
	}{}

	if err := jsonx.DecodeSingle(resp.Body, body); err != nil {
		return nil, ErrServerError
	}

	if len(body.Data) == 0 {
		return nil, ErrNotFound
	}

	return body.Data, nil
}

func fetchFirstFromList[T any](ctx context.Context, cli *httpClient, url string) (T, error) {
	list, err := fetchList[T](ctx, cli, url)
	if err != nil {
		var zero T
		return zero, err
	}
	return list[0], nil
}

func postAndDecodeFirstFromList[T any](ctx context.Context, cli *httpClient, url string, v any) (T, error) {
	list, err := postAndDecodeList[T](ctx, cli, url, v)
	if err != nil {
		var zero T
		return zero, err
	}
	return list[0], nil
}
