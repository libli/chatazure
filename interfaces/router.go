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
	// 统一代理：所有路径进入代理层处理
	mux.HandleFunc("/", proxy.HandleProxy)

	return mux
}
