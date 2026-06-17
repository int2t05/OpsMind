// Package service 实现 LLM 配置管理业务逻辑。
//
// llm_config_service.go 提供 LLMConfigService 和 LLMConfigManager。
//
// LLMConfigManager 使用 atomic.Value 实现零锁配置热替换（ADR-V2-005）：
//   - GetConfig() 调用 atomic.Value.Load()，零锁开销，适合高并发读取
//   - 更新默认配置后 atomic.Store，新请求即时生效无需重启
package service

import (
	"encoding/json"
	"strings"
	"sync/atomic"

	"opsmind/internal/model"
	"opsmind/internal/repository"
	"opsmind/pkg/errcode"

	"gorm.io/gorm"
)

// =============================================================================
// LLMConfigManager — atomic.Value 热替换
// =============================================================================

// LLMConfigManager 管理当前生效的 LLM 配置（热替换）。
//
// 使用 atomic.Value 存储 *model.LlmConfig，
// GetConfig 是零锁操作，高并发友好。
type LLMConfigManager struct {
	current atomic.Value // *model.LlmConfig
}

// NewLLMConfigManager 创建 Manager 实例。
func NewLLMConfigManager() *LLMConfigManager {
	return &LLMConfigManager{}
}

// GetConfig 返回当前生效的 LLM 配置（零锁读取）。
//
// 返回 nil 表示尚未加载任何配置。
func (m *LLMConfigManager) GetConfig() *model.LlmConfig {
	v := m.current.Load()
	if v == nil {
		return nil
	}
	return v.(*model.LlmConfig)
}

// store 原子替换当前配置。
func (m *LLMConfigManager) store(cfg *model.LlmConfig) {
	// TODO(service/llm_config): store 前应复制 cfg，避免调用方后续修改同一指针导致热配置被意外改变。
	// atomic.Value 只保证指针替换原子，不保证指向对象不可变。
	m.current.Store(cfg)
}

// =============================================================================
// llmConfigRepo — 仓库接口（依赖注入）
// =============================================================================

// llmConfigRepo 定义 LLM 配置仓库接口。
//
// 真实实现为 *repository.LlmConfigRepo，测试使用 mock。
// 接口定义在 service 包内的原因是遵循 Go 的"消费者定义接口"惯例。
type llmConfigRepo interface {
	Create(cfg *model.LlmConfig) error
	FindByID(id int64) (*model.LlmConfig, error)
	FindDefault() (*model.LlmConfig, error)
	List() ([]model.LlmConfig, error)
	Update(cfg *model.LlmConfig) error
	Delete(id int64) error
	ClearDefault() error
}

// =============================================================================
// LLMConfigService — CRUD + 业务规则
// =============================================================================

// LLMConfigService LLM 配置管理服务。
type LLMConfigService struct {
	repo    llmConfigRepo
	manager *LLMConfigManager
	db      *gorm.DB // 非 nil 时提供事务支持（mock 无 DB 时为空）
}

// NewLLMConfigService 创建 LLMConfigService 实例。
//
// 初始化时尝试从 DB 加载默认配置到 manager。
// repo 可以是 *repository.LlmConfigRepo 或测试 mock。
func NewLLMConfigService(repo interface{}) *LLMConfigService {
	svc := &LLMConfigService{
		manager: NewLLMConfigManager(),
	}

	// 适配不同的仓库类型，同时提取 DB 引用用于事务支持
	switch r := repo.(type) {
	case *repository.LlmConfigRepo:
		svc.repo = r
		svc.db = r.DB()
	case llmConfigRepo:
		svc.repo = r
	default:
		// TODO(service/llm_config): 构造函数不应 panic，建议返回 (*LLMConfigService, error)。
		// 启动装配错误应由 main 统一记录并退出，测试也更容易断言。
		panic("NewLLMConfigService: unsupported repo type")
	}

	// 加载默认配置到 manager
	if cfg, err := svc.repo.FindDefault(); err == nil {
		svc.manager.store(cfg)
	}

	return svc
}

// GetManager 返回 Manager 实例（供 ChatService 等高频读取）。
func (s *LLMConfigService) GetManager() *LLMConfigManager {
	return s.manager
}

