// Package main — 知识库种子数据工具
//
// 直接通过 service 层批量创建文章、审核、发布、生成向量嵌入。
// 解决 HTTP API + Windows 控制台编码引起的 UTF-8 乱码问题。
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"opsmind/internal/adapter"
	"opsmind/internal/config"
	"opsmind/internal/database"
	"opsmind/internal/model"
	"opsmind/internal/rag"
)

type articleSeed struct {
	Title   string
	Content string
}

var articles = []articleSeed{
	{
		Title: "服务器 SSH 连接超时排查指南",
		Content: `服务器 SSH 连接超时通常由以下原因导致：

1. 网络连通性问题
   - 使用 ping 命令测试客户端到服务器的网络是否通畅
   - 检查防火墙规则是否放行 SSH 端口（默认 22）
   - 确认服务器安全组和 iptables 规则

2. SSH 服务状态检查
   - 在服务器控制台执行 systemctl status sshd 检查服务运行状态
   - 查看 /var/log/secure 或 /var/log/auth.log 日志定位错误
   - 确认 sshd 监听端口是否正确：netstat -tlnp | grep 22

3. DNS 解析问题
   - 如果使用域名连接，检查 DNS 解析是否正常
   - 使用 nslookup 或 dig 命令验证域名解析

4. 服务器负载过高
   - 使用 top 或 htop 检查 CPU 和内存使用情况
   - 高负载会导致 SSH 响应超时，需先降低负载

5. 连接数限制
   - 检查 /etc/ssh/sshd_config 中的 MaxStartups 和 MaxSessions
   - 查看当前连接数：ss -t state established | grep :22 | wc -l

处理流程：先确认网络可达性 → 检查服务状态 → 排查系统负载 → 检查配置限制。
如果以上步骤无法解决，建议通过服务器管理控制台（iLO/iDRAC/IPMI）直接登录排查。`,
	},
	{
		Title: "MySQL 数据库连接超时故障处理",
		Content: `MySQL 数据库连接超时是常见的运维故障，按以下步骤排查：

1. 检查 MySQL 服务状态
   - systemctl status mysqld 或 service mysql status
   - 查看错误日志：tail -100 /var/log/mysql/error.log

2. 检查连接数是否达到上限
   - 登录 MySQL 执行：SHOW VARIABLES LIKE 'max_connections';
   - 查看当前连接数：SHOW STATUS LIKE 'Threads_connected';
   - 如果接近上限，临时调高：SET GLOBAL max_connections = 500;

3. 检查连接超时配置
   - connect_timeout：客户端连接握手超时（默认10秒）
   - wait_timeout：非交互连接空闲超时（默认28800秒）
   - interactive_timeout：交互连接空闲超时（默认28800秒）

4. 网络层面排查
   - 检查应用服务器到数据库服务器的网络延迟和丢包
   - 确认防火墙是否放行 MySQL 端口（默认3306）
   - 检查是否触发了 TCP 超时重传

5. 慢查询和锁等待
   - 查看慢查询日志定位耗时 SQL
   - 执行 SHOW PROCESSLIST 查看当前运行的查询
   - 检查是否有锁等待：SELECT * FROM information_schema.INNODB_LOCKS;

6. 连接池配置
   - 检查应用侧连接池最大连接数是否超过数据库限制
   - 验证连接池空闲连接回收策略
   - 确保连接池开启连接有效性检测（testOnBorrow）`,
	},
	{
		Title: "Nginx 502 Bad Gateway 排查流程",
		Content: `Nginx 返回 502 Bad Gateway 表示上游服务无法正常响应，排查步骤：

1. 确认上游服务状态
   - 检查后端应用是否在运行：systemctl status <app-service>
   - 查看应用日志确认是否有异常或崩溃
   - 检查应用端口是否正常监听：netstat -tlnp | grep <port>

2. 检查 Nginx 错误日志
   - 日志位置：/var/log/nginx/error.log
   - 常见错误信息：
     * "connect() failed (111: Connection refused)" — 上游端口未监听
     * "upstream timed out (110: Connection timed out)" — 上游响应超时
     * "no live upstreams" — 所有上游节点不可用

3. 检查 Nginx 超时配置
   - proxy_connect_timeout：连接上游超时（默认60秒）
   - proxy_read_timeout：读取上游响应超时（默认60秒）
   - proxy_send_timeout：发送请求到上游超时（默认60秒）
   - 根据业务需求适当调整，如 proxy_read_timeout 设置为 300s

4. 上游服务性能问题
   - 检查后端应用 CPU 和内存是否过载
   - 查看数据库连接池是否耗尽
   - 确认 PHP-FPM 进程数是否足够（如使用 PHP）

5. 临时应急措施
   - 重启上游服务：systemctl restart <app-service>
   - 重载 Nginx 配置：nginx -s reload
   - 如持续故障，切换到备用节点`,
	},
	{
		Title: "Linux 磁盘空间不足应急处理",
		Content: `Linux 服务器磁盘空间不足会导致服务异常，应急处理流程：

1. 快速定位大文件
   - df -h：查看各分区磁盘使用率
   - du -sh /* | sort -rh | head -20：找出根目录下最大的目录
   - find / -type f -size +100M -exec ls -lh {} \; 2>/dev/null：查找大于100MB的文件

2. 常见可清理的内容
   - 日志文件：/var/log/ 下的 *.log 和 journal 日志
     * journalctl --vacuum-size=500M 清理 systemd 日志
     * find /var/log -name "*.log.*" -mtime +7 -delete 删除7天前归档日志
   - 临时文件：/tmp 和 /var/tmp 下超过7天的文件
   - Docker：docker system prune -a 清理未使用的镜像和容器
   - 包管理器缓存：yum clean all 或 apt-get clean

3. 已删除但未释放的文件
   - lsof | grep deleted：查找被进程持有但已删除的文件
   - 重启持有这些文件句柄的进程或使用 > /proc/<pid>/fd/<fd> 截断

4. 应急扩容
   - 如果是 LVM，可在线扩容：lvextend -L +10G /dev/vg0/root && resize2fs /dev/vg0/root
   - 云服务器可通过控制台扩容云磁盘

5. 预防措施
   - 配置日志轮转：/etc/logrotate.d/
   - 设置磁盘告警阈值（如使用率超过80%触发告警）
   - 定期巡检清理计划任务`,
	},
	{
		Title: "Docker 容器常见故障排查",
		Content: `Docker 容器运行中常见故障及处理方法：

1. 容器无法启动
   - docker logs <container> 查看容器日志
   - docker inspect <container> 查看容器详细配置
   - 检查端口是否冲突：docker ps 确认端口映射
   - 检查挂载卷是否存在且权限正确

2. 容器频繁重启
   - 查看退出码：docker ps -a 中的 STATUS 列
   - 退出码 137：OOM 被杀死，增加内存限制
   - 退出码 139：段错误，应用自身崩溃
   - docker logs --tail 50 <container> 查看最后日志

3. 容器内无法访问外部服务
   - 检查网络模式：docker inspect <container> | grep NetworkMode
   - DNS 解析问题：检查 /etc/docker/daemon.json 中的 dns 配置
   - 防火墙规则：iptables -L -n | grep DOCKER

4. 磁盘空间占用过大
   - docker system df 查看空间使用
   - docker image prune -a 清理未使用的镜像
   - docker volume prune 清理未使用的卷
   - 限制容器日志大小：在 docker-compose.yml 中配置 logging options

5. 资源限制
   - docker stats 查看容器实时资源使用
   - 设置 CPU 和内存限制：--cpus 和 --memory 参数
   - 配置健康检查：HEALTHCHECK 指令`,
	},
	{
		Title: "常见网络故障排查方法",
		Content: `运维中常见网络故障的系统性排查方法：

1. 基本连通性测试
   - ping：测试 ICMP 可达性和延迟
   - traceroute 或 mtr：追踪路由路径，定位丢包节点
   - telnet <host> <port>：测试 TCP 端口是否可达

2. DNS 解析故障
   - nslookup <domain> 或 dig <domain>：测试域名解析
   - 检查 /etc/resolv.conf 中的 DNS 服务器配置
   - 清除 DNS 缓存：systemd-resolve --flush-caches

3. 端口与服务检测
   - netstat -tlnp：查看当前监听的 TCP 端口
   - ss -tlnp：更快的端口查看工具
   - lsof -i :<port>：查看占用特定端口的进程

4. 防火墙排查
   - iptables -L -n -v：查看 iptables 规则
   - firewall-cmd --list-all：查看 firewalld 配置（CentOS 7以上）
   - 临时关闭防火墙测试：systemctl stop firewalld（测试后务必恢复）

5. 抓包分析
   - tcpdump -i eth0 host <target_ip>：捕获与目标 IP 的通信包
   - tcpdump -i eth0 port <port>：捕获特定端口的流量
   - 使用 Wireshark 进行更详细的分析

6. 网络性能
   - iperf3：测试网络带宽
   - ethtool <iface>：查看网卡速率和双工模式
   - iftop：实时查看网络流量`,
	},
}

