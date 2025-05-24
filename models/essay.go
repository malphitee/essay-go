package models

// EssayRequest 作文润色请求结构
type EssayRequest struct {
	Title   string `json:"title"`
	Content string `json:"content" binding:"required"`
}

// EssayResponse 作文润色响应结构
type EssayResponse struct {
	Title           string `json:"title"`
	PolishedContent string `json:"polishedContent"`
}