// =============================================================================
// CRUD 方法
// =============================================================================

// CreateConfig 创建 LLM 配置。
//
// 业务规则：is_default=true 时先清空其他默认配置（保证唯一性）。
// 返回创建的配置对象（含自增 ID），便于调用方获取创建结果。
func (s *LLMConfigService) CreateConfig(name string, providerType int16, baseURL, embeddingBaseURL, apiKey, llmModel, embeddingModel string, maxTokens, vectorDimension int, isDefault bool) (*model.LlmConfig, error) {
	// TODO(service/llm_config): 校验 providerType、baseURL URL 格式、maxTokens/vectorDimension 范围。
	// 目前非法配置可写入数据库，直到问答时才暴露为外部服务错误。
	if strings.TrimSpace(name) == "" {
		return nil, AppError{Code: errcode.ErrParam, Message: "名称不能为空"}
	}

	cfg := &model.LlmConfig{
		Name:             name,
		ProviderType:     providerType,
		BaseURL:          baseURL,
		EmbeddingBaseURL: embeddingBaseURL,
		APIKey:           apiKey,
		LLMModel:         llmModel,
		EmbeddingModel:   embeddingModel,
		MaxTokens:        maxTokens,
		VectorDimension:  vectorDimension,
		IsDefault:        isDefault,
	}

	// ClearDefault + Create 包裹在事务中，保证默认配置唯一性约束不被破坏
	if s.db != nil && isDefault {
		err := s.db.Transaction(func(tx *gorm.DB) error {
			// TODO(service/llm_config): 事务内仍调用 s.repo，真实 repo 持有的是原始 db 而不是 tx。
			// ClearDefault/Create 可能没有进入同一个事务，应创建 txRepo 或让 repo 方法接收 tx。
			if err := s.repo.ClearDefault(); err != nil {
				return AppError{Code: errcode.ErrUnknown, Message: "清空默认配置失败"}
			}
			return s.repo.Create(cfg)
		})
		if err != nil {
			return nil, err
		}
	} else {
		if isDefault {
			if err := s.repo.ClearDefault(); err != nil {
				return nil, AppError{Code: errcode.ErrUnknown, Message: "清空默认配置失败"}
			}
		}
		if err := s.repo.Create(cfg); err != nil {
			return nil, AppError{Code: errcode.ErrUnknown, Message: "创建 LLM 配置失败"}
		}
	}

	// 新默认配置立即生效
	if isDefault {
		// TODO(service/llm_config): 默认配置切换后只更新 Manager 中的 model 配置，没有重建 LLM/Embedding 客户端。
		// ChatService/Handler 注入的 llmClient 仍指向启动时 baseURL，热替换没有真正覆盖 HTTP 目标。
		s.manager.store(cfg)
	}

	return cfg, nil
}

// UpdateConfig 更新 LLM 配置。
//
// 更新为默认时先清空其他默认，更新后立即热替换。
func (s *LLMConfigService) UpdateConfig(cfg *model.LlmConfig) error {
	// TODO(service/llm_config): 更新时 api_key 为空应保留原密钥，当前会覆盖为空。
	// 需要先读旧配置并区分“不传”“传空字符串清空”两种语义。
	// ClearDefault + Update 包裹在事务中
	if s.db != nil && cfg.IsDefault {
		err := s.db.Transaction(func(tx *gorm.DB) error {
			if err := s.repo.ClearDefault(); err != nil {
				return AppError{Code: errcode.ErrUnknown, Message: "清空默认配置失败"}
			}
			return s.repo.Update(cfg)
		})
		if err != nil {
			return err
		}
	} else {
		if cfg.IsDefault {
			if err := s.repo.ClearDefault(); err != nil {
				return AppError{Code: errcode.ErrUnknown, Message: "清空默认配置失败"}
			}
		}
		if err := s.repo.Update(cfg); err != nil {
			return AppError{Code: errcode.ErrUnknown, Message: "更新 LLM 配置失败"}
		}
	}

	// 更新后立即热替换
	if cfg.IsDefault {
		s.manager.store(cfg)
	}

	return nil
}

