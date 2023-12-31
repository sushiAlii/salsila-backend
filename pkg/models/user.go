package models

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	UID 		string 			`gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"uid"`
	RoleID		uint			`gorm:"not null" json:"roleId"`
	PersonsUID	*string			`json:"personsUid"`
	Email 		string			`gorm:"uniqueIndex;not null" json:"email"`
	Password	string			`gorm:"not null" json:"password,omitempty"`
	CreatedAt	time.Time		`gorm:"type:timestamptz" json:"-"`
	UpdatedAt	*time.Time		`gorm:"type:timestamptz" json:"-"`
	DeletedAt	gorm.DeletedAt	`gorm:"type:timestamptz" json:"-"`
}

type UserService interface {
	ValidateUser(*User) error
	CreateUser(*User) error
	GetAllUsers() ([]User, error)
	GetUserByUID(string) (*User, error)
	GetUserByEmail(string) (*User, error)
	AttachPerson(string, string) error
	DeleteUserByUID(string) error
}

type userService struct {
	DB *gorm.DB
}

func NewUserService(db *gorm.DB) UserService {
	return &userService{DB: db}
}

func (us *userService) ValidateUser(user *User) error {
	if user.RoleID == 0 {
		return ErrRoleIDRequired
	}

	if strings.TrimSpace(user.Email) == "" {
		return ErrEmailRequired
	}

	var existingUser User

	fmt.Printf("Finding email with %s", user.Email)

	if err := us.DB.Where("email = ?", user.Email).First(&existingUser).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrEmailNotUnique
		}
	}

	password := strings.TrimSpace(user.Password)

	if password == "" {
		return ErrPasswordRequired
	}

	if len(password) <= 4 {
		return ErrPasswordMinChar
	}

	return nil
}

func (us *userService) CreateUser(newUser *User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	newUser.Password = string(hashedPassword)

	return us.DB.Omit("uid").Create(newUser).Error
}

func (us *userService) GetAllUsers() ([]User, error) {
	var usersList []User

	if err := us.DB.Select("uid, role_id, persons_uid, email").Find(&usersList).Error; err != nil {
		return nil, err
	}

	return usersList, nil
}

func (us *userService) GetUserByUID(uid string) (*User, error) {
	var user User

	if err := us.DB.Select("uid, role_id, persons_uid, email").Where("uid = ?", uid).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (us *userService) GetUserByEmail(email string) (*User, error) {
	var user User

	if err := us.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

func (us *userService) AttachPerson(personUid string, userUid string) error {
	tx := us.DB.Begin()

	if err := tx.Model(&User{}).Where("uid = ?", userUid).Update("persons_uid", personUid).Error; err != nil {
		tx.Rollback()

		return err
	}

	return tx.Commit().Error
}

func (us *userService) DeleteUserByUID(uid string) error {
	return us.DB.Delete(&User{}, "uid = ?", uid).Error
}