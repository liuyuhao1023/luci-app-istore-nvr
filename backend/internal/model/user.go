package model

import "time"

// UserRole 用户角色
type UserRole string

const (
	RoleAdmin   UserRole = "admin"  // 系统管理员
	RoleViewer  UserRole = "viewer" // 普通查看者 (仅能查看已授权画面，不可更改配置)
)

// User 系统用户模型
type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"size:64;uniqueIndex;not null"`
	Password  string    `json:"-" gorm:"size:128;not null"` // 加密散列
	Role      UserRole  `json:"role" gorm:"size:32;default:'admin'"`
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
