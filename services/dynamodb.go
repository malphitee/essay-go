package services

import (
	"context"
	"errors"
	"essay-go/models"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// DynamoDBClient 是 DynamoDB 客户端的包装
type DynamoDBClient struct {
	client    *dynamodb.Client
	tableName string
}

// 全局 DynamoDB 客户端实例
var dynamoDBClient *DynamoDBClient

// InitDynamoDB 初始化 DynamoDB 客户端
func InitDynamoDB(region, tableName string) {
	// 从环境变量获取 AWS 凭证
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	
	// 加载 AWS 配置
	cfgOptions := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	
	// 如果提供了凭证，则使用它们
	if accessKey != "" && secretKey != "" {
		cfgOptions = append(cfgOptions, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKey, secretKey, ""),
		))
	}
	
	cfg, err := awsconfig.LoadDefaultConfig(context.TODO(), cfgOptions...)
	if err != nil {
		log.Printf("无法加载 AWS 配置: %v", err)
		return
	}

	// 创建 DynamoDB 客户端
	client := dynamodb.NewFromConfig(cfg)

	// 创建并存储客户端实例
	dynamoDBClient = &DynamoDBClient{
		client:    client,
		tableName: tableName,
	}

	// 确保表存在
	ensureTableExists(client, tableName)
}

// GetDynamoDBClient 返回 DynamoDB 客户端实例
func GetDynamoDBClient() *DynamoDBClient {
	return dynamoDBClient
}

// 创建标记文件路径
const initFlagFile = "data/dynamodb_initialized.flag"

// checkInitialized 检查是否已经初始化过表
func checkInitialized() bool {
	// 检查标记文件是否存在
	_, err := os.Stat(initFlagFile)
	return err == nil
}

// markInitialized 标记表已经初始化
func markInitialized() error {
	// 确保 data 目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		return err
	}
	
	// 创建标记文件
	f, err := os.Create(initFlagFile)
	if err != nil {
		return err
	}
	defer f.Close()
	
	// 写入初始化时间
	_, err = f.WriteString(fmt.Sprintf("DynamoDB table initialized at %s\n", time.Now().Format(time.RFC3339)))
	return err
}

// deleteTable 删除现有表
func deleteTable(client *dynamodb.Client, tableName string) error {
	log.Printf("尝试删除表 %s...", tableName)
	_, err := client.DeleteTable(context.TODO(), &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})

	// 检查错误类型，如果表不存在，直接返回
	if err != nil {
		// 检查是否是表不存在的错误
		var notFoundErr *types.ResourceNotFoundException
		if ok := errors.As(err, &notFoundErr); ok {
			log.Printf("表 %s 不存在，无需删除", tableName)
			return nil
		}
		log.Printf("删除表失败: %v", err)
		return err
	}

	log.Printf("表 %s 删除成功", tableName)

	// 跳过等待表删除完成的步骤，直接继续
	log.Printf("跳过等待表删除完成的步骤，直接继续")
	
	// 等待一小段时间，给 AWS 一些时间处理删除请求
	time.Sleep(2 * time.Second)
	
	return nil
}

// ensureTableExists 检查表是否存在，如果不存在则创建表
func ensureTableExists(client *dynamodb.Client, tableName string) {
	// 强制重新创建表，删除初始化标记文件
	// 删除标记文件
	os.Remove(initFlagFile)
	
	// 删除现有表并重新创建
	deleteTable(client, tableName)

	// 检查表是否存在
	_, err := client.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})

	if err != nil {
		log.Printf("表 %s 不存在，错误详情: %v", tableName, err)
		
		// 创建表
		log.Printf("尝试创建表 %s...", tableName)
		_, createErr := client.CreateTable(context.TODO(), &dynamodb.CreateTableInput{
			TableName: aws.String(tableName),
			AttributeDefinitions: []types.AttributeDefinition{
				{
					AttributeName: aws.String("username"),
					AttributeType: types.ScalarAttributeTypeS,
				},
				{
					AttributeName: aws.String("id"),
					AttributeType: types.ScalarAttributeTypeN,
				},
			},
			KeySchema: []types.KeySchemaElement{
				{
					AttributeName: aws.String("username"),
					KeyType:       types.KeyTypeHash,
				},
				{
					AttributeName: aws.String("id"),
					KeyType:       types.KeyTypeRange,
				},
			},
			ProvisionedThroughput: &types.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(5),
				WriteCapacityUnits: aws.Int64(5),
			},
		})

		if createErr != nil {
			log.Printf("创建表失败: %v", createErr)
			return
		}

		// 等待表创建完成
		log.Printf("等待表创建完成...")
		waiter := dynamodb.NewTableExistsWaiter(client)
		if err := waiter.Wait(context.TODO(), &dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		}, 2*time.Minute); err != nil {
			log.Printf("等待表创建完成失败: %v", err)
			return
		}

		log.Printf("表 %s 创建成功", tableName)
		
		// 标记已初始化
		if err := markInitialized(); err != nil {
			log.Printf("标记初始化状态失败: %v", err)
		}
	} else {
		log.Printf("表 %s 已存在", tableName)
	}
}

// getMaxID 获取用户的最大 ID
func (db *DynamoDBClient) getMaxID(username string) (int64, error) {
	// 查询用户的所有作文
	resp, err := db.client.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(db.tableName),
		KeyConditionExpression: aws.String("username = :username"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":username": &types.AttributeValueMemberS{Value: username},
		},
		Limit: aws.Int32(1), // 只需要一个结果
		ScanIndexForward: aws.Bool(false), // 降序排序，最大的 ID 在前面
	})

	if err != nil {
		log.Printf("获取最大 ID 失败: %v", err)
		return 0, err
	}

	// 如果没有结果，返回 0
	if len(resp.Items) == 0 {
		return 0, nil
	}

	// 解析结果
	var essay models.Essay
	err = attributevalue.UnmarshalMap(resp.Items[0], &essay)
	if err != nil {
		log.Printf("解析作文失败: %v", err)
		return 0, err
	}

	return essay.ID, nil
}

