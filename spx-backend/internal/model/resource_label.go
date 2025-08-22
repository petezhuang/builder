package model

// ResourceLabel is the model for the many-to-many relationship between AI resources and labels.
type ResourceLabel struct {
	Model

	// AiResourceID is the foreign key to aiResource table
	AiResourceID int64 `gorm:"column:aiResourceId;not null;index"`
	AiResource   AiResource `gorm:"foreignKey:AiResourceID"`

	// LabelID is the foreign key to label table  
	LabelID int64 `gorm:"column:labelId;not null;index"`
	Label   Label `gorm:"foreignKey:LabelID"`
}

// TableName implements [gorm.io/gorm/schema.Tabler].
func (ResourceLabel) TableName() string {
	return "resource_label"
}