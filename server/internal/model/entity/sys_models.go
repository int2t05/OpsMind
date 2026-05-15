// 系统类 GORM 实体：用户、角色、权限、日志、配置、附件
package entity

import (
	"time"

	"gorm.io/gorm"
)

// ---------- 用户与权限 ----------

type SysUser struct {
	ID           int64          `gorm:"primaryKey" json:"id"`
	Username     string         `gorm:"size:64;not null;uniqueIndex" json:"username"`
	PasswordHash string         `gorm:"size:255;not null" json:"-"`
	RealName     string         `gorm:"size:64;not null" json:"real_name"`
	Phone        *string        `gorm:"size:32" json:"phone"`
	Email        *string        `gorm:"size:128" json:"email"`
	Status       int16          `gorm:"not null;default:1" json:"status"` // 1正常 2冻结
	LastLoginAt  *time.Time     `json:"last_login_at"`
	LastLoginIP  *string        `gorm:"size:64" json:"last_login_ip"`
	Remark       *string        `gorm:"size:255" json:"remark"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SysUser) TableName() string { return "sys_user" }

type SysRole struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Code      string    `gorm:"size:64;not null;uniqueIndex" json:"code"`
	Name      string    `gorm:"size:64;not null" json:"name"`
	Status    int16     `gorm:"not null;default:1" json:"status"` // 1启用 2停用
	Remark    *string   `gorm:"size:255" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SysRole) TableName() string { return "sys_role" }

type SysPermission struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	ParentID  *int64    `json:"parent_id"`
	Type      int16     `gorm:"not null" json:"type"` // 1菜单 2按钮 3接口
	Name      string    `gorm:"size:64;not null" json:"name"`
	Code      string    `gorm:"size:128;not null;uniqueIndex" json:"code"`
	Path      *string   `gorm:"size:255" json:"path"`
	Method    *string   `gorm:"size:16" json:"method"`
	SortOrder int       `gorm:"not null;default:0" json:"sort_order"`
	Visible   bool      `gorm:"not null;default:true" json:"visible"`
	Status    int16     `gorm:"not null;default:1" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (SysPermission) TableName() string { return "sys_permission" }

type SysUserRole struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"not null;index" json:"user_id"`
	RoleID    int64     `gorm:"not null;index" json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (SysUserRole) TableName() string { return "sys_user_role" }

type SysRolePermission struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	RoleID       int64     `gorm:"not null;index" json:"role_id"`
	PermissionID int64     `gorm:"not null;index" json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

func (SysRolePermission) TableName() string { return "sys_role_permission" }

// ---------- 日志 ----------

type SysLoginLog struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	UserID      *int64    `json:"user_id"`
	Username    string    `gorm:"size:64;not null" json:"username"`
	LoginResult int16     `gorm:"not null" json:"login_result"` // 1成功 2失败
	FailReason  *string   `gorm:"size:255" json:"fail_reason"`
	IPAddress   string    `gorm:"size:64;not null" json:"ip_address"`
	UserAgent   *string   `gorm:"size:255" json:"user_agent"`
	LoginAt     time.Time `gorm:"not null" json:"login_at"`
}

func (SysLoginLog) TableName() string { return "sys_login_log" }

type SysOperationLog struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	UserID        *int64    `json:"user_id"`
	Module        string    `gorm:"size:64;not null" json:"module"`
	Action        string    `gorm:"size:64;not null" json:"action"`
	TargetType    *string   `gorm:"size:64" json:"target_type"`
	TargetID      *int64    `json:"target_id"`
	RequestPath   string    `gorm:"size:255;not null" json:"request_path"`
	RequestMethod string    `gorm:"size:16;not null" json:"request_method"`
	RequestBody   *string   `gorm:"type:jsonb" json:"request_body"`
	ResponseCode  int       `gorm:"not null" json:"response_code"`
	Success       bool      `gorm:"not null" json:"success"`
	CreatedAt     time.Time `json:"created_at"`
}

func (SysOperationLog) TableName() string { return "sys_operation_log" }

type AuditLog struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	UserID     *int64    `json:"user_id"`
	Module     string    `gorm:"size:64;not null" json:"module"`
	Action     string    `gorm:"size:64;not null" json:"action"`
	BizType    string    `gorm:"size:64;not null" json:"biz_type"`
	BizID      *int64    `json:"biz_id"`
	BeforeData *string   `gorm:"type:jsonb" json:"before_data"`
	AfterData  *string   `gorm:"type:jsonb" json:"after_data"`
	IPAddress  *string   `gorm:"size:64" json:"ip_address"`
	CreatedAt  time.Time `json:"created_at"`
}

func (AuditLog) TableName() string { return "audit_log" }

// ---------- 配置与文件 ----------

type SysConfig struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	ConfigKey   string    `gorm:"size:128;not null;uniqueIndex" json:"config_key"`
	ConfigValue string    `gorm:"type:text;not null" json:"config_value"`
	ValueType   string    `gorm:"size:32;not null" json:"value_type"`
	Remark      *string   `gorm:"size:255" json:"remark"`
	UpdatedBy   *int64    `json:"updated_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (SysConfig) TableName() string { return "sys_config" }

type SysFile struct {
	ID              int64     `gorm:"primaryKey" json:"id"`
	BizType         string    `gorm:"size:64;not null" json:"biz_type"`
	BizID           int64     `gorm:"not null;default:0" json:"biz_id"`
	FileName        string    `gorm:"size:255;not null" json:"file_name"`
	FilePath        string    `gorm:"size:500;not null" json:"file_path"`
	FileSize        int64     `gorm:"not null" json:"file_size"`
	MimeType        *string   `gorm:"size:128" json:"mime_type"`
	StorageProvider string    `gorm:"size:64;not null;default:minio" json:"storage_provider"`
	UploadedBy      *int64    `json:"uploaded_by"`
	CreatedAt       time.Time `json:"created_at"`
}

func (SysFile) TableName() string { return "sys_file" }
