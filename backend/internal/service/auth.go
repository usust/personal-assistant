package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"personal_assistant_server/internal/model"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type Auth struct {
	db          *gorm.DB
	secret      []byte
	expireHours int
}

type Claims struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

func NewAuth(db *gorm.DB, secret string, expireHours int) *Auth {
	return &Auth{db: db, secret: []byte(secret), expireHours: expireHours}
}

func (a *Auth) Login(username, password string) (*model.User, string, time.Time, error) {
	var user model.User
	if err := a.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	expiresAt := time.Now().Add(time.Duration(a.expireHours) * time.Hour)
	claims := Claims{
		UserID: user.ID, Username: user.Username, Role: user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "personal-assistant",
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secret)
	return &user, token, expiresAt, err
}

func (a *Auth) Parse(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return a.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func (a *Auth) UserByID(id uint) (*model.User, error) {
	var user model.User
	if err := a.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}
