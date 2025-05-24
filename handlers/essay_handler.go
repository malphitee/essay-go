package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"essay-go/config"
	"essay-go/models"
	"essay-go/services"
)



// PolishEssayStream 处理作文润色流式输出请求
func PolishEssayStream(c *gin.Context) {
	// 记录请求开始
	gin.DefaultWriter.Write([]byte("[PolishEssayStream] 开始处理流式润色请求\n"))
	
	// 获取请求参数
	title := c.Query("title")
	content := c.Query("content")
	
	// 记录请求内容
	gin.DefaultWriter.Write([]byte(fmt.Sprintf("[PolishEssayStream] 收到请求: 标题=%s, 内容长度=%d\n", title, len(content))))
	
	// 验证内容不为空
	if content == "" {
		gin.DefaultWriter.Write([]byte("[PolishEssayStream] 作文内容为空\n"))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "作文内容不能为空",
		})
		return
	}
	
	// 设置SSE相关的响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Status(http.StatusOK)
	
	// 立即发送一个初始消息，确保连接建立
	c.SSEvent("", "正在润色中...")
	
	// 创建AI服务
	gin.DefaultWriter.Write([]byte("[PolishEssayStream] 创建AI服务\n"))
	aiService := services.NewAIService(config.LoadConfig())
	
	// 调用AI服务润色作文
	gin.DefaultWriter.Write([]byte("[PolishEssayStream] 调用AI服务润色作文\n"))
	polishedContent, err := aiService.PolishEssay(title, content)
	if err != nil {
		errMsg := fmt.Sprintf("[PolishEssayStream] 润色作文失败: %v\n", err)
		gin.DefaultWriter.Write([]byte(errMsg))
		
		// 发送错误事件
		c.SSEvent("error", fmt.Sprintf("润色失败: %v", err))
		return
	}
	
	// 将润色结果分成多个小块流式输出
	chunks := splitIntoChunks(polishedContent, 10) // 每10个字符一个块
	
	for i, chunk := range chunks {
		// 模拟处理时间
		time.Sleep(100 * time.Millisecond)
		
		// 使用Gin的SSEvent函数发送数据
		c.SSEvent("", chunk)
		
		gin.DefaultWriter.Write([]byte(fmt.Sprintf("[PolishEssayStream] 发送块 %d: %s\n", i, chunk)))
		
		// 手动刷新缓冲区
		c.Writer.Flush()
	}
	
	// 发送完成标记
	time.Sleep(100 * time.Millisecond)
	c.SSEvent("", "[DONE]")
	c.Writer.Flush()
	
	gin.DefaultWriter.Write([]byte("[PolishEssayStream] 处理完成\n"))
}

// splitIntoChunks 将文本按 rune (字符) 分成多个小块
func splitIntoChunks(text string, chunkSize int) []string {
	var chunks []string
	runes := []rune(text) // 将字符串转换为 rune 切片
	textLenInRunes := len(runes)

	for i := 0; i < textLenInRunes; i += chunkSize {
		end := i + chunkSize
		if end > textLenInRunes {
			end = textLenInRunes
		}
		chunks = append(chunks, string(runes[i:end])) // 将 rune 子切片转换回字符串
	}
	return chunks
}

// PolishEssay 处理作文润色请求
func PolishEssay(c *gin.Context) {
	var request models.EssayRequest

	gin.DefaultWriter.Write([]byte("[PolishEssay] 开始处理润色请求\n"))

	if err := c.ShouldBindJSON(&request); err != nil {
		errMsg := fmt.Sprintf("[PolishEssay] 请求参数绑定失败: %v\n", err)
		gin.DefaultWriter.Write([]byte(errMsg))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "请求参数无效",
			"error":   err.Error(),
		})
		return
	}

	reqJSON, _ := json.Marshal(request) // Safe to ignore error for logging
	gin.DefaultWriter.Write([]byte(fmt.Sprintf("[PolishEssay] 收到请求: %s\n", string(reqJSON))))

	if request.Content == "" {
		gin.DefaultWriter.Write([]byte("[PolishEssay] 作文内容为空\n"))
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "作文内容不能为空",
		})
		return
	}
	
	gin.DefaultWriter.Write([]byte("[PolishEssay] 创建AI服务\n"))
	aiService := services.NewAIService(config.LoadConfig())

	gin.DefaultWriter.Write([]byte("[PolishEssay] 调用AI服务润色作文\n"))
	// 修改：接收 polishedContent
	polishedContent, err := aiService.PolishEssay(request.Title, request.Content) 
	if err != nil {
		errMsg := fmt.Sprintf("[PolishEssay] AI服务润色作文失败: %v\n", err)
		gin.DefaultWriter.Write([]byte(errMsg))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "润色作文失败 (AI service error)",
			"error":   err.Error(),
		})
		return
	}
	
	gin.DefaultWriter.Write([]byte(fmt.Sprintf("[PolishEssay] AI服务调用完成, 润色后内容长度: %d\n", len(polishedContent))))

	// ---- 修改：返回实际的润色结果 ----
	actualResponse := gin.H{
		"title":           request.Title, // 或者您可以考虑让 AI 服务也返回处理后的标题
		"polishedContent": polishedContent,
		"status":          "success", // 或者 "ok"
	}
	
	// 可选：记录实际发送的响应
	actualRespJSON, _ := json.Marshal(actualResponse)
	gin.DefaultWriter.Write([]byte(fmt.Sprintf("[PolishEssay] PREPARING ACTUAL RESPONSE: %s\n", string(actualRespJSON))))
	
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")

	gin.DefaultWriter.Write([]byte("[PolishEssay] SENDING ACTUAL JSON RESPONSE\n"))
	c.JSON(http.StatusOK, actualResponse) 
	// ---- 结束修改 ----
	
	gin.DefaultWriter.Write([]byte("[PolishEssay] 处理完成 (after sending actual response)\n"))
}
