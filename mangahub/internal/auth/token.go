package auth

import (
	"database/sql"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

var (
	jwtSecret       = []byte("supersecretkey") // change later
	accessTokenTTL  = 1 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func CreateAccessToken(userID, username string) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(jwtSecret)
}

func CreateRefreshToken(db *sql.DB, userID string) (string, error) {
	token := uuid.NewString()
	expires := time.Now().Add(refreshTokenTTL)

	_, err := db.Exec(`
        INSERT INTO refresh_tokens (token, user_id, expires_at)
        VALUES (?, ?, ?)
    `, token, userID, expires)

	return token, err
}

func ValidateRefreshToken(db *sql.DB, token string) (string, error) {
	var userID string
	var expiresAt time.Time

	err := db.QueryRow(`
        SELECT user_id, expires_at 
        FROM refresh_tokens 
        WHERE token = ?
    `, token).Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		return "", errors.New("invalid refresh token")
	}
	if err != nil {
		return "", err
	}
	if time.Now().After(expiresAt) {
		_ = RevokeRefreshToken(db, token)
		return "", errors.New("refresh token expired")
	}

	return userID, nil
}

func RevokeRefreshToken(db *sql.DB, token string) error {
	_, err := db.Exec(`DELETE FROM refresh_tokens WHERE token = ?`, token)
	return err
}

func ParseAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
