package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"essay-go/models"
	"essay-go/services"
)

// 用于签名JWT的密钥
var jwtSecret = []byte("your-secret-key")

// LoginRequest 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应结构
type LoginResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// SyncEssaysRequest 同步请求结构
type SyncEssaysRequest struct {
	Essays []models.Essay `json:"essays" binding:"required"`
}

// Login 处理用户登录
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 验证用户凭据
	authService := services.GetAuthService()
	if !authService.Authenticate(req.Username, req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 创建JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":      time.Now().Add(time.Hour * 24 * 7).Unix(), // 7天过期
	})

	// 签名JWT
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法生成令牌"})
		return
	}

	// 返回令牌和用户信息
	user := authService.GetUser(req.Username)
	c.JSON(http.StatusOK, LoginResponse{
		Token: tokenString,
		User:  *user,
	})
}

// GetUserInfo 获取当前登录用户信息
func GetUserInfo(c *gin.Context) {
	// 从上下文中获取用户名
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 获取用户信息
	authService := services.GetAuthService()
	user := authService.GetUser(username.(string))
	if user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// SyncEssays 同步作文到DynamoDB
func SyncEssays(c *gin.Context) {
	// 从上下文中获取用户名
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	var req SyncEssaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 获取DynamoDB客户端
	dynamoDBClient := services.GetDynamoDBClient()
	if dynamoDBClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DynamoDB服务未初始化"})
		return
	}

	// 保存每篇作文
	for _, essay := range req.Essays {
		// 确保作文属于当前用户
		essay.Username = username.(string)
		
		// 保存到DynamoDB
		err := dynamoDBClient.SaveEssay(essay)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存作文失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "同步成功"})
}

// GetEssays 从DynamoDB获取用户的所有作文
func GetEssays(c *gin.Context) {
	// 从上下文中获取用户名
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 获取DynamoDB客户端
	dynamoDBClient := services.GetDynamoDBClient()
	if dynamoDBClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DynamoDB服务未初始化"})
		return
	}

	// 获取用户的所有作文
	essays, err := dynamoDBClient.GetEssaysByUsername(username.(string))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取作文失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"essays": essays})
}

// DeleteEssay 从DynamoDB软删除作文
func DeleteEssay(c *gin.Context) {
	// 从上下文中获取用户名
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未登录"})
		return
	}

	// 获取作文ID
	essayIDStr := c.Param("id")
	if essayIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少作文ID"})
		return
	}

	// 将字符串ID转换为 int64
	essayID, err := strconv.ParseInt(essayIDStr, 10, 64)
	if err != nil {
		// 如果不是数字，尝试将其作为时间戳处理
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作文ID格式"})
		return
	}

	// 获取DynamoDB客户端
	dynamoDBClient := services.GetDynamoDBClient()
	if dynamoDBClient == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DynamoDB服务未初始化"})
		return
	}

	// 软删除作文，使用 username 和 id 作为主键和排序键
	err = dynamoDBClient.DeleteEssay(username.(string), essayID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除作文失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
