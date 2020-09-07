package web

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/zikaeroh/ctxlog"
	"go.uber.org/zap"
)

func (a *App) routeDebug(r chi.Router) {
	r.Use(middleware.NoCache)
	r.Get("/request", dumpRequest)
}

func dumpRequest(w http.ResponseWriter, r *http.Request) {
	b, err := httputil.DumpRequest(r, true)
	if err != nil {
		ctxlog.Error(r.Context(), "error dumping request", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", b)
}
