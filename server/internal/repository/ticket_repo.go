// Package repository 提供申告工单的数据访问层。
//
// TicketRepo 封装 tickets 和 ticket_records 表的 CRUD 操作，供 TicketService 调用。
// 为什么独立于 UserRepo：申告表涉及状态筛选、分页查询、批量关闭等复杂操作，
// 独立 Repo 更利于维护和测试。
package repository

import (
	"time"

	"opsmind/internal/model"

	"gorm.io/gorm"
)

// TicketRepo 申告数据访问
type TicketRepo struct {
	db *gorm.DB
}

// NewTicketRepo 创建 TicketRepo 实例
func NewTicketRepo(db *gorm.DB) *TicketRepo {
	return &TicketRepo{db: db}
}

// =============================================================================
// Ticket
// =============================================================================

// Create 创建申告工单。
//
// 创建后 ticket.ID 会被 GORM 自动填充。
// ticket_no 唯一约束由数据库保证。
func (r *TicketRepo) Create(ticket *model.Ticket) error {
	return r.db.Create(ticket).Error
}

// FindByID 按 ID 查询申告，预加载 User 和 TicketRecords。
//
// 为什么预加载 User：详情页需要显示提交人信息（姓名、手机号）。
// 为什么预加载 TicketRecords：详情页需要展示处理记录时间线。
// TicketRecords 按 created_at 升序排列（最早在前）。
func (r *TicketRepo) FindByID(id int64) (*model.Ticket, error) {
	var ticket model.Ticket
	err := r.db.Preload("User").
		Preload("TicketRecords", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Where("id = ?", id).
		First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

// Update 更新申告全部字段。
//
// 为什么用 Save 而非 Updates：Service 层修改多个字段后全量保存，
// Save 会更新所有字段（包括零值），确保数据一致性。
func (r *TicketRepo) Update(ticket *model.Ticket) error {
	return r.db.Save(ticket).Error
}

// UpdateStatus 更新申告状态。
//
// 为什么单独封装：状态转换是高频操作，仅更新 status 字段避免
// Save 意外覆盖其他字段（如 supplement_count）。
func (r *TicketRepo) UpdateStatus(id int64, status int) error {
	return r.db.Model(&model.Ticket{}).Where("id = ?", id).Update("status", status).Error
}

// IncrementSupplementCount 自增补充信息计数。
//
// 为什么用 gorm.Expr 而非读取后更新：原子操作避免并发竞态条件。
func (r *TicketRepo) IncrementSupplementCount(id int64) error {
	return r.db.Model(&model.Ticket{}).Where("id = ?", id).
		UpdateColumn("supplement_count", gorm.Expr("supplement_count + 1")).Error
}

// ListByUser 分页查询指定用户的申告列表。
//
// 按 id DESC 排序（最新在前），返回总数和列表。
func (r *TicketRepo) ListByUser(userID int64, page, pageSize int) ([]model.Ticket, int64, error) {
	var tickets []model.Ticket
	var total int64

	query := r.db.Model(&model.Ticket{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&tickets).Error; err != nil {
		return nil, 0, err
	}

	return tickets, total, nil
}

// ListAll 分页查询全部申告，支持按状态和紧急程度筛选。
//
// 参数说明：
//   - status: -1 表示不过滤，其他值按精确匹配
//   - urgency: 0 表示不过滤，其他值按精确匹配
func (r *TicketRepo) ListAll(status int, urgency int, page, pageSize int) ([]model.Ticket, int64, error) {
	var tickets []model.Ticket
	var total int64

	query := r.db.Model(&model.Ticket{})
	if status >= 0 {
		query = query.Where("status = ?", status)
	}
	if urgency > 0 {
		query = query.Where("urgency = ?", urgency)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id DESC").Find(&tickets).Error; err != nil {
		return nil, 0, err
	}

	// 批量查询用户名并填充（GORM Preload 在 Count 后不可靠）
	if len(tickets) > 0 {
		ids := make([]int64, len(tickets))
		for i, t := range tickets {
			ids[i] = t.UserID
		}
		var users []model.User
		if err := r.db.Where("id IN ?", ids).Find(&users).Error; err == nil {
			userMap := make(map[int64]model.User, len(users))
			for _, u := range users {
				userMap[u.ID] = u
			}
			for i := range tickets {
				if u, ok := userMap[tickets[i].UserID]; ok {
					tickets[i].User = u
				}
			}
		}
	}

	return tickets, total, nil
}

// AutoCloseTickets 批量关闭超过指定时间的申告，返回关闭数量。
//
// 关闭条件：status IN (1,2,3) AND created_at < olderThan。
// 只关闭待处理(1)、处理中(2)、需补充信息(3)的申告。
// 已解决(4)和已关闭(5)的申告不处理。
// TODO: 魔数 1,2,3,5 应替换为 model 枚举常量。
// 应使用 model.TicketStatusPending/Processing/NeedSupplement/Closed。
// TODO: 批量关闭不创建 TicketRecord，ticket 直接从 1/2/3→5 无 timeline 记录。
// 应在 scheduler 层遍历关闭的 ticket，逐个创建 TicketRecord。
func (r *TicketRepo) AutoCloseTickets(olderThan time.Time) (int64, error) {
	result := r.db.Model(&model.Ticket{}).
		Where("status IN (1,2,3) AND created_at < ?", olderThan).
		Update("status", 5)
	return result.RowsAffected, result.Error
}

// =============================================================================
// TicketRecord
// =============================================================================

// CreateRecord 创建申告处理记录。
//
// 创建后 record.ID 会被 GORM 自动填充。
func (r *TicketRepo) CreateRecord(record *model.TicketRecord) error {
	return r.db.Create(record).Error
}

// FindByTicketID 按申告 ID 查询全部处理记录。
//
// 按 created_at ASC 排序（最早在前），形成处理时间线。
func (r *TicketRepo) FindByTicketID(ticketID int64) ([]model.TicketRecord, error) {
	var records []model.TicketRecord
	err := r.db.Where("ticket_id = ?", ticketID).Order("created_at ASC").Find(&records).Error
	if err != nil {
		return nil, err
	}
	return records, nil
}
