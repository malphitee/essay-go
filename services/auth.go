package services

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"essay-go/models"
)

// AuthService 提供认证相关功能
type AuthService struct {
	users     map[string]string // 用户名 -> 密码
	authFile  string
	userMutex sync.RWMutex
}

// 全局认证服务实例
var authService *AuthService
var authOnce sync.Once

// GetAuthService 返回认证服务的单例实例
func GetAuthService() *AuthService {
	authOnce.Do(func() {
		authService = &AuthService{
			users:    make(map[string]string),
			authFile: "data/auth.txt",
		}
		authService.loadUsers()
	})
	return authService
}

// loadUsers 从文件加载用户信息
func (a *AuthService) loadUsers() error {
	a.userMutex.Lock()
	defer a.userMutex.Unlock()

	file, err := os.Open(a.authFile)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			username := parts[0]
			password := parts[1]
			a.users[username] = password
		}
	}

	return scanner.Err()
}

// Authenticate 验证用户凭据
func (a *AuthService) Authenticate(username, password string) bool {
	a.userMutex.RLock()
	defer a.userMutex.RUnlock()

	storedPassword, exists := a.users[username]
	if !exists {
		return false
	}

	return storedPassword == password
}

// GetUser 获取用户信息（不包含密码）
func (a *AuthService) GetUser(username string) *models.User {
	a.userMutex.RLock()
	defer a.userMutex.RUnlock()

	_, exists := a.users[username]
	if !exists {
		return nil
	}

	return &models.User{
		Username: username,
		LoggedIn: true,
	}
}
