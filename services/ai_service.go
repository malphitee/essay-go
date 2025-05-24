package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"essay-go/config"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AIService AI服务接口
type AIService interface {
	PolishEssay(title, content string) (string, error)
}

// DefaultAIService 默认AI服务实现
type DefaultAIService struct {
	cfg *config.Config
}

// NewAIService 创建新的AI服务
func NewAIService(cfg *config.Config) AIService {
	return &DefaultAIService{
		cfg: cfg,
	}
}

// PolishEssay 使用AI润色作文
func (s *DefaultAIService) PolishEssay(title, content string) (string, error) {
	fmt.Printf("[PolishEssay] 开始润色作文, 标题: %s, 内容长度: %d字符\n", title, len(content))
	
	// 如果配置了DeepSeek API密钥，使用DeepSeek润色
	if s.cfg.DeepSeekAPIKey != "" {
		fmt.Printf("[PolishEssay] 使用DeepSeek API润色, 密钥长度: %d\n", len(s.cfg.DeepSeekAPIKey))
		return s.deepSeekPolish(title, content)
	}

	// 如果配置了其他AI服务，使用其他AI服务
	if s.cfg.AIEndpoint != "" && s.cfg.AIKey != "" {
		fmt.Printf("[PolishEssay] 使用其他AI服务润色, 端点: %s\n", s.cfg.AIEndpoint)
		return s.otherAIPolish(title, content)
	}

	// 如果没有配置AI服务，使用模拟润色
	fmt.Println("[PolishEssay] 未配置AI服务，使用模拟润色")
	return s.mockPolish(title, content), nil
}

// deepSeekPolish 使用DeepSeek API润色作文
func (s *DefaultAIService) deepSeekPolish(title, content string) (string, error) {
	// DeepSeek API端点
	apiEndpoint := "https://api.deepseek.com/v1/chat/completions"
	fmt.Printf("[deepSeekPolish] 开始调用DeepSeek API, 端点: %s\n", apiEndpoint)

	// 准备提示词
	prompt := fmt.Sprintf("你是一位专业的中文作文润色专家，尤其擅长帮助小学生改进作文。\n\n"+
		"请帮我润色以下作文，使其更加生动、有表现力、结构合理。保持原文的主要意思和结构，但可以改进语言表达、修正语法错误、丰富词汇和优化段落结构。\n\n"+
		"作文标题：%s\n\n"+
		"作文正文：\n%s\n\n"+
		"请直接返回润色后的完整作文，不需要其他解释。", title, content)

	// 检查DeepSeek模型配置
	if s.cfg.DeepSeekModel == "" {
		s.cfg.DeepSeekModel = "deepseek-chat"
		fmt.Printf("[deepSeekPolish] 未指定模型，使用默认模型: %s\n", s.cfg.DeepSeekModel)
	} else {
		fmt.Printf("[deepSeekPolish] 使用模型: %s\n", s.cfg.DeepSeekModel)
	}

	// 准备请求数据
	requestData := map[string]interface{}{
		"model": s.cfg.DeepSeekModel,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": 0.7,
		"max_tokens":  2000,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("[deepSeekPolish] 序列化请求数据失败: %v\n", err)
		return "", fmt.Errorf("序列化请求数据失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", apiEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("[deepSeekPolish] 创建HTTP请求失败: %v\n", err)
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.DeepSeekAPIKey)
	fmt.Println("[deepSeekPolish] 请求头设置完成，准备发送请求")

	// 发送请求
	client := &http.Client{Timeout: 60 * time.Second}
	fmt.Println("[deepSeekPolish] 开始发送HTTP请求...")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[deepSeekPolish] 发送DeepSeek请求失败: %v\n", err)
		return "", fmt.Errorf("发送DeepSeek请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	fmt.Printf("[deepSeekPolish] 收到响应，状态码: %d\n", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		// 读取错误响应体
		var errorResponse map[string]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err == nil {
			fmt.Printf("[deepSeekPolish] DeepSeek API错误响应: %v\n", errorResponse)
			return "", fmt.Errorf("DeepSeek API错误: %v", errorResponse)
		}
		fmt.Printf("[deepSeekPolish] DeepSeek API返回错误状态码: %d\n", resp.StatusCode)
		return "", fmt.Errorf("DeepSeek API返回错误状态码: %d", resp.StatusCode)
	}

    // 读取响应体
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        fmt.Printf("[deepSeekPolish] 读取DeepSeek响应体失败: %v\n", err)
        return "", fmt.Errorf("读取DeepSeek响应体失败: %w", err)
    }
    
    // 打印响应体（截断版本以防止日志过长）
    const maxLogResponseBodyLength = 1024 // 允许记录的最大响应体长度
    responseBodyStr := string(respBody)
    if len(responseBodyStr) > maxLogResponseBodyLength {
        fmt.Printf("[deepSeekPolish] DeepSeek API响应 (truncated to %d chars): %s...\n", maxLogResponseBodyLength, responseBodyStr[:maxLogResponseBodyLength])
    } else {
        fmt.Printf("[deepSeekPolish] DeepSeek API响应: %s\n", responseBodyStr)
    }
    
    // 解析响应
    fmt.Println("[deepSeekPolish] 开始解析响应...")
    var result map[string]interface{}
    if err := json.Unmarshal(respBody, &result); err != nil {
        fmt.Printf("[deepSeekPolish] 解析DeepSeek响应失败: %v\n", err)
        return "", fmt.Errorf("解析DeepSeek响应失败: %w", err)
    }

	// 获取润色后的内容
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		fmt.Println("[deepSeekPolish] DeepSeek响应中未找到有效的choices")
		return "", errors.New("DeepSeek响应中未找到有效的choices")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		fmt.Println("[deepSeekPolish] DeepSeek响应中的choice格式无效")
		return "", errors.New("DeepSeek响应中的choice格式无效")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		fmt.Println("[deepSeekPolish] DeepSeek响应中的message格式无效")
		return "", errors.New("DeepSeek响应中的message格式无效")
	}

	polishedContent, ok := message["content"].(string)
	if !ok {
		fmt.Println("[deepSeekPolish] DeepSeek响应中未找到润色内容")
		return "", errors.New("DeepSeek响应中未找到润色内容")
	}

	fmt.Printf("[deepSeekPolish] 润色成功，润色后内容长度: %d字符\n", len(polishedContent))
	return polishedContent, nil
}

