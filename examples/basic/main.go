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
	cruder.RegisterHandler(mux, "GET /echo", getEchoHandler)

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

type getEchoRequest struct {
	Name fullName `query:"name"`
}

type fullName struct {
	First  string  `query:"first"`
	Last   string  `query:"last"`
	Middle *string `query:"middle"`
}

type getEchoResponse struct {
	Name string `json:"name"`
}

// expected request: GET /echo?name.first=John&name.last=Doe should return "John Doe"
func getEchoHandler(ctx context.Context, req getEchoRequest) (getEchoResponse, error) {
	name := req.Name.First + " " + req.Name.Last
	if req.Name.Middle != nil {
		name += " " + *req.Name.Middle
	}
	return getEchoResponse{
		Name: name,
	}, nil
}
