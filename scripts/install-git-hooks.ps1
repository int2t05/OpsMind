# 安装 Git Hooks 脚本
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  安装 Git Hooks" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$hooksDir = ".git\hooks"
$sourceHooksDir = "scripts\git-hooks"

# 创建源 hooks 目录
if (-not (Test-Path $sourceHooksDir)) {
    New-Item -ItemType Directory -Path $sourceHooksDir -Force | Out-Null
    Write-Host "✅ 创建目录: $sourceHooksDir" -ForegroundColor Green
}

# 创建 post-merge hook（合并后自动提交）
$postMergeHook = @"
#!/bin/sh
# Auto commit after merge
echo "🔄 Post-merge hook triggered"
git add -A
git commit -m "chore: auto commit after merge" 2>/dev/null || true
"@

$postMergeHook | Out-File -FilePath "$sourceHooksDir\post-merge" -Encoding utf8
Write-Host "✅ 创建: post-merge hook" -ForegroundColor Green

# 创建 pre-commit hook（可选的格式化检查）
$preCommitHook = @"
#!/bin/sh
# Pre-commit checks
echo "🔍 Running pre-commit checks..."
# 在这里可以添加 lint、格式化等检查
exit 0
"@

$preCommitHook | Out-File -FilePath "$sourceHooksDir\pre-commit" -Encoding utf8
Write-Host "✅ 创建: pre-commit hook" -ForegroundColor Green

# 创建符号链接（Windows 需要管理员权限或使用复制）
Write-Host "`n📦 安装 hooks..." -ForegroundColor Yellow
if (Test-Path $hooksDir) {
    Copy-Item -Path "$sourceHooksDir\*" -Destination $hooksDir -Force
    Write-Host "✅ Hooks 已安装到: $hooksDir" -ForegroundColor Green
} else {
    Write-Host "⚠️  找不到 .git\hooks 目录" -ForegroundColor Yellow
}

Write-Host "`n🎉 Git Hooks 安装完成!" -ForegroundColor Green
Write-Host "   - post-merge: 合并后自动提交" -ForegroundColor Gray
Write-Host "   - pre-commit: 提交前检查" -ForegroundColor Gray
