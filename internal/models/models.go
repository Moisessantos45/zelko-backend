package models

import (
	"time"
)

type Auth struct {
	ID               uint64 `json:"id" gorm:"primaryKey"`
	Email            string `json:"email" gorm:"unique;not null"`
	PasswordHash     string `json:"-" gorm:"not null"`
	TwoFactorSecret  string `json:"-" gorm:"type:varchar(26)"`
	TwoFactorEnabled bool   `json:"two_factor_enabled" gorm:"default:false"`
	FirstSession     bool   `json:"first_session" gorm:"default:true"`
	FullProfile      bool   `json:"full_profile" gorm:"default:false"`
	EmailConfirmed   bool   `json:"email_confirmed" gorm:"default:false"`
	Active           bool   `json:"active" gorm:"default:true"`
	Token            string `json:"token" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type User struct {
	ID        uint64 `json:"id" gorm:"primaryKey"`
	AuthID    uint64 `json:"auth_id" gorm:"not null"`
	FirstName string `json:"first_name" gorm:"not null;varchar(100)"`
	LastName  string `json:"last_name" gorm:"not null;varchar(100)"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	Auth Auth `json:"-" gorm:"foreignKey:AuthID"`
}

type Visibility struct {
	ID   uint64 `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"unique;not null;varchar(50)"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

type Document struct {
	ID           uint64 `json:"id" gorm:"primaryKey"`
	Slug         string `json:"slug" gorm:"size:64;unique;not null"`
	UserID       uint64 `json:"user_id" gorm:"not null"`
	Title        string `json:"title" gorm:"size:255;not null"`
	Content      string `json:"content" gorm:"type:text"`
	URL          string `json:"url" gorm:"size:255"`
	VisibilityID uint64 `json:"visibility_id" gorm:"not null"`
	TypeDocument string `json:"type_document" gorm:"size:5;not null;default:'latex'"`

	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	User       User       `json:"-" gorm:"foreignKey:UserID"`
	Visibility Visibility `json:"-" gorm:"foreignKey:VisibilityID"`
}

var VisibilityOptions = []Visibility{
	{ID: 1, Name: "public"},
	{ID: 2, Name: "private"},
	{ID: 3, Name: "shared"},
}

var Models = []any{
	&Auth{},
	&User{},
	&Visibility{},
	&Document{},
}
