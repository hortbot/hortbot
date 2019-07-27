package xkcd

import (
	"errors"

	"github.com/rkoesters/xkcd"
)

var ErrNotFound = errors.New("xkcd: not found")

type Comic struct {
	Title string
	Img   string
	Alt   string
}

//go:generate gobin -m -run github.com/maxbrunsfeld/counterfeiter/v6 -generate

//counterfeiter:generate . API
type API interface {
	GetComic(id int) (*Comic, error)
}

// TODO: Fork the XKCD library to allow for a custom HTTP client.

type XKCD struct{}

var _ API = &XKCD{}

func New() *XKCD {
	return &XKCD{}
}

func (*XKCD) GetComic(id int) (*Comic, error) {
	c, err := xkcd.Get(id)
	if err != nil {
		if err == xkcd.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &Comic{
		Title: c.SafeTitle,
		Img:   c.Img,
		Alt:   c.Alt,
	}, nil
}
