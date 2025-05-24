package models

// User 表示系统用户
type User struct {
	Username string `json:"username"`
	Password string `json:"-"` // 不在JSON中返回密码
	LoggedIn bool   `json:"loggedIn"`
}

// Essay 表示一篇作文，适应 DynamoDB 表结构
type Essay struct {
	Username        string `json:"username" dynamodbav:"username"`                     // 主键，用户名字段
	ID              int64  `json:"id" dynamodbav:"id"`                                // 排序键，自增的ID
	UpdatedAt       string `json:"updated_at" dynamodbav:"updated_at"`                // 更新时间
	DeletedAt       string `json:"deleted_at,omitempty" dynamodbav:"deleted_at,omitempty"` // 软删除时间，如果为空表示未删除
	Title           string `json:"title" dynamodbav:"title"`
	OriginalContent string `json:"originalContent" dynamodbav:"originalContent"`
	PolishedContent string `json:"polishedContent" dynamodbav:"polishedContent"`
	ParentID        int64  `json:"parentId,omitempty" dynamodbav:"parentId,omitempty"` // 父版本的ID，用于跟踪版本关系
}
