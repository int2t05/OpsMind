// Package adapter 提供外部服务的适配层。
//
// rag_client.go 定义 RagClient 接口和 AnythingLLM HTTP 实现。
// 所有 AnythingLLM API 调用必须通过此适配层，禁止直接 HTTP 调用。
//
// 接口与 TECH.md §7.1 完全对齐，包含 4 个方法：
// Query / SyncDocument / DisableDocument / CreateWorkspace
package adapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// =============================================================================
// 接口定义
// =============================================================================

// RagClient 定义 RAG 服务适配器接口。
//
// KnowledgeService 依赖此接口完成 AnythingLLM workspace 和文档管理。
type RagClient interface {
	// Query 向 AnythingLLM 发送问答请求，返回答案和来源。
	Query(ctx context.Context, req RAGQueryRequest) (*RAGQueryResponse, error)
	// SyncDocument 同步文档到 AnythingLLM 工作区。
	SyncDocument(ctx context.Context, req RAGSyncRequest) (*RAGSyncResponse, error)
	// DisableDocument 从 AnythingLLM 工作区停用并删除文档。
	DisableDocument(ctx context.Context, req RAGDisableRequest) error
	// CreateWorkspace 在 AnythingLLM 中创建工作区。
	CreateWorkspace(ctx context.Context, req RAGCreateWorkspaceRequest) (*RAGCreateWorkspaceResponse, error)
}

// =============================================================================
// 请求/响应类型
// =============================================================================

// RAGQueryRequest 问答请求参数。
type RAGQueryRequest struct {
	WorkspaceSlug string `json:"-"`        // 工作区 slug（路径参数）
	Question      string `json:"message"`   // 用户问题
	TopK          int    `json:"top_k,omitempty"` // 检索 Top K
}

// RAGQueryResponse 问答响应。
type RAGQueryResponse struct {
	Answer     string       `json:"answer"`     // 答案文本（映射自 textResponse）
	Confidence float64      `json:"confidence"`  // 置信度（max(sources[].score)）
	Sources    []RAGSource  `json:"sources"`    // 来源列表
	Error      string       `json:"error,omitempty"` // 错误信息（服务器返回 non-null error 时）
}

// RAGSource RAG 检索来源。
type RAGSource struct {
	DocName      string  `json:"doc_name"`      // 文档名称（映射自 sources[].title）
	ChunkContent string  `json:"chunk_content"` // 切片内容（映射自 sources[].text）
	Confidence   float64 `json:"confidence"`    // 该来源的置信度
}

// RAGSyncRequest 文档同步请求。
type RAGSyncRequest struct {
	WorkspaceSlug string `json:"-"`       // 工作区 slug（路径参数）
	Title         string `json:"title"`   // 文档标题
	Content       string `json:"content"` // 文档内容（raw-text 模式）
	Mode          string `json:"mode"`    // raw-text（结构化 FAQ）或 file-upload
}

// RAGSyncResponse 文档同步响应。
type RAGSyncResponse struct {
	DocumentLocation string `json:"document_location"` // 文档在 AnythingLLM 中的位置
}

// RAGDisableRequest 文档停用请求。
type RAGDisableRequest struct {
	WorkspaceSlug     string   `json:"-"`               // 工作区 slug（路径参数）
	DocumentLocations []string `json:"document_locations"` // 要删除的文档位置列表
}

// RAGCreateWorkspaceRequest 工作区创建请求。
type RAGCreateWorkspaceRequest struct {
	Name string `json:"name"` // 工作区名称
}

// RAGCreateWorkspaceResponse 工作区创建响应。
type RAGCreateWorkspaceResponse struct {
	Slug string `json:"slug"` // 工作区 slug
	ID   int64  `json:"id"`   // 工作区 ID
}

// =============================================================================
// AnythingLLM HTTP 实现
// =============================================================================

// AnythingLLMConfig AnythingLLM 客户端配置。
type AnythingLLMConfig struct {
	BaseURL        string
	APIKey         string
	TimeoutSeconds int
}

