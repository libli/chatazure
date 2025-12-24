package logic

import (
	"chatazure/repo"
)

// UserLogic 用户逻辑
type UserLogic struct {
	user *repo.UserRepo
}

// NewUserLogic 创建用户逻辑
func NewUserLogic(user *repo.UserRepo) *UserLogic {
	return &UserLogic{
		user: user,
	}
}

// Auth 验证用户
func (u *UserLogic) Auth(password string) (bool, string) {
	if password == "" {
		return false, ""
	}
	user := u.user.GetByPassword(password)
	if user.Password == password {
		return true, user.Username
	}
	return false, ""
}

// UpdateCount 更新用户使用次数
func (u *UserLogic) UpdateCount(username string) {
	user := u.user.GetByName(username)
	u.user.UpdateCount(user)
}
