package model

import (
	"time"

	"gorm.io/datatypes"
)

// SystemConfig 系统配置表
type SystemConfig struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string         `gorm:"type:varchar(128);uniqueIndex;not null" json:"key"`
	Value       datatypes.JSON `gorm:"type:jsonb" json:"value"`
	// TODO(model/system): 配置表缺少 value_type、editable、validation_schema。
	// 没有类型元数据时，后台配置页面只能猜测 number/string/object。
	Description string         `gorm:"type:varchar(255)" json:"description"`
	UpdatedBy   int64          `gorm:"column:updated_by" json:"updated_by"`
	UpdatedAt   time.Time      `gorm:"not null" json:"updated_at"`
}

func (SystemConfig) TableName() string { return "system_configs" }
