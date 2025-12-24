package main

import (
	"net/http"

	"chatazure/config"
	"chatazure/interfaces"
	"chatazure/model"
	"chatazure/repo"
	"chatazure/tlog"
)

var version = "unknown"

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

	mux := interfaces.SetupRouter(sqliteRepo, cfg)
	port := ":" + cfg.Web.Port
	tlog.Info.Printf("Starting server on %s", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		tlog.Error.Fatalf("Server failed: %v", err)
	}
}
