package grpc

import (
	"net/http"

	"connectrpc.com/connect"
	authv3connect "github.com/astro-web3/oauth2-token-exchange/pb/gen/go/envoy/service/auth/v3/authv3connect"
)

func NewRouter(handler authv3connect.AuthorizationHandler) http.Handler {
	mux := http.NewServeMux()

	path, httpHandler := authv3connect.NewAuthorizationHandler(handler, connect.WithInterceptors(
		recoveryInterceptor(),
		loggingInterceptor(),
	))
	mux.Handle(path, httpHandler)

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return mux
}
