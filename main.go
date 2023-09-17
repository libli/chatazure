package main

import (
	"fmt"
	"log"
	"net/http"

	"chatazure/config"
	"chatazure/handler"
	"chatazure/model"
	"chatazure/repo"
	"chatazure/tlog"
)

var version = "unknown"

func health(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

func main() {
	tlog.Info.Printf("chat azure (%s)", version)

	// 读取配置文件
	cfg, err := config.Get()
	if err != nil {
		tlog.Error.Fatalf("failed to get config: %v", err)
	}

	// 初始化配置文件中的用户，添加到数据库中
	var users []model.User
	for _, u := range cfg.Users {
		user := model.User{
			Username: u.Username,
			Password: u.Password,
		}
		users = append(users, user)
	}
	sqliteRepo, err := repo.NewSQLiteRepo(cfg.Web.DBName, users)
	if err != nil {
		tlog.Error.Fatalf("failed to init sqlite: %v", err)
	}

	proxy := handler.NewProxyHandler(sqliteRepo.User, cfg.Azure)

	http.HandleFunc("/", proxy.Proxy)
	http.HandleFunc("/v1/models", proxy.HandleModels)
	http.HandleFunc("/health", health)
	tlog.Info.Printf("web server at port: %s", cfg.Web.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Web.Port, nil))
}
