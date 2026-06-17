# 申告生命周期 — 函数级调用链

> 代码基准：`handler/ticket.go` → `service/ticket_service.go` → `repository/ticket_repo.go` / `service/scheduler.go`
> 更新于 2026-06-12 — 反映 AutoClose 上移至 Service + TxManager 重构

## 1. 完整生命周期（创建→处理→关闭）

```mermaid
sequenceDiagram
    actor R as 报障人
    actor O as 运维人员
    participant H as TicketHandler<br/>handler/ticket.go
    participant TS as TicketService<br/>service/ticket_service.go
    participant TR as TicketRepo<br/>repository/ticket_repo.go
    participant MS as MessageService.NotifySupplement<br/>service/message_service.go
    participant DB as PostgreSQL

    Note over R,DB: ====== 1. 创建申告 ======
    R->>H: POST /api/v1/portal/tickets<br/>{title, description, urgency, contact_phone}
    H->>H: c.ShouldBindJSON(&CreateTicketRequest)
    H->>H: getCurrentUserID(c) → (userID, bool)
    H->>TS: CreateTicket(req, userID)

    TS->>TS: 参数校验: title/description/contact_phone 必填
    TS->>TS: urgency ∈ [1,3] 范围校验
    TS->>TS: ticket_no = TK-YYYYMMDD-XXXXXX<br/>(日期 + time.Now().UnixNano()%1000000)
    TS->>TR: Create(&Ticket{Status: Pending=1, Source: Portal=1})
    TR->>DB: INSERT INTO tickets
    DB-->>TR: ticket.ID
    TS-->>H: nil
    H-->>R: 200 {message: "申告已创建"}

    Note over R,DB: ====== 2. 开始处理 (start) ======
    O->>H: PATCH /api/v1/admin/tickets/:id/status<br/>{action: "start"}
    H->>H: parseID(c, "id")
    H->>TS: UpdateStatus(id, operatorID, {Action: "start"})

    TS->>TR: FindByID(id)
    TR->>DB: SELECT * FROM tickets WHERE id=?
    DB-->>TR: *Ticket{Status: Pending=1}

    TS->>TS: 状态机校验: Pending → Processing ✓
    TS->>TS: txManager.Transaction(func(tx){...})

    Note over TS,DB: 事务内:
    TS->>TR: Update(ticket) → status=Processing(2)
    TR->>DB: UPDATE tickets SET status=2 WHERE id=?
    TS->>TR: CreateRecord(&TicketRecord{action:"start", content})
    TR->>DB: INSERT INTO ticket_records

    TS-->>H: nil
    H-->>O: 200

    Note over R,DB: ====== 3. 要求补充信息 (request_info) ======
    O->>H: PATCH .../status {action: "request_info"}
    H->>TS: UpdateStatus(id, operatorID, {Action: "request_info"})

    TS->>TR: FindByID(id)
    TS->>TS: 状态机校验: Processing → NeedSupplement ✓
    TS->>TS: txManager.Transaction(func(tx){...})
    Note over TS,DB: 事务内:
    TS->>TR: Update(ticket) → status=NeedSupplement(3)
    TS->>TR: CreateRecord({action:"request_info"})

    TS->>MS: NotifySupplement(ticketID, userID)
    MS->>TR: Create(&Message{type:"ticket_supplement", ...})
    TR->>DB: INSERT INTO messages

    TS-->>H: nil

    Note over R,DB: ====== 4. 补充信息 (supplement) ======
    R->>H: PATCH /api/v1/portal/tickets/:id/supplement<br/>{content}
    H->>TS: SupplementTicket(id, userID, {Content})

    TS->>TR: FindByID(id)
    TS->>TS: 状态机: NeedSupplement → Processing ✓
    TS->>TR: IncrementSupplementCount(id)
    Note over TR,DB: UPDATE tickets SET supplement_count = supplement_count + 1<br/>WHERE id=? AND supplement_count < 3 (原子检查)
    TS->>TR: CreateRecord({action:"supplement", content})
    TS-->>H: nil

    Note over R,DB: ====== 5. 解决 (resolve) ======
    O->>H: PATCH .../status {action: "resolve"}
    H->>TS: UpdateStatus(id, operatorID, {Action: "resolve"})

    TS->>TR: FindByID(id)
    TS->>TS: 状态机: Processing → Resolved ✓
    TS->>TS: txManager.Transaction(func(tx){...})
    TS->>TR: Update(ticket) → status=Resolved(4)
    TS->>TR: CreateRecord({action:"resolve"})
    TS-->>H: nil

    Note over R,DB: ====== 6. 关闭 (close) ======
    O->>H: PATCH .../status {action: "close"}
    H->>TS: UpdateStatus(id, operatorID, {Action: "close"})

    TS->>TS: 状态机: Resolved → Closed ✓
    TS->>TS: txManager.Transaction(func(tx){...})
    TS->>TR: Update(ticket) → status=Closed(5)
    TS->>TR: CreateRecord({action:"close"})
    TS-->>H: nil
```

## 2. 自动关闭 (AutoClose) — Service 层编排

```mermaid
sequenceDiagram
    participant Sched as Scheduler.runAutoCloseLoop<br/>service/scheduler.go:55
    participant TS as TicketService.AutoClose<br/>service/ticket_service.go:407
    participant TR as TicketRepo.AutoCloseTickets<br/>repository/ticket_repo.go:158
    participant DB as PostgreSQL

    Note over Sched: 每小时触发一次 (time.NewTicker(1*Hour))

    Sched->>TS: AutoClose(time.Now().Add(-7*24*Hour))

    Note over TS: === 1. 纯数据操作：查询 + 批量关闭 ===
    TS->>TR: AutoCloseTickets(olderThan)
    TR->>DB: SELECT id FROM tickets<br/>WHERE status IN (1,2,3) AND created_at < ?
    DB-->>TR: []int64 IDs

    alt len(ids) > 0
        TR->>DB: UPDATE tickets SET status=5 WHERE id IN (...)
        DB-->>TR: rows affected
    end

    TR-->>TS: []int64 ids

    Note over TS: === 2. 事务编排：创建 TicketRecord ===
    alt len(ids) > 0
        TS->>TS: txManager.Transaction(func(tx){...})
        Note over TS,DB: 事务内: 为每个 ticket 创建 auto_close 记录
        loop 每个 id
            TS->>DB: INSERT INTO ticket_records<br/>(ticket_id, operator_id=0, action="auto_close",<br/>content="系统自动关闭：申告超过 7 天未处理")
        end
    end

    TS-->>Sched: int64 count
```

## 3. 状态机转换规则

```mermaid
stateDiagram-v2
    [*] --> Pending : CreateTicket()

    Pending --> Processing : UpdateStatus("start")
    Processing --> NeedSupplement : UpdateStatus("request_info")<br/>→ NotifySupplement()
    NeedSupplement --> Processing : SupplementTicket()<br/>(supplement_count < 3 原子检查)
    Processing --> Resolved : UpdateStatus("resolve")
    Resolved --> Closed : UpdateStatus("close")

    Pending --> Closed : AutoClose(olderThan)<br/>→ txManager.Transaction()
    Processing --> Closed : AutoClose(olderThan)
    NeedSupplement --> Closed : AutoClose(olderThan)

    note right of NeedSupplement
        supplement_count ≥ 3 时
        补充操作被拒绝
    end note

    note right of Closed
        AutoClose 在 Service 层
        通过 TxManager 编排事务
        Repo 只做纯数据 UPDATE
    end note
```
