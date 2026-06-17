# Git 自动提交工具

本目录包含 OpsMind 项目的 Git 自动化工具。

## 📦 可用工具

### 1. PowerShell 自动提交脚本 (推荐)

最简单的自动提交方式。

#### 使用方法

```powershell
# 进入项目目录
cd d:\software\OpsMind

# 单次提交（不推送）
.\scripts\auto-commit.ps1

# 单次提交并推送
.\scripts\auto-commit.ps1 -Push

# 自定义提交信息
.\scripts\auto-commit.ps1 -Message "feat: 添加新功能" -Push

# 监控模式（每30秒检查一次变更并自动提交）
.\scripts\auto-commit.ps1 -Watch -Push
```

#### 参数说明

| 参数 | 说明 |
|------|------|
| `-Message` | 提交信息，默认: "chore: auto update" |
| `-Push` | 提交后自动推送到远程 |
| `-Watch` | 监控模式，持续检查变更 |

---

### 2. Git Hooks 自动提交

在特定 git 事件（如合并、拉取）后自动提交。

#### 安装

```powershell
.\scripts\install-git-hooks.ps1
```

#### 包含的 Hooks

| Hook | 说明 |
|------|------|
| `post-merge` | 合并后自动提交变更 |
| `pre-commit` | 提交前检查（可扩展） |

---

### 3. GitHub Actions 自动创建 PR

在 GitHub 云端自动创建 PR。

#### 使用方法

1. 访问你的仓库：https://github.com/hua-2005/OpsMind
2. 点击 **Actions** 标签
3. 选择 **"Auto PR Creator"** 工作流
4. 点击 **"Run workflow"** 按钮

---

## 🎯 快速开始

### 日常开发使用

```powershell
# 开始工作前
cd d:\software\OpsMind

# 打开一个终端，运行监控模式
.\scripts\auto-commit.ps1 -Watch -Push
```

然后继续开发，脚本会每 30 秒自动保存你的更改！

---

## 📝 提示

- **PowerShell 执行策略**：如果遇到执行策略限制，运行：
  ```powershell
  Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
  ```
- **GitHub Token**：如需在脚本中使用 GitHub CLI，设置环境变量：
  ```powershell
  $env:GH_TOKEN = "你的token"
  ```

---

## 📂 文件结构

```
scripts/
├── README.md                    # 本文件
├── auto-commit.ps1             # PowerShell 自动提交脚本
├── install-git-hooks.ps1       # Git Hooks 安装脚本
└── git-hooks/                  # Git Hook 模板
    ├── post-merge
    └── pre-commit
```
