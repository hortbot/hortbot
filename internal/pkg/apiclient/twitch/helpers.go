package twitch

import (
	"context"
	"net/http"
	"strconv"

	"github.com/carlmjohnson/requests"
	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"golang.org/x/oauth2"
)

func setToken(newToken **oauth2.Token) func(tok *oauth2.Token, err error) {
	return func(tok *oauth2.Token, err error) {
		if err == nil {
			*newToken = tok
		}
	}
}

func paginate[T any](ctx context.Context, req *requests.Builder, perPage int, limit int) (items []T, err error) {
	if perPage > 0 {
		req.Param("first", strconv.Itoa(perPage))
	}
	cursor := ""

	doOne := func() error {
		req := req.Clone()

		if cursor != "" {
			req.Param("after", cursor)
		}

		var v struct {
			Data       []T `json:"data"`
			Pagination struct {
				Cursor string `json:"cursor"`
			} `json:"pagination"`
		}

		if err := req.Handle(httpx.ToJSON(&v)).Fetch(ctx); err != nil { //nolint:bodyclose
			return apiclient.WrapRequestErr("twitch", err, nil)
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

func fetchList[T any](ctx context.Context, req *requests.Builder) ([]T, error) {
	body := &struct {
		Data []T `json:"data"`
	}{}

	if err := req.Handle(httpx.ToJSON(body)).Fetch(ctx); err != nil { //nolint:bodyclose
		return nil, apiclient.WrapRequestErr("twitch", err, nil)
	}

	if len(body.Data) == 0 {
		return nil, apiclient.NewStatusError("twitch", http.StatusNotFound)
	}

	return body.Data, nil
}

func fetchFirstFromList[T any](ctx context.Context, req *requests.Builder) (T, error) {
	list, err := fetchList[T](ctx, req)
	if err != nil {
		var zero T
		return zero, err
	}
	return list[0], nil
}