// ListConfigs 列出全部配置（API Key 脱敏）。
func (s *LLMConfigService) ListConfigs() ([]LlmConfigResponse, error) {
	configs, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	result := make([]LlmConfigResponse, len(configs))
	for i, c := range configs {
		result[i] = LlmConfigResponse{
			ID:               c.ID,
			Name:             c.Name,
			ProviderType:     c.ProviderType,
			BaseURL:          c.BaseURL,
			EmbeddingBaseURL: c.EmbeddingBaseURL,
			APIKey:           maskAPIKey(c.APIKey),
			LLMModel:         c.LLMModel,
			EmbeddingModel:   c.EmbeddingModel,
			MaxTokens:        c.MaxTokens,
			VectorDimension:  c.VectorDimension,
			IsDefault:        c.IsDefault,
		}
	}
	return result, nil
}

// GetConfig 获取单个配置详情。
func (s *LLMConfigService) GetConfig(id int64) (*model.LlmConfig, error) {
	cfg, err := s.repo.FindByID(id)
	if err != nil {
		return nil, AppError{Code: errcode.ErrNotFound, Message: "LLM 配置不存在"}
	}
	return cfg, nil
}

// DeleteConfig 删除 LLM 配置。
//
// 业务规则：不允许删除默认配置。
func (s *LLMConfigService) DeleteConfig(id int64) error {
	// TODO(service/llm_config): 删除前应检查 knowledge_bases 是否引用该配置。
	// 否则知识库可能保留悬空 llm_config_id，后续发布或问答找不到模型参数。
	cfg, err := s.repo.FindByID(id)
	if err != nil {
		return AppError{Code: errcode.ErrNotFound, Message: "LLM 配置不存在"}
	}
	if cfg.IsDefault {
		return AppError{Code: errcode.ErrParam, Message: "不能删除默认配置，请先设置其他配置为默认"}
	}
	return s.repo.Delete(id)
}

// =============================================================================
// 辅助
// =============================================================================

// LlmConfigResponse 列表响应（API Key 脱敏）。
//
// MarshalJSON 在序列化时自动对 APIKey 脱敏，即使 Service 层遗漏 maskAPIKey 调用
// 也不会泄露完整密钥——提供编译期级别的安全保障。
type LlmConfigResponse struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	ProviderType     int16  `json:"provider_type"`
	BaseURL          string `json:"base_url"`
	EmbeddingBaseURL string `json:"embedding_base_url"`
	APIKey           string `json:"api_key"`
	LLMModel         string `json:"llm_model"`
	EmbeddingModel   string `json:"embedding_model"`
	MaxTokens        int    `json:"max_tokens"`
	VectorDimension  int    `json:"vector_dimension"`
	IsDefault        bool   `json:"is_default"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// MarshalJSON 自定义 JSON 序列化，自动对 APIKey 脱敏。
//
// 使用类型别名避免无限递归：先 mask APIKey，再序列化。
func (r LlmConfigResponse) MarshalJSON() ([]byte, error) {
	type Alias LlmConfigResponse
	return json.Marshal(&struct {
		*Alias
		APIKey string `json:"api_key"`
	}{
		Alias:  (*Alias)(&r),
		APIKey: maskAPIKey(r.APIKey),
	})
}

// NewLlmConfigResponse 从 model 构造脱敏后的响应 DTO。
func NewLlmConfigResponse(cfg *model.LlmConfig) LlmConfigResponse {
	return LlmConfigResponse{
		ID:               cfg.ID,
		Name:             cfg.Name,
		ProviderType:     cfg.ProviderType,
		BaseURL:          cfg.BaseURL,
		EmbeddingBaseURL: cfg.EmbeddingBaseURL,
		APIKey:           cfg.APIKey, // MarshalJSON 自动脱敏
		LLMModel:         cfg.LLMModel,
		EmbeddingModel:   cfg.EmbeddingModel,
		MaxTokens:        cfg.MaxTokens,
		VectorDimension:  cfg.VectorDimension,
		IsDefault:        cfg.IsDefault,
		CreatedAt:        cfg.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:        cfg.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// maskAPIKey 对 API Key 进行脱敏处理。
//
// 规则：显示前 4 位 + **** + 后 4 位，长度 ≤ 8 时全部替换为 ****。
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
