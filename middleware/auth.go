package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTSecret 用于签名JWT的密钥
var JWTSecret = func() []byte {
	// 从环境变量获取密钥，如果不存在则使用默认值
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "default-secret-key-please-change-in-production"
	}
	return []byte(jwtSecret)
}()

// AuthRequired 验证JWT令牌的中间件
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			c.Abort()
			return
		}

		// 解析令牌
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("无效的签名方法: %v", token.Header["alg"])
			}
			return JWTSecret, nil
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌"})
			c.Abort()
			return
		}

		// 验证令牌有效性
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// 将用户名添加到上下文
			username, ok := claims["username"].(string)
			if !ok {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌内容"})
				c.Abort()
				return
			}
			c.Set("username", username)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的令牌"})
			c.Abort()
			return
		}
	}
}

// OptionalAuth 可选的认证中间件，不会阻止未认证的请求
func OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取令牌
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// 没有令牌，继续处理请求
			c.Next()
			return
		}

		// 解析令牌
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 验证签名方法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("无效的签名方法: %v", token.Header["alg"])
			}
			return JWTSecret, nil
		})

		if err != nil {
			// 令牌无效，但不阻止请求
			c.Next()
			return
		}

		// 验证令牌有效性
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// 将用户名添加到上下文
			username, ok := claims["username"].(string)
			if ok {
				c.Set("username", username)
			}
		}

		c.Next()
	}
}
