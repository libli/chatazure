package model

import (
	"time"
)

// User 用户表
type User struct {
	ID         uint      `gorm:"primaryKey"`
	Username   string    `gorm:"size:50;not null;uniqueIndex:uk_username;comment:用户唯一标识号"`
	Password   string    `gorm:"size:50;not null;uniqueIndex:uk_password;comment:用户登录凭证"`
	Count      int       `gorm:"not null;default:0;comment:用户使用 GPT 次数"`
	Status     uint8     `gorm:"not null;default:1;comment:用户状态, 1: 正常, 0: 禁用"`
	CreateTime time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;comment:创建时间"`
	UpdateTime time.Time `gorm:"not null;default:CURRENT_TIMESTAMP;comment:更新时间"`
}
