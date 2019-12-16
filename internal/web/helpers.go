package web

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/hortbot/hortbot/internal/pkg/ctxlog"
	"go.uber.org/zap"
)

func notAuthorized(w http.ResponseWriter, header bool) {
	if header {
		w.Header().Add("WWW-Authenticate", `Basic realm="hortbot"`)
	}
	httpError(w, http.StatusUnauthorized)
}

func httpError(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

func dumpRequest(w http.ResponseWriter, r *http.Request) {
	b, err := httputil.DumpRequest(r, true)
	if err != nil {
		ctxlog.Error(r.Context(), "error dumping request", zap.Error(err))
		httpError(w, http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "%s", b)
}
