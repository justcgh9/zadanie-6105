package user

import "time"

type User struct {
	Id        string    `json:"id"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type OrganizationResponsible struct {
	Id             string `json:"id"`
	OrganizationId string `json:"organization_id"`
	UserId         string `json:"user_id"`
}
