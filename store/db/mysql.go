package db

import (
	"github.com/jmoiron/sqlx"
	"life-online/config"
	"time"
)

var MainDB *sqlx.DB

func InitDB() {
	MainDB = sqlx.MustConnect("mysql", config.GetMainDBUrl())
	// 设置连接池配置
	MainDB.SetMaxOpenConns(100)                // 最大打开连接数
	MainDB.SetMaxIdleConns(10)                 // 最大空闲连接数
	MainDB.SetConnMaxLifetime(time.Hour)       // 连接最大存活时间
	MainDB.SetConnMaxIdleTime(time.Minute * 5) // 空闲连接最大存活时间
}
