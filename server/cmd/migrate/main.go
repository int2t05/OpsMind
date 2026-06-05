// Package main 数据库迁移工具。
package main

import (
	"fmt"

	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/model"
)

func main() {
	cfg := config.DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "opsmind",
		Password: "opsmind123",
		DBName:   "opsmind_test",
		SSLMode:  "disable",
	}
	db, err := database.Init(cfg)
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&model.Role{}, &model.UserRole{}, &model.Menu{}, &model.RoleMenu{}); err != nil {
		panic(err)
	}
	fmt.Println("Migration completed")
}