func main() {
	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}
	cfg.LLM.APIKey = os.Getenv("OPSMIND_LLM_API_KEY")
	if os.Getenv("OPSMIND_EMBEDDING_BASE_URL") != "" {
		cfg.Embedding.BaseURL = os.Getenv("OPSMIND_EMBEDDING_BASE_URL")
	}

	// 初始化数据库
	db, err := database.Init(cfg.Database)
	if err != nil {
		fmt.Fprintf(os.Stderr, "数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化 embedder（使用配置中的 embedding 服务）
	embedBaseURL := cfg.Embedding.BaseURL
	embedAPIKey := cfg.Embedding.APIKey
	if embedBaseURL == "" {
		embedBaseURL = cfg.LLM.BaseURL
	}
	if embedAPIKey == "" {
		embedAPIKey = cfg.LLM.APIKey
	}
	embeddingClient := adapter.NewOpenAIEmbeddingClient(embedBaseURL, embedAPIKey, cfg.Embedding.Model, 30*1_000_000_000)
	embedder := rag.NewEmbedder(embeddingClient, 20)
	chunker := rag.NewChunker(1000, 200)

	ctx := context.Background()

	for i, art := range articles {
		fmt.Printf("[%d/%d] %s\n", i+1, len(articles), art.Title)

		// 1. 创建文章
		article := &model.KnowledgeArticle{
			KBID:    1,
			Title:   art.Title,
			Content: art.Content,
			Status:  1, // 草稿
		}
		if err := db.Create(article).Error; err != nil {
			fmt.Printf("  创建失败: %v\n", err)
			continue
		}
		fmt.Printf("  创建成功 id=%d\n", article.ID)

		// 2. 审核通过 (status 1 → 2)
		article.Status = 2
		db.Save(article)

		// 3. 分块
		chunks := chunker.Split(art.Content)
		if len(chunks) == 0 {
			chunks = splitSimple(art.Content, 800)
		}
		fmt.Printf("  分块: %d 个\n", len(chunks))

		// 4. 生成 embedding
		vectors, dim, err := embedder.Embed(ctx, chunks)
		if err != nil {
			fmt.Printf("  Embedding 失败: %v\n", err)
			continue
		}
		fmt.Printf("  向量: %d 个, 维度=%d\n", len(vectors), dim)

		// 5. 写入知识分块（embedding 是 halfvec 类型，必须用原始 SQL）
		for j, chunk := range chunks {
			vecParts := make([]string, len(vectors[j]))
			for k, v := range vectors[j] {
				vecParts[k] = fmt.Sprintf("%.8f", v)
			}
			vecStr := "[" + strings.Join(vecParts, ",") + "]"
			sql := `INSERT INTO knowledge_chunks
				(article_id, kb_id, content, chunk_index, embedding_model, vector_dimension, embedding, created_at)
				VALUES (?, ?, ?, ?, ?, ?, ?::halfvec, NOW())`
			if err := db.Exec(sql, article.ID, 1, chunk, j, cfg.Embedding.Model, dim, vecStr).Error; err != nil {
				fmt.Printf("  分块写入失败: %v\n", err)
			}
		}

		// 6. 发布 (status 2 → 4)
		article.Status = 4
		article.WordCount = len([]rune(art.Content))
		article.ChunkCount = len(chunks)
		if err := db.Save(article).Error; err != nil {
			fmt.Printf("  发布失败: %v\n", err)
		} else {
			fmt.Printf("  发布完成! 字数=%d, 分块=%d\n", article.WordCount, article.ChunkCount)
		}
	}

	fmt.Println("\n=== 知识库种子数据填充完成 ===")
}

// splitSimple 简单按段落+长度分块（当 chunker 返回空时使用）
func splitSimple(content string, maxLen int) []string {
	var chunks []string
	paragraphs := strings.Split(content, "\n")
	current := ""
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(current)+len(p)+1 < maxLen {
			if current != "" {
				current += "\n"
			}
			current += p
		} else {
			if current != "" {
				chunks = append(chunks, current)
			}
			current = p
		}
	}
	if current != "" {
		chunks = append(chunks, current)
	}
	return chunks
}
