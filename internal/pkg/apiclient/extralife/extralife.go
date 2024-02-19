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

//go:generate go run github.com/matryer/moq -fmt goimports -out extralifemocks/mocks.go -pkg extralifemocks . API

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
func New(cli *http.Client) *ExtraLife {
	return &ExtraLife{
		cli: httpx.NewClient(cli, "extralife", false),
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
