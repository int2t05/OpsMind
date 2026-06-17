# 申告状态机 (Ticket State Machine)

> **设计来源：** `model/enums.go` 状态常量 + `service/ticket_service.go` 状态机
> **实现文件：** `handler/ticket.go` → `service/ticket_service.go` → `repository/ticket_repo.go`

---

## 1. 状态转换图

```mermaid
stateDiagram-v2
    [*] --> 待处理 : 提交申告 (CreateTicket)

    待处理 --> 处理中 : start (运维人员)
    处理中 --> 需补充信息 : request_info (运维人员)<br/>supplement_count ≤ 3
    需补充信息 --> 处理中 : supplement (报障人)
    处理中 --> 已解决 : resolve (运维人员)<br/>含回访 detail
    处理中 --> 已关闭 : close (运维人员)
    需补充信息 --> 已关闭 : close (运维人员)
    待处理 --> 已关闭 : close (运维人员/自动)

    已解决 --> [*]
    已关闭 --> [*]

    note right of 需补充信息
        supplement_count +1
        超过 3 次 → 禁止 request_info
        强制走 resolve
    end note

    note left of 已关闭
        7 天超时自动关闭
        TicketAutoCloseJob
        status ∈ {1,2,3} AND
        created_at < NOW() - 7天
    end note
```

---

## 2. 状态转换函数调用链

```mermaid
sequenceDiagram
    autonumber
    actor Operator as 运维人员
    participant H as TicketHandler
    participant S as TicketService
    participant R as TicketRepo
    participant MSG as MessageService
    participant DB as PostgreSQL

    Note over Operator,DB: === PATCH /api/v1/admin/tickets/:id/status ===

    Operator->>H: {action: "resolve", result: "已重置密码", to_knowledge_candidate: true}
    H->>H: c.ShouldBindJSON(&UpdateTicketStatusRequest)
    H->>S: UpdateStatus(ticketID, operatorID, req)

    rect rgb(40, 50, 60)
        Note over S: === TicketService.UpdateStatus 状态机校验 ===
        S->>R: FindByID(ticketID)
        R->>DB: SELECT * FROM tickets WHERE id=?
        DB-->>R: *Ticket
        R-->>S: *Ticket (含当前 status)

        S->>S: switch req.Action { ... }
        Note over S: 校验: 当前状态 → 允许的目标状态

        alt action = "start"
            S->>S: 校验 ticket.Status == 1 (待处理)
            S->>R: UpdateStatus(ticketID, 2)
        else action = "request_info"
            S->>S: 校验 ticket.Status == 2 (处理中)
            S->>S: 校验 ticket.SupplementCount < 3
            S->>R: IncrementSupplementCount(ticketID)
            S->>R: UpdateStatus(ticketID, 3)
            S->>MSG: NotifySupplement(ticketID, userID)
            MSG->>DB: INSERT INTO messages (type='ticket_supplement')
        else action = "resolve"
            S->>S: 校验 ticket.Status == 2 (处理中)
            S->>R: UpdateStatus(ticketID, 4)
        else action = "close"
            S->>S: 允许从任意状态关闭
            S->>R: UpdateStatus(ticketID, 5)
        end

        S->>R: CreateRecord(&TicketRecord{action, content, detail})
        R->>DB: INSERT INTO ticket_records (...)
    end

    S-->>H: nil → response.Success
    H-->>Operator: 200 {code:0}
```

---

## 3. 申告编号生成

```mermaid
flowchart LR
    subgraph TicketNo["TicketService.CreateTicket"]
        A["获取当前日期 YYYYMMDD"] --> B["查询当日已有申告数<br/>SELECT COUNT(*) FROM tickets<br/>WHERE ticket_no LIKE 'TK-YYYYMMDD-%'"]
        B --> C["序号 = count + 1<br/>格式化为 4 位 (0001, 0002...)"]
        C --> D["ticket_no = 'TK-YYYYMMDD-XXXX'"]
    end
```

---

## 4. 补充信息流程 (门户端视角)

```mermaid
sequenceDiagram
    autonumber
    actor Reporter as 报障人
    participant H as TicketHandler
    participant S as TicketService
    participant R as TicketRepo
    participant DB as PostgreSQL

    Note over Reporter,DB: === PATCH /api/v1/portal/tickets/:id/supplement ===

    Reporter->>H: {content: "补充的系统截图如下..."}
    H->>H: c.ShouldBindJSON(&SupplementTicketRequest)
    H->>S: SupplementTicket(ticketID, userID, req)

    S->>R: FindByID(ticketID)
    R-->>S: *Ticket

    S->>S: 校验: ticket.UserID == userID (仅申告人)
    S->>S: 校验: ticket.Status == 3 (需补充信息)

    S->>R: CreateRecord(&TicketRecord{action: "supplement", content})
    R->>DB: INSERT INTO ticket_records (...)

    S->>R: UpdateStatus(ticketID, 2)
    R->>DB: UPDATE tickets SET status=2 WHERE id=?

    S-->>H: nil
    H-->>Reporter: 200 {code:0}
```

---

## 5. Ticket 相关 GORM 模型

```mermaid
classDiagram
    class Ticket {
        +int64 ID
        +string TicketNo
        +int64 UserID
        +string Title
        +string Description
        +int16 Urgency
        +int16 ImpactScope
        +jsonb AffectedSystems
        +string ContactPhone
        +string ContactEmail
        +int16 Status
        +int16 SupplementCount
        +jsonb ChatContext
        +int16 Source
        +time CreatedAt
        +time UpdatedAt
    }

    class TicketRecord {
        +int64 ID
        +int64 TicketID
        +int64 OperatorID
        +string Action
        +string Content
        +jsonb Detail
        +time CreatedAt
    }

    Ticket "1" --> "*" TicketRecord : has records
```
