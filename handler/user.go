package handler

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"feed/service"

	"github.com/gin-gonic/gin"
)

type accountStruct struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}
type changePWDStruct struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
type updateStruct struct {
	Name *string `json:"name"`
	Bio  *string `json:"bio"`
}

// isValidAccount 检查账号是否为邮箱或11位纯数字手机号
func isValidAccount(account string) bool {
	if strings.Contains(account, "@") {
		parts := strings.Split(account, "@")
		if len(parts) != 2 {
			return false
		}
		local, domain := parts[0], parts[1]
		if len(local) == 0 || len(domain) == 0 {
			return false
		}
		if !strings.Contains(domain, ".") {
			return false
		}
		return true
	}

	if len(account) == 11 {
		for _, ch := range account {
			if ch < '0' || ch > '9' {
				return false
			}
		}
		return true
	}

	return false
}

// Register 注册handler
func Register(ctx *gin.Context) {
	var tmp accountStruct
	if err := ctx.ShouldBindJSON(&tmp); err != nil {
		Fail(ctx, CodeParamError, "请求参数错误")
		return
	}

	if valid := isValidAccount(tmp.Account); !valid {
		Fail(ctx, CodeLoginNameInvalid, "账号格式错误，请使用邮箱或11位手机号")
		return
	}

	if length := len(tmp.Password); length < 6 || length > 32 {
		Fail(ctx, CodePasswordLenInvalid, "密码长度不合法（需 6-32 位）")
		return
	}

	token, userID, err := service.Register(ctx.ClientIP(), tmp.Account, tmp.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserExists):
			Fail(ctx, CodeLoginNameRegistered, "用户已存在")
		case errors.Is(err, service.ErrRatelimited):
			Fail(ctx, CodeTooManyRequests, "注册过于频繁，请稍后再试")

		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Println("register err:", err) // 或 log.Println(err)
		}
		return
	}
	OK(ctx, gin.H{
		"token":  token,
		"userId": userID,
	})
}

// Login 登录handler
func Login(ctx *gin.Context) {
	var tmp accountStruct
	if err := ctx.ShouldBindJSON(&tmp); err != nil {
		Fail(ctx, CodeParamError, "请求参数错误")
		return
	}

	if valid := isValidAccount(tmp.Account); !valid {
		Fail(ctx, CodeLoginNameInvalid, "账号格式错误，请使用邮箱或11位手机号")
		return
	}

	if length := len(tmp.Password); length < 6 || length > 32 {
		Fail(ctx, CodePasswordLenInvalid, "密码长度不合法（需 6-32 位）")
		return
	}
	token, userID, err := service.Login(tmp.Account, tmp.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			Fail(ctx, CodeLoginNameOrPwdWrong, "错误的登陆名或密码")
		case errors.Is(err, service.ErrBanned):
			Fail(ctx, CodeAccountBanned, "用户已被封禁")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, gin.H{
		"token":  token,
		"userId": userID,
	})
}

func UpdateProfile(ctx *gin.Context) {
	userIDVal, exists := ctx.Get("UserId")
	if !exists {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "Token损坏或失效")
		return
	}
	var tmp updateStruct
	if err := ctx.ShouldBindJSON(&tmp); err != nil {
		Fail(ctx, CodeParamError, "请求参数错误")
		return
	}
	if tmp.Name == nil && tmp.Bio == nil {
		Fail(ctx, CodeParamError, "至少修改一个字段")
		return
	}

	if err := service.UpdateProfile(userID, tmp.Name, tmp.Bio); err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		return
	}
	OK(ctx, nil)
}

// GetProfile 获取用户公开资料
func GetProfile(ctx *gin.Context) {
	userID, err := strconv.ParseInt(ctx.Query("userId"), 10, 64)
	if err != nil {
		Fail(ctx, CodeParamError, "参数错误")
		return
	}
	profile, err := service.GetProfile(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUserNotFound):
			Fail(ctx, CodeUserNotFound, "用户不存在")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, profile)
}

func ChangePassword(ctx *gin.Context) {
	userIDVal, exists := ctx.Get("UserId")
	if !exists {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "Token损坏或失效")
		return
	}
	var tmp changePWDStruct
	if err := ctx.ShouldBindJSON(&tmp); err != nil {
		Fail(ctx, CodeParamError, "请求参数错误")
		return
	}
	if err := service.ChangePassword(userID, tmp.NewPassword, tmp.OldPassword); err != nil {
		switch {
		case errors.Is(err, service.ErrOldPasswordWrong):
			Fail(ctx, CodeOldPasswordWrong, "旧密码错误")
		default:
			Fail(ctx, CodeInternalError, "服务器内部错误")
			fmt.Printf("error:%v\n", err)
		}
		return
	}
	OK(ctx, nil)
}

func UploadAvatar(ctx *gin.Context) {
	userIDVal, exists := ctx.Get("UserId")
	if !exists {
		Fail(ctx, CodeTokenMissingOrExpired, "未登录")
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		Fail(ctx, CodeTokenMissingOrExpired, "Token损坏或失效")
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		Fail(ctx, CodeParamError, "输入文件有误")
		return
	}
	fileName := file.Filename
	if ext := filepath.Ext(fileName); ext != ".jpg" && ext != ".png" && ext != ".webp" {
		Fail(ctx, CodeFileTypeUnsupported, "不被允许的图片后缀")
		return
	}
	if file.Size > 5*1024*1024 {
		Fail(ctx, CodeFileTooLarge, "图片过大！")
		return
	}

	tmp, err := file.Open()
	if err != nil {
		Fail(ctx, CodeInternalError, "文件打开失败")
		return
	}
	data, err := io.ReadAll(tmp)
	defer tmp.Close()
	if err != nil {
		Fail(ctx, CodeInternalError, "文件读取失败")
		return
	}

	urlPath, err := service.UploadAvatar(userID, data, fileName)
	if err != nil {
		Fail(ctx, CodeInternalError, "服务器内部错误")
		fmt.Printf("error:%v\n", err)
		return
	}
	OK(ctx, gin.H{"avatarUrl": urlPath})
}
