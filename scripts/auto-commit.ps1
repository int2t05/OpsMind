# OpsMind 自动提交脚本
# 用法: .\scripts\auto-commit.ps1 [-Message "提交信息"] [-Push] [-Watch]

param(
    [string]$Message = "chore: auto update",
    [switch]$Push,
    [switch]$Watch
)

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  OpsMind Git Auto Commit" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 检查是否在 git 仓库中
if (-not (Test-Path ".git")) {
    Write-Host "❌ 错误: 不在 Git 仓库中" -ForegroundColor Red
    exit 1
}

# 设置 git 环境变量（如果需要）
$env:GIT_REDIRECT_STDERR = "2>&1"

function Invoke-GitCommand {
    param([string]$Command)
    Write-Host "→ 执行: git $Command" -ForegroundColor Gray
    $output = git $Command.Split(' ') 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Host "❌ 命令失败: git $Command" -ForegroundColor Red
        Write-Host $output -ForegroundColor Red
        return $false
    }
    return $true
}

function Do-Commit {
    Write-Host "`n📋 检查变更..." -ForegroundColor Yellow
    
    # 检查状态
    $status = git status --porcelain
    if ([string]::IsNullOrWhiteSpace($status)) {
        Write-Host "✅ 没有变更需要提交" -ForegroundColor Green
        return $false
    }
    
    Write-Host "`n📝 变更内容:" -ForegroundColor Yellow
    Write-Host $status
    Write-Host ""
    
    # 添加所有变更
    if (-not (Invoke-GitCommand "add -A")) {
        return $false
    }
    
    # 提交
    $commitMsg = $Message
    if ($Message -eq "chore: auto update") {
        # 生成更智能的提交信息
        $changedFiles = git diff --name-only --cached
        $fileCount = ($changedFiles | Measure-Object -Line).Lines
        $commitMsg = "chore: auto update ($fileCount files changed)"
    }
    
    if (-not (Invoke-GitCommand "commit -m `"$commitMsg`"")) {
        return $false
    }
    
    Write-Host "`n✅ 提交成功!" -ForegroundColor Green
    
    # 推送（如果指定）
    if ($Push) {
        Write-Host "`n🚀 推送到远程仓库..." -ForegroundColor Yellow
        if (-not (Invoke-GitCommand "push")) {
            return $false
        }
        Write-Host "✅ 推送成功!" -ForegroundColor Green
    }
    
    return $true
}

# 监控模式
if ($Watch) {
    Write-Host "`n👀 监控模式启动中，按 Ctrl+C 停止..." -ForegroundColor Cyan
    Write-Host "   每 30 秒检查一次变更" -ForegroundColor Gray
    Write-Host ""
    
    while ($true) {
        $result = Do-Commit
        Start-Sleep -Seconds 30
    }
} else {
    # 单次提交模式
    Do-Commit
}