// AnythingLLMClient 通过 HTTP 调用 AnythingLLM API。
//
// 为什么使用标准 net/http 而非第三方 SDK：
// AnythingLLM 没有官方 Go SDK，HTTP API 足够简单，标准库即可满足需求。
type AnythingLLMClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewAnythingLLMClient 创建 AnythingLLMClient 实例。
func NewAnythingLLMClient(cfg AnythingLLMConfig) *AnythingLLMClient {
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 20 * time.Second
	}

	return &AnythingLLMClient{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// BaseURL 返回客户端的基础 URL（用于测试）。
func (c *AnythingLLMClient) BaseURL() string {
	return c.baseURL
}

// =============================================================================
// Query — POST /api/v1/workspace/{slug}/chat
// =============================================================================

// anythingLLMChatRequest AnythingLLM chat API 请求体。
type anythingLLMChatRequest struct {
	Message string `json:"message"`
	Mode    string `json:"mode"` // "query"
}

// anythingLLMChatResponse AnythingLLM chat API 响应体。
type anythingLLMChatResponse struct {
	ID           string                   `json:"id"`
	Type         string                   `json:"type"`
	Close        bool                     `json:"close"`
	Error        *string                  `json:"error"`
	TextResponse string                   `json:"textResponse"`
	Sources      []anythingLLMChatSource  `json:"sources"`
}

type anythingLLMChatSource struct {
	Title   string  `json:"title"`
	Text    string  `json:"text"`
	Score   float64 `json:"score"`
	URL     string  `json:"url"`
	DocName string  `json:"doc_name"`
}

// Query 发送问答请求并映射响应字段。
func (c *AnythingLLMClient) Query(ctx context.Context, req RAGQueryRequest) (*RAGQueryResponse, error) {
	url := fmt.Sprintf("%s/v1/workspace/%s/chat", c.baseURL, req.WorkspaceSlug)

	chatReq := anythingLLMChatRequest{
		Message: req.Question,
		Mode:    "query",
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 AnythingLLM 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("AnythingLLM 返回 HTTP %d", resp.StatusCode)
	}

	var chatResp anythingLLMChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 映射响应字段
	result := &RAGQueryResponse{
		Answer: chatResp.TextResponse,
	}

	// 服务器返回 error
	if chatResp.Error != nil {
		result.Error = *chatResp.Error
		return result, nil
	}

	// 映射 sources
	if len(chatResp.Sources) > 0 {
		result.Sources = make([]RAGSource, len(chatResp.Sources))
		var maxScore float64
		for i, s := range chatResp.Sources {
			result.Sources[i] = RAGSource{
				DocName:      s.DocName,
				ChunkContent: s.Text,
				Confidence:   s.Score,
			}
			if s.Score > maxScore {
				maxScore = s.Score
			}
		}
		result.Confidence = maxScore
	}

	return result, nil
}

// =============================================================================
// CreateWorkspace — POST /api/v1/workspace/new
// =============================================================================

// anythingLLMWorkspaceResponse AnythingLLM workspace/new 响应体。
type anythingLLMWorkspaceResponse struct {
	Workspace struct {
		ID   int64  `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	} `json:"workspace"`
	Error *string `json:"error"`
}

// CreateWorkspace 在 AnythingLLM 中创建工作区。
func (c *AnythingLLMClient) CreateWorkspace(ctx context.Context, req RAGCreateWorkspaceRequest) (*RAGCreateWorkspaceResponse, error) {
	url := fmt.Sprintf("%s/v1/workspace/new", c.baseURL)

	body, err := json.Marshal(map[string]string{"name": req.Name})
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 AnythingLLM 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("AnythingLLM 返回 HTTP %d", resp.StatusCode)
	}

	var wsResp anythingLLMWorkspaceResponse
	if err := json.NewDecoder(resp.Body).Decode(&wsResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if wsResp.Error != nil {
		return nil, fmt.Errorf("创建 workspace 失败: %s", *wsResp.Error)
	}

	return &RAGCreateWorkspaceResponse{
		Slug: wsResp.Workspace.Slug,
		ID:   wsResp.Workspace.ID,
	}, nil
}

// =============================================================================
// SyncDocument — POST /api/v1/document/raw-text
// =============================================================================

// anythingLLMSyncResponse AnythingLLM sync 响应体。
type anythingLLMSyncResponse struct {
	Success  bool    `json:"success"`
	Error   *string `json:"error"`
	Document struct {
		Location string `json:"location"`
	} `json:"document"`
}

// SyncDocument 同步文档到 AnythingLLM。
//
// 使用 POST /api/v1/document/raw-text 方式，
// 将 title/content 作为文档正文写入指定 workspace。
func (c *AnythingLLMClient) SyncDocument(ctx context.Context, req RAGSyncRequest) (*RAGSyncResponse, error) {
	url := fmt.Sprintf("%s/v1/document/raw-text", c.baseURL)

	// AnythingLLM raw-text API 要求 metadata.title 为必填字段（见 anything-llm source:
	// server/endpoints/api/document/index.js:569-576）
	payload := map[string]interface{}{
		"textContent":   req.Content,
		"workspaceSlug": req.WorkspaceSlug,
		"metadata": map[string]string{
			"title": req.Title,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("请求 AnythingLLM 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AnythingLLM 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var syncResp anythingLLMSyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&syncResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if syncResp.Error != nil {
		return nil, fmt.Errorf("同步文档失败: %s", *syncResp.Error)
	}

	return &RAGSyncResponse{
		DocumentLocation: syncResp.Document.Location,
	}, nil
}

// =============================================================================
// DisableDocument — POST /api/v1/workspace/{slug}/update-embeddings
// =============================================================================

// anythingLLMDisableResponse AnythingLLM update-embeddings 响应体。
type anythingLLMDisableResponse struct {
	Success bool    `json:"success"`
	Error   *string `json:"error"`
}

// DisableDocument 从 AnythingLLM 中删除文档向量。
func (c *AnythingLLMClient) DisableDocument(ctx context.Context, req RAGDisableRequest) error {
	url := fmt.Sprintf("%s/v1/workspace/%s/update-embeddings", c.baseURL, req.WorkspaceSlug)

	payload := map[string]interface{}{
		"deletes": req.DocumentLocations,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	c.setHeaders(httpReq)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("请求 AnythingLLM 失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("AnythingLLM 返回 HTTP %d", resp.StatusCode)
	}

	var disableResp anythingLLMDisableResponse
	if err := json.NewDecoder(resp.Body).Decode(&disableResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if disableResp.Error != nil {
		return fmt.Errorf("停用文档失败: %s", *disableResp.Error)
	}

	return nil
}

// =============================================================================
// 辅助方法
// =============================================================================

// setHeaders 设置通用请求头。
func (c *AnythingLLMClient) setHeaders(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
}
