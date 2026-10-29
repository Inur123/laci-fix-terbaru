package domain

import "time"

const (
	RoleCabang = "SEKRETARIS_CABANG"
	RolePAC    = "SEKRETARIS_PAC"
)

type User struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	EmailVerified  bool       `json:"emailVerified"`
	Image          *string    `json:"image"`
	Role           string     `json:"role"`
	IsActive       bool       `json:"isActive"`
	PeriodeAktifID *string    `json:"periodeAktifId"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	LastLogoutAt   *time.Time `json:"lastLogoutAt,omitempty"`
}

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Token     string    `json:"-"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
}

type SessionResponse struct {
	Session Session `json:"session"`
	User    User    `json:"user"`
}

type ErrorBody struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
