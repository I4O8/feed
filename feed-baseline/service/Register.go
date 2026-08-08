package service

import (
	"errors"
	"math/rand"
	"strings"

	"feed/dao"
	"feed/model"
)

var ErrUserExists = errors.New("user already exists")

// generateRandomString 随机字符生成
func generateRandomString(length int) string {
	const charset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// Register 注册（退化版：无 IP 限流）
func Register(ip, input, pwd string) (string, int64, error) {
	var email, pNumber *string

	if strings.Contains(input, "@") {
		acc, err := dao.FindByEmail(input)
		if acc == nil && err == nil {
			email = &input
		} else if acc != nil && err == nil {
			return "", -1, ErrUserExists
		} else {
			return "", -1, err
		}
	} else {
		acc, err := dao.FindByPNumber(input)
		if acc == nil && err == nil {
			pNumber = &input
		} else if acc != nil && err == nil {
			return "", -1, ErrUserExists
		} else {
			return "", -1, err
		}
	}
	hash, err := HashPassword(pwd)
	if err != nil {
		return "", -1, err
	}
	name := "新用户_" + generateRandomString(6)
	tmp := model.Account{Name: name, PNumber: pNumber, Email: email, Password: hash}
	userID, err := dao.CreateAccount(&tmp)
	if err != nil {
		return "", -1, err
	}
	token, err := Sign(userID)
	if err != nil {
		return "", -1, err
	}
	return token, userID, nil
}
