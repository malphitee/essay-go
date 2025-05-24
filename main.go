package main

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin" // 保持这个

	// 确认这些路径相对于 go.mod 中的 "module essay-go" 是正确的
	"essay-go/config"
	"essay-go/handlers"
	"essay-go/middleware"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()

	// 设置Gin模式
	if cfg.Production {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode) // 确保在非生产环境下是Debug模式
	}

	// 初始化路由
	router := gin.Default()

	// 加载HTML模板
	// 确保 'templates' 文件夹在项目的根目录下，并且包含 index.html
	router.LoadHTMLGlob("templates/*")

	// 添加自定义中间件
	// 注意：Logger中间件可能会因为CORS的移除而需要调整或移除，取决于其实现
	// 如果Logger中间件与CORS有强依赖或特定逻辑，可能需要修改或暂时注释掉
	router.Use(middleware.Logger()) // 暂时保留，后续观察
	router.Use(middleware.Recovery())

	// API路由
	api := router.Group("/api")
	{
		api.POST("/polish", handlers.PolishEssay) // Keep the non-streaming version for now, or remove if fully replaced
		// 如果 PolishEssayStream 仍然需要，保留它
		api.GET("/polish/stream", handlers.PolishEssayStream) // Enable the streaming API
	}

	// 根路由，渲染 index.html
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "作文润色工具", // 可以传递给模板的数据
		})
	})

	// 启动服务器
	serverAddr := "0.0.0.0:" + cfg.Port
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second, // 稍微增加超时以应对潜在的AI长响应
		WriteTimeout: 60 * time.Second, // 显著增加写入超时
		IdleTimeout:  120 * time.Second,
	}

	log.Printf("服务器启动在 http://localhost:%s (或 http://%s)", cfg.Port, serverAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
