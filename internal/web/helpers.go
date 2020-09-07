package web

import (
	"net/http"

	"github.com/hortbot/hortbot/internal/web/templates"
	"golang.org/x/net/publicsuffix"
)

type errorPage struct {
	message string
	image   string
}

var errorPages = map[int]errorPage{
	http.StatusNotFound: {
		message: "Sorry, this page doesn't exist.",
		image:   "/static/img/notfound_800.png",
	},
	http.StatusUnauthorized: {
		message: "Nice try.",
		image:   "/static/img/forbidden_800.png",
	},
	http.StatusForbidden: {
		message: "Nice try.",
		image:   "/static/img/forbidden_800.png",
	},
	http.StatusBadRequest: {
		message: "Nice try.",
		image:   "/static/img/forbidden_800.png",
	},
}

func (a *App) httpError(w http.ResponseWriter, r *http.Request, code int) {
	e, ok := errorPages[code]
	if !ok {
		http.Error(w, http.StatusText(code), code)
		return
	}

	page := &templates.ErrorPage{
		BasePage: a.basePage(r),
		Message:  e.message,
		Image:    e.image,
	}

	w.WriteHeader(code)
	templates.WritePageTemplate(w, page)
}

func (a *App) notAuthorized(w http.ResponseWriter, r *http.Request, header bool) {
	if header {
		w.Header().Add("WWW-Authenticate", `Basic realm="hortbot"`)
	}
	a.httpError(w, r, http.StatusUnauthorized)
}

func normalizeHost(host string) string {
	if host == "" {
		return host
	}

	if tld, err := publicsuffix.EffectiveTLDPlusOne(host); err == nil {
		return tld
	}

	return host
}
