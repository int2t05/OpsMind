// Package hash 提供密码哈希和验证工具。
//
// 使用 bcrypt 算法进行密码哈希，cost=10 在 4C8GB 环境下单次哈希约 100ms，
// 满足登录场景性能要求。
//
// 密码策略遵循 TECH.md §8.3 规范：
// - 长度 8-32 位
// - 必须包含大写字母、小写字母和数字
package hash

import (
	"errors"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// ErrPasswordTooShort 密码太短
var ErrPasswordTooShort = errors.New("密码长度不足 8 位")

// ErrPasswordTooLong 密码太长
var ErrPasswordTooLong = errors.New("密码长度超过 32 位")

// ErrPasswordWeak 密码强度不足
var ErrPasswordWeak = errors.New("密码必须包含大写字母、小写字母和数字")

// HashPassword 使用 bcrypt 对密码进行单向哈希。
// cost=10 在 4C8GB 环境下单次哈希约 100ms，满足登录场景性能要求。
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	return string(bytes), err
}

// CheckPassword 验证密码是否匹配哈希值
func CheckPassword(hashed, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(password))
	return err == nil
}

// ValidatePassword 校验密码是否符合策略要求。
// 策略：至少一个小写字母、一个大写字母、一个数字，长度 8-32。
func ValidatePassword(password string) error {
	// TODO(hash): 当前长度用 len(bytes) 计算，含中文或 emoji 时会按字节数限制。
	// 如果允许非 ASCII 密码，应改用 utf8.RuneCountInString 并明确前端同样规则。
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 32 {
		return ErrPasswordTooLong
	}

	hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
	hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasDigit := false
	for _, r := range password {
		if unicode.IsDigit(r) {
			hasDigit = true
			break
		}
	}

	if !hasLower || !hasUpper || !hasDigit {
		return ErrPasswordWeak
	}

	return nil
}
