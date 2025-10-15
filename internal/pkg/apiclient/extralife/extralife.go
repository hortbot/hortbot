// Package extralife provides an Extra-Life API client.
package extralife

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hortbot/hortbot/internal/pkg/apiclient"
	"github.com/hortbot/hortbot/internal/pkg/httpx"
)

//go:generate go tool github.com/matryer/moq -fmt goimports -out extralifemocks/mocks.go -pkg extralifemocks . API

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
		cli: httpx.NewClient(cli, "extralife"),
	}
}

// GetDonationAmount gets the current donation total for the given Extra-Life
// participant.
func (e *ExtraLife) GetDonationAmount(ctx context.Context, participantID int) (float64, error) {
	url := fmt.Sprintf("https://www.extra-life.org/api/participants/%d", participantID)

	var v struct {
		SumDonations float64 `json:"sumDonations"`
	}

	if err := e.cli.NewRequestToJSON(url, &v).Fetch(ctx); err != nil {
		return 0, apiclient.WrapRequestErr("extralife", err, nil)
	}

	return v.SumDonations, nil
}