// otherAIPolish 使用其他AI服务润色作文
func (s *DefaultAIService) otherAIPolish(title, content string) (string, error) {
	// 准备请求数据
	requestData := map[string]interface{}{
		"title":   title,
		"content": content,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return "", fmt.Errorf("序列化请求数据失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", s.cfg.AIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.AIKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.AIKey)
	}

	// 发送请求
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送AI请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI服务返回错误状态码: %d", resp.StatusCode)
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析AI响应失败: %w", err)
	}

	// 获取润色后的内容
	polishedContent, ok := result["polished_content"].(string)
	if !ok {
		return "", errors.New("AI响应中未找到润色内容")
	}

	return polishedContent, nil
}

// mockPolish 模拟润色功能（当未配置AI服务时使用）
func (s *DefaultAIService) mockPolish(title, content string) string {
	// 简单的模拟润色逻辑
	polished := content

	// 1. 修正标点符号
	polished = strings.ReplaceAll(polished, "，", "，")
	polished = strings.ReplaceAll(polished, "。", "。")
	polished = strings.ReplaceAll(polished, "？", "？")
	polished = strings.ReplaceAll(polished, "！", "！")

	// 2. 添加一些润色词汇
	polished = strings.ReplaceAll(polished, "很好", "非常棒")
	polished = strings.ReplaceAll(polished, "看到", "目睹")
	polished = strings.ReplaceAll(polished, "说", "表达")

	// 3. 添加结尾评语
	polished = polished + "\n\n【AI点评】这篇作文结构清晰，内容生动。可以适当增加一些细节描写，让文章更加丰富多彩。"

	return polished
}
