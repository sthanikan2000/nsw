package company

import (
	"encoding/json"
	"time"
)

// Record represents a company's persisted profile in the database.
type Record struct {
	ID        string          `gorm:"type:varchar(100);column:id;primaryKey;not null" json:"id"`
	Name      string          `gorm:"type:varchar(255);column:name;not null" json:"name"`
	OUID      string          `gorm:"type:varchar(255);column:ou_id;unique;not null" json:"ouId"`
	OUHandle  string          `gorm:"type:varchar(255);column:ou_handle;unique;not null" json:"ouHandle"`
	Data      json.RawMessage `gorm:"type:jsonb;column:data;not null;default:'{}'" json:"data"`
	CreatedAt time.Time       `gorm:"column:created_at;autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time       `gorm:"column:updated_at;autoUpdateTime" json:"updatedAt"`
}

func (r *Record) TableName() string {
	return "company_records"
}
