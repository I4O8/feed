package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// signTokens 为 1..n 号用户各签一个 token，写文件（每行 `userID token`）
func signTokens(n int, secret, path string) error {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		t, err := signToken(int64(i), secret)
		if err != nil {
			return err
		}
		sb.WriteString(fmt.Sprintf("%d %s\n", i, t))
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("已签发 %d 个 token → %s\n", n, path)
	return nil
}

// signToken 与 feed 项目 service.Sign 保持一致（HS256，claims: userID + exp 7d）
func signToken(userID int64, secret string) (string, error) {
	claims := jwt.MapClaims{
		"userID": userID,
		"exp":    time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
