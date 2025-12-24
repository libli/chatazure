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
	// 统一代理：/v1/* -> /openai/v1/*，/openai/* 直通
	mux.HandleFunc("/v1/", proxy.HandleProxy)
	mux.HandleFunc("/openai/", proxy.HandleProxy)

	return mux
}
