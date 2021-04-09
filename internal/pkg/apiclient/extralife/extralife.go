// Package extralife provides an Extra-Life API client.
package extralife

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
	"github.com/hortbot/hortbot/internal/pkg/jsonx"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . API

// API represents the supported API functions. It's defined for fake generation.
type API interface {
	GetDonationAmount(ctx context.Context, participantID int) (float64, error)
}

// ExtraLife is an Extra-Life API client.
type ExtraLife struct {
	cli httpx.Client
}

var _ API = &ExtraLife{}

// New creates a new Extra-Life API client.
func New(opts ...Option) *ExtraLife {
	e := &ExtraLife{}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Option controls client functionality.
type Option func(*ExtraLife)

// HTTPClient sets the HTTP client used when making requests to the Extra-Life
// API. If given a nil client (or not set), the client will use the default
// HTTP client in net/http.
func HTTPClient(cli *http.Client) Option {
	return func(e *ExtraLife) {
		e.cli.Client = cli
	}
}

// GetDonationAmount gets the current donation total for the given Extra-Life
// participant.
func (e *ExtraLife) GetDonationAmount(ctx context.Context, participantID int) (float64, error) {
	url := fmt.Sprintf("https://www.extra-life.org/api/participants/%d", participantID)

	resp, err := e.cli.Get(ctx, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if !apiclient.IsOK(resp.StatusCode) {
		return 0, &apiclient.Error{API: "extralife", StatusCode: resp.StatusCode}
	}

	var v struct {
		SumDonations float64 `json:"sumDonations"`
	}

	if err := jsonx.DecodeSingle(resp.Body, &v); err != nil {
		return 0, &apiclient.Error{API: "extralife", Err: err}
	}

	return v.SumDonations, nil
}
