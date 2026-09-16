package domain

import "time"

type UserRecord struct {
	ID           string    `json:"id" db:"id"`
	FullName     string    `json:"full_name" db:"full_name"`
	Email        string    `json:"email" db:"email"`
	Phone        *string   `json:"phone,omitempty" db:"phone"`
	PasswordHash string    `json:"-" db:"password_hash"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type PublicUser struct {
	ID       string  `json:"id"`
	FullName string  `json:"full_name"`
	Email    string  `json:"email"`
	Phone    *string `json:"phone,omitempty"`
}

func (u *UserRecord) ToPublic() *PublicUser {
	return &PublicUser{
		ID:       u.ID,
		FullName: u.FullName,
		Email:    u.Email,
		Phone:    u.Phone,
	}
}
