package model

// Label is the model for labels.
type Label struct {
	Model

	// LabelName is the unique label name
	LabelName string `gorm:"column:labelName;unique;not null;size:50"`

	// AiResources is the many-to-many relationship with AI resources
	AiResources []AiResource `gorm:"many2many:resource_label;"`
}

// TableName implements [gorm.io/gorm/schema.Tabler].
func (Label) TableName() string {
	return "label"
}