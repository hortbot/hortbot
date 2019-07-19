package extralife

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

var ErrNotFound = errors.New("extralife: not found")

//counterfeiter:generate . API
type API interface {
	GetDonationAmount(participantID int) (float64, error)
}

type ExtraLife struct{}

var _ API = &ExtraLife{}

func New() *ExtraLife {
	return &ExtraLife{}
}

func (*ExtraLife) GetDonationAmount(participantID int) (float64, error) {
	url := fmt.Sprintf("https://www.extra-life.org/api/participants/%d", participantID)

	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	log.Println(resp.StatusCode, resp.Status)

	if resp.StatusCode < 200 && resp.StatusCode >= 300 {
		return 0, ErrNotFound
	}

	var v struct {
		SumDonations float64 `json:"sumDonations"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return 0, err
	}

	return v.SumDonations, nil
}
