package model

// Blog model
type Blog struct {
	BaseModel
	Content string `gorm:"column:content" json:"content"`
	Author  string `gorm:"column:author" json:"author"`
}

// TableName returns table name of t_blog.
func (Blog) TableName() string {
	return "t_blog"
}
