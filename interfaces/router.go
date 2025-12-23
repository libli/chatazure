package interfaces

import (
	"net/http"

	"chatazure/config"
	"chatazure/interfaces/handler"
	"chatazure/repo"
)

func SetupRouter(repo *repo.SQLiteRepo, config *config.Config) *http.ServeMux {
	mux := http.NewServeMux()

	health := handler.NewHealthHandler()
	mux.HandleFunc("/healthz", health.Healthz)

	proxy := handler.NewProxyHandler(repo.User, config.Azure)
	mux.HandleFunc("/v1/models", proxy.HandleModels)
	mux.HandleFunc("/v1/chat/completions", proxy.HandleChat)
	mux.HandleFunc("/v1/responses", proxy.HandleResponses)
	// Azure native passthrough (whitelist inside handler).
	mux.HandleFunc("/openai/", proxy.HandleAzureResponsesPassthrough)

	return mux
}
