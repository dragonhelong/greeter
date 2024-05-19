package model

// User model
type User struct {
	BaseModel
	Name  string `gorm:"column:name" json:"name"`
	Email string `gorm:"column:email" json:"email"`
	Phone string `gorm:"column:phone" json:"phone"`
}

// TableName returns table name of t_user.
func (User) TableName() string {
	return "t_user"
}
