package repo

import (
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

// NewSQLiteRepo 创建 SQLite 数据库
func NewSQLiteRepo(dbname string, users []model.User) (*SQLiteRepo, error) {
	tlog.Info.Printf("Init sqlite: %v", dbname)
	db, err := gorm.Open(sqlite.Open(dbname), &gorm.Config{})
	if err != nil {
		tlog.Error.Printf("failed to open db: %v", err)
		return nil, err
	}
	repo := SQLiteRepo{
		db:   db,
		User: newUserRepo(db),
	}
	repo.AutoMigrate()
	for _, it := range users {
		it.CreateTime = time.Now()
		u := repo.User.GetByName(it.Username)
		if u.Username == it.Username && u.Password == it.Password {
			continue
		} else {
			if u.Username == it.Username {
				u.Password = it.Password
				repo.User.db.Save(u)
			} else {
				repo.User.Add(&it)
				if repo.User.GetByName(it.Username).Username == it.Username {
					tlog.Info.Printf("init user: %v", it.Username)
				}
			}
		}
	}
	return &repo, nil
}

// AutoMigrate 创建用户表
func (r *SQLiteRepo) AutoMigrate() error {
	return r.db.AutoMigrate(&model.User{})
}
