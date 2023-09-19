package repo

import (
	"time"

	"chatazure/model"
	"chatazure/tlog"

	"gorm.io/gorm"
)

// UserRepo 用户表DAO
type UserRepo struct {
	db *gorm.DB
}

func newUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Add 添加用户
func (u *UserRepo) Add(user *model.User) error {
	tlog.Info.Printf("Add user: %v", user)
	result := u.db.Create(user)
	return result.Error
}

// GetByPassword 根据 password 获取没被禁用的用户
func (u *UserRepo) GetByPassword(password string) *model.User {
	var user model.User
	u.db.Where("password = ? AND status = ?", password, 1).First(&user)
	return &user
}

// GetByName 根据 username 获取用户
func (u *UserRepo) GetByName(name string) *model.User {
	var user model.User
	u.db.Where("username = ?", name).First(&user)
	return &user
}

// UpdateCount 更新用户使用次数
func (u *UserRepo) UpdateCount(user *model.User) {
	user.Count++
	user.UpdateTime = time.Now()
	u.db.Save(&user)
}