// SaveEssay 保存作文到 DynamoDB
func (db *DynamoDBClient) SaveEssay(essay models.Essay) error {
	// 确保更新时间格式正确
	if essay.UpdatedAt == "" {
		essay.UpdatedAt = time.Now().Format(time.RFC3339)
	}

	// 确保 username 字段不为空
	if essay.Username == "" {
		log.Printf("错误: 用户名不能为空")
		return fmt.Errorf("用户名不能为空")
	}

	// 如果 ID 为 0，获取新的 ID
	if essay.ID == 0 {
		maxID, err := db.getMaxID(essay.Username)
		if err != nil {
			log.Printf("获取最大 ID 失败: %v", err)
			return err
		}
		essay.ID = maxID + 1
		log.Printf("为新作文分配 ID: %d", essay.ID)
	}

	log.Printf("尝试保存作文, ID: %d, 标题: %s, 用户名: %s, 更新时间: %s", 
		essay.ID, essay.Title, essay.Username, essay.UpdatedAt)

	// 将作文转换为 DynamoDB 属性值
	item, err := attributevalue.MarshalMap(essay)
	if err != nil {
		log.Printf("将作文转换为 DynamoDB 属性值失败: %v", err)
		return err
	}

	// 保存到 DynamoDB
	_, err = db.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(db.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("保存作文到 DynamoDB 失败: %v", err)
	} else {
		log.Printf("作文保存成功, 用户名: %s, ID: %d", essay.Username, essay.ID)
	}

	return err
}

// GetEssaysByUsername 根据用户名获取所有作文
func (db *DynamoDBClient) GetEssaysByUsername(username string) ([]models.Essay, error) {
	log.Printf("尝试获取用户 %s 的所有作文", username)
	
	// 使用 Query 操作直接查询主键
	resp, err := db.client.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(db.tableName),
		KeyConditionExpression: aws.String("username = :username"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":username": &types.AttributeValueMemberS{Value: username},
		},
		// 按 ID 降序排序，最新的在前面
		ScanIndexForward: aws.Bool(false),
	})

	if err != nil {
		log.Printf("从 DynamoDB 获取作文失败: %v", err)
		
		// 检查是否是表不存在的错误
		var notFoundErr *types.ResourceNotFoundException
		if ok := errors.As(err, &notFoundErr); ok {
			log.Printf("表不存在，尝试初始化表...")
			
			// 删除标记文件，强制重新初始化
			os.Remove(initFlagFile)
			
			// 初始化表
			ensureTableExists(db.client, db.tableName)
			
			// 等待表创建完成
			time.Sleep(5 * time.Second)
			
			// 重新查询
			return db.GetEssaysByUsername(username)
		}
		
		return nil, err
	}

	log.Printf("从 DynamoDB 获取到 %d 条记录", len(resp.Items))
	
	// 解析结果
	var essays []models.Essay
	err = attributevalue.UnmarshalListOfMaps(resp.Items, &essays)
	if err != nil {
		log.Printf("解析 DynamoDB 结果失败: %v", err)
		return nil, err
	}

	// 过滤掉已软删除的作文
	var activeEssays []models.Essay
	for _, essay := range essays {
		if essay.DeletedAt == "" {
			activeEssays = append(activeEssays, essay)
		}
	}

	log.Printf("成功获取并解析了 %d 篇作文，其中有效作文 %d 篇", 
		len(essays), len(activeEssays))
	return activeEssays, nil
}

// DeleteEssay 从 DynamoDB 软删除作文
func (db *DynamoDBClient) DeleteEssay(username string, essayID int64) error {
	log.Printf("尝试软删除作文, 用户名: %s, ID: %d", username, essayID)
	
	// 首先获取该作文
	resp, err := db.client.Query(context.TODO(), &dynamodb.QueryInput{
		TableName:              aws.String(db.tableName),
		KeyConditionExpression: aws.String("username = :username AND id = :id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":username": &types.AttributeValueMemberS{Value: username},
			":id":       &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", essayID)},
		},
		Limit: aws.Int32(1),
	})

	if err != nil {
		log.Printf("获取作文失败: %v", err)
		return err
	}

	if len(resp.Items) == 0 {
		log.Printf("未找到要删除的作文, 用户名: %s, ID: %d", username, essayID)
		return fmt.Errorf("未找到要删除的作文")
	}

	// 解析作文
	var essay models.Essay
	err = attributevalue.UnmarshalMap(resp.Items[0], &essay)
	if err != nil {
		log.Printf("解析作文失败: %v", err)
		return err
	}

	// 设置删除时间
	essay.DeletedAt = time.Now().Format(time.RFC3339)

	// 将修改后的作文保存回 DynamoDB
	item, err := attributevalue.MarshalMap(essay)
	if err != nil {
		log.Printf("将作文转换为 DynamoDB 属性值失败: %v", err)
		return err
	}

	// 更新记录
	_, err = db.client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: aws.String(db.tableName),
		Item:      item,
	})

	if err != nil {
		log.Printf("软删除作文失败: %v", err)
	} else {
		log.Printf("作文软删除成功, 用户名: %s, ID: %d", username, essayID)
	}
	
	return err
}
