package model

// AiResource is the model for AI resources.
type AiResource struct {
	Model

	// AiResourceID is the primary key (using Model.ID instead)
	// URL is the resource URL
	URL string `gorm:"column:url;not null"`

	// Labels is the many-to-many relationship with labels
	Labels []Label `gorm:"many2many:resource_label;"`
}

// TableName implements [gorm.io/gorm/schema.Tabler].
func (AiResource) TableName() string {
	return "aiResource"
}