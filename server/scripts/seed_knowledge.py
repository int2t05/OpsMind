"""Seed knowledge base with practical IT运维 articles, then publish and embed."""
import json, requests, subprocess, sys, time

BASE = "http://localhost:8080/api/v1"
EMBED_API = "https://api.siliconflow.cn/v1/embeddings"
EMBED_KEY = "Bearer sk-pclemkqvcnntefyrtgehdxznpqazgpvqwwazznarfznfcprg"
EMBED_MODEL = "BAAI/bge-m3"

ARTICLES = [
    {
        "title": "服务器 SSH 连接超时排查指南",
        "content": """服务器 SSH 连接超时通常由以下原因导致：

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

处理流程：先确认网络可达性 → 检查服务状态 → 排查系统负载 → 检查配置限制。如果以上步骤无法解决，建议通过服务器管理控制台（iLO/iDRAC/IPMI）直接登录排查。"""
    },
    {
        "title": "MySQL 数据库连接超时故障处理",
        "content": """MySQL 数据库连接超时是常见的运维故障，按以下步骤排查：

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
   - 确保连接池开启连接有效性检测（testOnBorrow）"""
    },
    {
        "title": "Nginx 502 Bad Gateway 排查流程",
        "content": """Nginx 返回 502 Bad Gateway 表示上游服务无法正常响应，排查步骤：

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
   - 根据业务需求适当调整，如 proxy_read_timeout 300s

4. 上游服务性能问题
   - 检查后端应用 CPU/内存是否过载
   - 查看数据库连接池是否耗尽
   - 确认 PHP-FPM 进程数是否足够（如使用 PHP）

5. 临时应急措施
   - 重启上游服务：systemctl restart <app-service>
   - 重载 Nginx 配置：nginx -s reload
   - 如持续故障，切换到备用节点"""
    },
    {
        "title": "Linux 磁盘空间不足应急处理",
        "content": """Linux 服务器磁盘空间不足会导致服务异常，应急处理流程：

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
   - 定期巡检清理计划任务"""
    },
    {
        "title": "Docker 容器常见故障排查",
        "content": """Docker 容器运行中常见故障及处理方法：

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
   - 设置 CPU/内存限制：--cpus 和 --memory 参数
   - 配置健康检查：HEALTHCHECK 指令"""
    },
    {
        "title": "常见网络故障排查方法",
        "content": """运维中常见网络故障的系统性排查方法：

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
   - firewall-cmd --list-all：查看 firewalld 配置（CentOS 7+）
   - 临时关闭防火墙测试：systemctl stop firewalld（测试后务必恢复）

5. 抓包分析
   - tcpdump -i eth0 host <target_ip>：捕获与目标 IP 的通信包
   - tcpdump -i eth0 port <port>：捕获特定端口的流量
   - 使用 Wireshark 进行更详细的分析

6. 网络性能
   - iperf3：测试网络带宽
   - ethtool <iface>：查看网卡速率和双工模式
   - iftop：实时查看网络流量"""
    },
    {
        "title": "系统服务异常重启排查",
        "content": """生产环境服务异常重启的排查步骤：

1. 确定重启时间和频率
   - systemctl status <service> 查看最近启动时间
   - journalctl -u <service> --since "1 hour ago" 查看最近日志
   - 确认是手动重启、计划任务还是异常崩溃

2. 内存不足（OOM）
   - dmesg | grep -i "out of memory" 查看 OOM Killer 记录
   - grep -i "killed process" /var/log/messages
   - 调整应用内存配置或增加服务器内存

3. CPU 过载
   - 查看 sar -u 历史 CPU 使用率
   - 检查是否有定时任务（crontab）在服务重启时间点运行
   - top 或 htop 实时观察 CPU 使用模式

4. 应用自身 Bug
   - 查看应用错误日志中的异常堆栈
   - 检查是否有死锁、无限循环或内存泄漏
   - 分析 core dump 文件（如果配置了）

5. 依赖服务故障
   - 数据库连接断开导致应用健康检查失败
   - Redis 等缓存服务不可用触发重启策略
   - 配置文件语法错误导致启动失败

6. 系统级问题
   - 系统时间跳变（NTP 同步）
   - 文件句柄耗尽：ulimit -n 查看限制
   - 磁盘写满导致日志写入失败"""
    },
    {
        "title": "SSL 证书过期和配置问题处理",
        "content": """SSL/TLS 证书相关问题的处理方法：

1. 证书过期检查
   - openssl s_client -connect <domain>:443 -servername <domain> | openssl x509 -noout -dates
   - echo | openssl s_client -connect <domain>:443 2>/dev/null | openssl x509 -noout -enddate
   - 设置监控告警，在证书到期前30天提醒

2. 证书链不完整
   - 检查证书链：openssl s_client -connect <domain>:443 -showcerts
   - 确保中间证书已正确配置
   - 使用 SSL Labs 在线检测：https://www.ssllabs.com/ssltest/

3. Nginx HTTPS 配置
   - ssl_certificate 指向完整证书链文件
   - ssl_certificate_key 指向私钥文件
   - 推荐配置：ssl_protocols TLSv1.2 TLSv1.3;
   - 检查配置后执行 nginx -t 测试再 reload

4. 证书续期（Let's Encrypt）
   - certbot renew --dry-run 测试续期
   - 配置 crontab 自动续期：0 3 * * * certbot renew --quiet --post-hook "nginx -s reload"
   - 检查续期日志：/var/log/letsencrypt/letsencrypt.log

5. 客户端证书错误
   - 确认客户端系统时间正确（证书有效期校验依赖系统时间）
   - 检查客户端是否信任根证书
   - 旧版系统（Windows 7/Android 7以下）可能不支持新证书"""
    },
]

