package service

import (
	"errors"
	"strings"

	"feed/dao"
)

var (
	ErrInvalidCredentials = errors.New("登录名或密码错误")
	ErrBanned             = errors.New("账户已被封禁")
)

func Login(input, pwd string) (string, int64, error) {
	if strings.Contains(input, "@") {
		acc, err := dao.FindByEmail(input)
		if err != nil {
			return "", -1, err
		}
		if acc == nil {
			return "", -1, ErrInvalidCredentials
		}
		if VerifyPassword(acc.Password, pwd) {
			if !acc.Banned {
				token, err := Sign(acc.ID)
				if err != nil {
					return "", -1, err
				}
				return token, acc.ID, nil
			}
		} else {
			return "", -1, ErrInvalidCredentials
		}
	} else {
		acc, err := dao.FindByPNumber(input)
		if err != nil {
			return "", -1, err
		}
		if acc == nil {
			return "", -1, ErrInvalidCredentials
		}
		if VerifyPassword(acc.Password, pwd) {
			if !acc.Banned {
				token, err := Sign(acc.ID)
				if err != nil {
					return "", -1, err
				}
				return token, acc.ID, nil
			}
		} else {
			return "", -1, ErrInvalidCredentials
		}
	}

	return "", -1, ErrBanned
}
