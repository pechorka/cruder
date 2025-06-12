package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/pechorka/cruder"
)

func main() {
	if err := run(); err != nil {
		slog.Error("failed to run app", "error", err)
		os.Exit(1)
	}
}

func run() error {
	mux := cruder.NewMux()
	cruder.RegisterHandler(mux, "POST /echo", echoHandler)

	return http.ListenAndServe(":8080", mux)
}

type request struct {
	Name string `json:"name"`
}

type response struct {
	Name string `json:"name"`
}

func echoHandler(ctx context.Context, req request) (response, error) {
	return response{
		Name: req.Name,
	}, nil
}
