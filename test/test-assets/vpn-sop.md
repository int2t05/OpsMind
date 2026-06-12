# OpsMind 运维平台 — VPN 故障排查 SOP

## 场景 A：VPN 客户端无法连接

### 排查步骤
1. 确认本地网络连通性：打开终端执行 `ping 10.0.1.1`
2. 检查 VPN 客户端版本：须 >= v3.2.1-beta7
3. 切换备用线路：`vpn-backup.internal.opsmind.io`，端口 8443
4. 清除本地 DNS 缓存：`ipconfig /flushdns`

### 如果以上步骤均无效
联系 NOC 值班电话：**400-888-9999**（7×24 小时）

## 场景 B：VPN 连接后无法访问内网

1. 检查路由表：`route print | findstr 10.0`
2. 确认 DNS 解析：`nslookup internal.opsmind.io`
3. 如 DNS 异常，手动设置 DNS 为 10.0.1.53
4. 代理检查：确保系统代理已关闭

> 以上 SOP 最后更新：2026-06-01，维护人：NetOps Team
