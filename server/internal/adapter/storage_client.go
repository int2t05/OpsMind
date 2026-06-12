// Package adapter 提供外部服务的适配层。
//
// storage_client.go 定义 StorageClient 接口和 MinIO 实现。
// 所有 MinIO 对象存储操作必须通过此适配层，禁止直接 HTTP 调用。
//
// 接口与 TECH.md §7.3 完全对齐，包含 3 个方法：
// Upload / GetPresignedURL / Delete
//
// Bucket 规划：opsmind-attachments（申告附件）、opsmind-documents（知识文档）。
// 初始化时自动创建 bucket（MakeBucket）。
package adapter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
)

// =============================================================================
// 接口定义
// =============================================================================

// StorageClient 定义对象存储适配器接口。
//
// TicketService 和 KnowledgeService 依赖此接口完成附件上传和文档管理。
// 为什么定义为接口：后续可替换为其他 S3-compatible 存储（如 AWS S3、阿里云 OSS）
// 而不影响上层业务逻辑。
type StorageClient interface {
	// Upload 上传文件到指定 bucket，返回对象 key。
	//
	// key 建议格式："{resource_type}/{timestamp}_{filename}"，如 "tickets/20260610_attach.pdf"
	Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) (string, error)

	// GetPresignedURL 获取对象的预签名下载 URL，expiry 为有效期。
	//
	// 预签名 URL 可直接通过 HTTP GET 访问，无需认证。
	// 典型用途：返回给前端直接下载附件。
	GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)

	// Delete 从指定 bucket 删除对象。
	//
	// 找不到对象时不报错（幂等性），因为删除操作的目标状态是"对象不存在"。
	Delete(ctx context.Context, bucket, key string) error
}

// =============================================================================
// MinIO 实现
// =============================================================================

// MinIOClient 通过 minio-go SDK 实现 StorageClient 接口。
type MinIOClient struct {
	client *minio.Client
}

// NewMinIOClient 创建 MinIOClient 实例，自动确保指定 buckets 存在。
//
// buckets 参数为需确保存在的 bucket 列表（如 "opsmind-attachments", "opsmind-documents"）。
// 为什么在构造函数中确保 bucket 存在：应用启动时一次性创建，避免每个请求都检查。
func NewMinIOClient(client *minio.Client, buckets ...string) *MinIOClient {
	mc := &MinIOClient{client: client}

	for _, bucket := range buckets {
			// TODO(adapter/storage): ensureBucket 失败只 warn，启动仍继续。
			// 如果文档上传是核心能力，应把错误返回给 main，避免运行时才发现 bucket 不可用。
			if err := mc.ensureBucket(context.Background(), bucket); err != nil {
				slog.Warn("ensureBucket 失败", "bucket", bucket, "error", err)
			}
	}

	return mc
}

// ensureBucket 确保 bucket 存在，不存在则创建。
func (c *MinIOClient) ensureBucket(ctx context.Context, bucket string) error {
	exists, err := c.client.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("检查 bucket %s 失败: %w", bucket, err)
	}
	if !exists {
		if err := c.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("创建 bucket %s 失败: %w", bucket, err)
		}
	}
	return nil
}

// =============================================================================
// Upload
// =============================================================================

// Upload 上传文件到 MinIO。
//
// 使用 PutObject API，自动检测 Content-Type。
// 返回与入参相同的 key，方便调用方确认。
func (c *MinIOClient) Upload(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) (string, error) {
	// TODO(adapter/storage): 上传 key 应统一由上层 helper 生成并做路径清理。
	// 原始文件名如果包含 ../、反斜杠或控制字符，可能造成对象命名混乱。
	opts := minio.PutObjectOptions{
		ContentType: contentType,
	}
	if contentType == "" {
		opts.ContentType = "application/octet-stream"
	}

	if _, err := c.client.PutObject(ctx, bucket, key, reader, size, opts); err != nil {
		return "", fmt.Errorf("上传文件失败 [%s/%s]: %w", bucket, key, err)
	}

	return key, nil
}

// =============================================================================
// GetPresignedURL
// =============================================================================

// GetPresignedURL 生成对象的预签名下载 URL。
//
// 使用 PresignedGetObject API，生成有时效的下载链接。
// expiry 建议：附件 1 小时，文档 24 小时。
func (c *MinIOClient) GetPresignedURL(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	reqParams := make(url.Values)
	url, err := c.client.PresignedGetObject(ctx, bucket, key, expiry, reqParams)
	if err != nil {
		return "", fmt.Errorf("生成预签名 URL 失败 [%s/%s]: %w", bucket, key, err)
	}
	return url.String(), nil
}

// =============================================================================
// Delete
// =============================================================================

// Delete 从 MinIO 删除对象。
//
// 使用 RemoveObject API。对象不存在时不报错（MinIO 默认行为即幂等，
// 返回 204 No Content 无论对象是否存在）。
func (c *MinIOClient) Delete(ctx context.Context, bucket, key string) error {
	if err := c.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("删除文件失败 [%s/%s]: %w", bucket, key, err)
	}
	return nil
}
