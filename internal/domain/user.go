package domain

// User is an account that owns tasks and may belong to teams.
type User struct {
	Base
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"`
}

func (User) TableName() string { return "users" }
