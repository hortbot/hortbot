package extralife

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"golang.org/x/net/context/ctxhttp"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

var (
	ErrNotFound    = errors.New("extralife: not found")
	ErrServerError = errors.New("extralife: server error")
)

//counterfeiter:generate . API
type API interface {
	GetDonationAmount(ctx context.Context, participantID int) (float64, error)
}

type ExtraLife struct {
	cli *http.Client
}

var _ API = &ExtraLife{}

func New(opts ...Option) *ExtraLife {
	e := &ExtraLife{}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

type Option func(*ExtraLife)

func HTTPClient(cli *http.Client) Option {
	return func(e *ExtraLife) {
		e.cli = cli
	}
}

func (e *ExtraLife) GetDonationAmount(ctx context.Context, participantID int) (float64, error) {
	url := fmt.Sprintf("https://www.extra-life.org/api/participants/%d", participantID)

	resp, err := ctxhttp.Get(ctx, e.cli, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return 0, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return 0, ErrServerError
	}

	var v struct {
		SumDonations float64 `json:"sumDonations"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return 0, ErrServerError
	}

	return v.SumDonations, nil
}
