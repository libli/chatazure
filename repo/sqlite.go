package repo

import (
	"fmt"
	"time"

	"chatazure/model"
	"chatazure/tlog"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// SQLiteRepo SQLite 数据库
type SQLiteRepo struct {
	db   *gorm.DB
	User *UserRepo
}

// NewSQLiteRepo 创建 SQLite 数据库, 并初始化用户
func NewSQLiteRepo(dbname string, users []model.User) (*SQLiteRepo, error) {
	tlog.Info.Printf("init sqlite: %v", dbname)
	db, err := gorm.Open(sqlite.Open(dbname), &gorm.Config{})
	if err != nil {
		tlog.Error.Printf("failed to open db: %v", err)
		return nil, err
	}
	repo := SQLiteRepo{
		db:   db,
		User: newUserRepo(db),
	}
	if err := repo.AutoMigrate(); err != nil {
		tlog.Error.Printf("failed to auto migrate db: %v", err)
		return nil, err
	}
	if err := repo.initUsers(users); err != nil {
		tlog.Error.Printf("failed to init users: %v", err)
		return nil, err
	}
	return &repo, nil
}

// AutoMigrate 创建用户表
func (r *SQLiteRepo) AutoMigrate() error {
	return r.db.AutoMigrate(&model.User{})
}

// initUsers 初始化用户
func (r *SQLiteRepo) initUsers(users []model.User) error {
	for _, user := range users {
		user.CreateTime = time.Now()
		user.UpdateTime = time.Now()
		existingUser := r.User.GetByName(user.Username)
		if existingUser.Username == user.Username {
			// 用户已存在, 更新密码，更新 CanUseGPT4
			if existingUser.Password != user.Password || existingUser.CanUseGPT4 != user.CanUseGPT4 {
				existingUser.Password = user.Password
				existingUser.CanUseGPT4 = user.CanUseGPT4
				r.User.db.Save(&existingUser)
			}
		} else {
			// 用户不存在, 添加用户
			err := r.User.Add(&user)
			if err != nil {
				tlog.Error.Printf("failed to add user: %v", err)
				return fmt.Errorf("failed to add user %v: %w", user.Username, err)
			}
			tlog.Info.Printf("init user: %v", user.Username)
		}
	}
	return nil
}
