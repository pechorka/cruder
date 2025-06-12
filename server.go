package cruder

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/pechorka/cruder/pkg/httpio"
	"github.com/pechorka/cruder/pkg/swaggergen"
)

type Mux struct {
	sg  *swaggergen.Generator
	mux *http.ServeMux
}

func NewMux() *Mux {
	sg := swaggergen.NewGenerator()
	mux := http.NewServeMux()
	// TODO: allow to customize swagger path
	mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sg.Schema()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	return &Mux{
		sg:  sg,
		mux: mux,
	}
}

// pattern is GET /api/v1/users/{id}
func RegisterHandler[Req, Resp any](mux *Mux, pattern string, hndl func(ctx context.Context, req Req) (Resp, error)) error {
	method, path, ok := strings.Cut(pattern, " ")
	if !ok {
		return fmt.Errorf("invalid template: %s", pattern)
	}

	mux.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		var req Req
		if err := httpio.Unmarshal(r, &req); err != nil {
			// TODO: allow to customize error response
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		resp, err := hndl(r.Context(), req)
		if err != nil {
			// TODO: allow user to specify http status code
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			// TODO: allow to customize error response
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	var req Req
	var resp Resp
	mux.sg.RegisterHandler(swaggergen.HandlerInfo{
		Name:         pattern,
		Path:         path,
		Method:       method,
		RequestType:  reflect.TypeOf(req),
		ResponseType: reflect.TypeOf(resp),
	})
	return nil
}

func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	mux.mux.ServeHTTP(w, r)
}