def main():
    # 1. 获取 admin token
    print("登录 admin...")
    resp = requests.post(f"{BASE}/auth/login", json={
        "username": "admin", "password": "Admin@123456"
    })
    token = resp.json()["data"]["access_token"]
    headers = {"Authorization": f"Bearer {token}", "Content-Type": "application/json"}

    # 2. 创建文章 (status=2 表示已审核通过)
    article_ids = []
    for i, art in enumerate(ARTICLES):
        print(f"[{i+1}/{len(ARTICLES)}] 创建: {art['title'][:40]}...")
        resp = requests.post(f"{BASE}/admin/knowledge-bases/1/articles",
                            json={"title": art["title"], "content": art["content"]}, headers=headers)
        if resp.status_code == 200:
            # 文章创建后需要审核和发布
            print(f"  创建成功")
            article_ids.append(None)  # 需要从 DB 查询 id
        else:
            print(f"  创建失败: {resp.status_code} {resp.text[:200]}")

    # 3. 查询文章列表获取 ID
    resp = requests.get(f"{BASE}/admin/knowledge-bases/1/articles?status=0", headers=headers)
    print(f"\n文章列表: {resp.text[:300]}")

    # 4. 直接从 DB 操作：审核通过 + 发布
    # 使用 psql 批量操作
    sql = """
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN SELECT id FROM knowledge_articles WHERE status = 1 AND kb_id = 1 LOOP
        -- Submit for review -> approve
        UPDATE knowledge_articles SET status = 2, updated_at = NOW() WHERE id = r.id;
        -- Publish -> status = 4
        -- Note: Publish requires embedding generation, handled below
    END LOOP;
END $$;
"""
    subprocess.run(["docker", "exec", "opsmind-postgres", "psql", "-U", "opsmind", "-d", "opsmind", "-c", sql],
                   capture_output=True)
    print("已将草稿文章设为已审核状态")

    # 5. 分块 + 生成 embedding
    # 查询所有已审核但未发布（status=2）的文章
    result = subprocess.run(["docker", "exec", "opsmind-postgres", "psql", "-U", "opsmind", "-d", "opsmind", "-t", "-A",
        "-c", "SELECT id, title, content FROM knowledge_articles WHERE status = 2 AND kb_id = 1 ORDER BY id"],
        capture_output=True, text=True)
    print(f"\n待处理文章: {len([l for l in result.stdout.strip().split(chr(10)) if l])} 篇")

    lines = [l for l in result.stdout.strip().split(chr(10)) if l]
    for line in lines:
        parts = line.split("|", 2)
        if len(parts) < 3:
            continue
        aid, title, content = parts
        print(f"\n处理文章 {aid}: {title[:50]}...")

        # 简单分块：按段落 + 长度切分，每块约500-800字符
        paragraphs = content.split("\n\n")
        chunks = []
        current = ""
        for p in paragraphs:
            p = p.strip()
            if not p:
                continue
            if len(current) + len(p) < 800:
                current = (current + "\n" + p).strip()
            else:
                if current:
                    chunks.append(current)
                current = p
        if current:
            chunks.append(current)

        print(f"  分块: {len(chunks)} 个")

        # 生成 embedding
        resp = requests.post(EMBED_API, json={
            "model": EMBED_MODEL, "input": chunks, "encoding_format": "float"
        }, headers={"Authorization": EMBED_KEY, "Content-Type": "application/json"}, timeout=60)
        if resp.status_code != 200:
            print(f"  Embedding API 错误: {resp.status_code}")
            continue

        data = resp.json()
        embs = sorted(data["data"], key=lambda x: x["index"])
        print(f"  Embedding: {len(embs)} 个向量, dim={len(embs[0]['embedding'])}")

        # 删除旧 chunks, 写入新 chunks + embeddings
        for i, (chunk, emb) in enumerate(zip(chunks, embs)):
            vec_str = "[" + ",".join(f"{v:.8f}" for v in emb["embedding"]) + "]"
            subprocess.run(["docker", "exec", "opsmind-postgres", "psql", "-U", "opsmind", "-d", "opsmind", "-c",
                f"INSERT INTO knowledge_chunks (article_id, kb_id, content, chunk_index, embedding_model, vector_dimension, embedding, created_at) "
                f"VALUES ({aid}, 1, $chunk${chunk}$chunk$, {i}, 'bge-m3', 1024, '{vec_str}'::halfvec, NOW())"],
                capture_output=True)

        # 更新文章状态为已发布 + 设置字数
        word_count = len(content.replace("\n", ""))
        subprocess.run(["docker", "exec", "opsmind-postgres", "psql", "-U", "opsmind", "-d", "opsmind", "-c",
            f"UPDATE knowledge_articles SET status = 4, chunk_count = {len(chunks)}, word_count = {word_count}, updated_at = NOW() WHERE id = {aid}"],
            capture_output=True)
        print(f"  发布完成: {len(chunks)} 个分块已写入向量")

    print("\n=== 知识库填充完成 ===")

if __name__ == "__main__":
    main()
