package service

import (
	"errors"
	"fmt"
	"time"

	"feed/config"

	"github.com/golang-jwt/jwt/v5"
)

// Sign 签发jtw凭证
func Sign(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}

// Parse jtw认证
func Parse(tokenStr string) (int64, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(config.JWTSecret), nil
	})
	if err != nil {
		return 0, err
	}

	userIDFloat, ok := claims["userID"].(float64)
	if !ok {
		return 0, errors.New("token 缺少 userID 或类型不正确")
	}
	return int64(userIDFloat), nil
}
