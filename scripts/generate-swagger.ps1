# Swagger 文档生成脚本 (PowerShell 版本)
# 用于生成 AI API Gateway 的 Swagger 文档

param(
    [string]$Action = "generate",
    [switch]$Help,
    [switch]$Clean,
    [switch]$Verify
)

# 设置控制台编码为 UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8

# 颜色定义
$Colors = @{
    Red    = "Red"
    Green  = "Green"
    Yellow = "Yellow"
    Blue   = "Blue"
    Cyan   = "Cyan"
    White  = "White"
}

# 项目路径
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir
$DocsDir = Join-Path $ProjectRoot "docs"
$MainFile = Join-Path $ProjectRoot "cmd\server\main.go"

function Write-ColorText {
    param(
        [string]$Text,
        [string]$Color = "White"
    )
    Write-Host $Text -ForegroundColor $Color
}

function Write-Info {
    param([string]$Message)
    Write-ColorText "[信息] $Message" $Colors.Blue
}

function Write-Success {
    param([string]$Message)
    Write-ColorText "[成功] $Message" $Colors.Green
}

function Write-Warning {
    param([string]$Message)
    Write-ColorText "[警告] $Message" $Colors.Yellow
}

function Write-Error {
    param([string]$Message)
    Write-ColorText "[错误] $Message" $Colors.Red
}

function Show-Help {
    Write-ColorText "AI API Gateway Swagger 文档生成器" $Colors.Blue
    Write-Host ""
    Write-Host "用法: .\scripts\generate-swagger.ps1 [选项]"
    Write-Host ""
    Write-Host "选项:"
    Write-Host "  -Help      显示此帮助信息"
    Write-Host "  -Clean     仅清理旧文档"
    Write-Host "  -Verify    仅验证现有文档"
    Write-Host ""
    Write-Host "示例:"
    Write-Host "  .\scripts\generate-swagger.ps1           # 生成完整的 Swagger 文档"
    Write-Host "  .\scripts\generate-swagger.ps1 -Clean   # 清理旧文档"
    Write-Host "  .\scripts\generate-swagger.ps1 -Verify  # 验证现有文档"
}

function Test-SwagTool {
    Write-Info "检查 swag 工具..."
    
    try {
        $version = & swag --version 2>$null
        if ($LASTEXITCODE -eq 0) {
            Write-Success "swag 工具已安装"
            Write-Host "版本: $version"
            return $true
        }
    }
    catch {
        # swag 未安装
    }
    
    Write-Warning "swag 工具未安装，正在安装..."
    try {
        & go install github.com/swaggo/swag/cmd/swag@latest
        if ($LASTEXITCODE -eq 0) {
            Write-Success "swag 工具安装成功"
            return $true
        }
        else {
            Write-Error "swag 工具安装失败"
            return $false
        }
    }
    catch {
        Write-Error "swag 工具安装失败: $_"
        return $false
    }
}

function Test-MainFile {
    Write-Info "检查主文件..."
    
    if (Test-Path $MainFile) {
        Write-Success "主文件存在: $MainFile"
        return $true
    }
    else {
        Write-Error "主文件不存在: $MainFile"
        return $false
    }
}

function New-DocsDirectory {
    Write-Info "创建文档目录..."
    
    if (-not (Test-Path $DocsDir)) {
        New-Item -ItemType Directory -Path $DocsDir -Force | Out-Null
        Write-Success "文档目录已创建: $DocsDir"
    }
    else {
        Write-Success "文档目录已存在: $DocsDir"
    }
    return $true
}

function Remove-OldDocs {
    Write-Info "清理旧文档..."
    
    $files = @("docs.go", "swagger.json", "swagger.yaml")
    $cleaned = $false
    
    foreach ($file in $files) {
        $filePath = Join-Path $DocsDir $file
        if (Test-Path $filePath) {
            Remove-Item $filePath -Force
            Write-Success "已删除旧的 $file"
            $cleaned = $true
        }
    }
    
    if (-not $cleaned) {
        Write-Info "没有需要清理的旧文档"
    }
    
    return $true
}

function Invoke-SwaggerGeneration {
    Write-Info "生成 Swagger 文档..."
    
    # 切换到项目根目录
    Push-Location $ProjectRoot
    
    try {
        # 运行 swag init 命令
        & swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
        
        if ($LASTEXITCODE -eq 0) {
            Write-Success "Swagger 文档生成成功！"
            return $true
        }
        else {
            Write-Error "Swagger 文档生成失败"
            return $false
        }
    }
    catch {
        Write-Error "Swagger 文档生成失败: $_"
        return $false
    }
    finally {
        Pop-Location
    }
}

function Test-GeneratedDocs {
    Write-Info "验证生成的文档..."
    
    $files = @("docs.go", "swagger.json", "swagger.yaml")
    $allExist = $true
    
    foreach ($file in $files) {
        $filePath = Join-Path $DocsDir $file
        if (Test-Path $filePath) {
            $size = (Get-Item $filePath).Length
            Write-Success "✓ $file ($size bytes)"
        }
        else {
            Write-Error "✗ $file 不存在"
            $allExist = $false
        }
    }
    
    if ($allExist) {
        Write-Success "所有文档文件生成成功"
        return $true
    }
    else {
        Write-Error "部分文档文件生成失败"
        return $false
    }
}

function Show-DocumentStats {
    Write-Info "文档统计信息:"
    
    $swaggerJsonPath = Join-Path $DocsDir "swagger.json"
    if (Test-Path $swaggerJsonPath) {
        $content = Get-Content $swaggerJsonPath -Raw
        
        Write-Host "• API 端点: 已生成"
        Write-Host "• 数据模型: 已生成"
        
        # 检查是否包含 Midjourney 端点
        if ($content -match "Midjourney") {
            Write-Success "• Midjourney API: ✓ 已包含"
        }
        else {
            Write-Warning "• Midjourney API: ⚠ 未找到"
        }
        
        # 检查是否包含认证配置
        if ($content -match "securityDefinitions") {
            Write-Success "• 认证配置: ✓ 已包含"
        }
        else {
            Write-Warning "• 认证配置: ⚠ 未找到"
        }
    }
}

function Show-AccessInfo {
    Write-ColorText "=== 文档访问信息 ===" $Colors.Green
    Write-ColorText "Swagger UI: http://localhost:8080/swagger/index.html" $Colors.Blue
    Write-ColorText "JSON 文档:  http://localhost:8080/swagger/doc.json" $Colors.Blue
    Write-ColorText "本地文件:" $Colors.Blue
    Write-Host "  • JSON: $(Join-Path $DocsDir 'swagger.json')"
    Write-Host "  • YAML: $(Join-Path $DocsDir 'swagger.yaml')"
    Write-Host "  • Go:   $(Join-Path $DocsDir 'docs.go')"
    Write-Host ""
    Write-Warning "注意: 需要启动服务器才能访问 Swagger UI"
    Write-Info "启动命令: go run cmd/server/main.go"
}

function Invoke-MainProcess {
    Write-ColorText "=== AI API Gateway Swagger 文档生成器 ===" $Colors.Blue
    Write-Host "项目根目录: $ProjectRoot"
    Write-Host ""
    
    if (-not (Test-SwagTool)) { return $false }
    if (-not (Test-MainFile)) { return $false }
    if (-not (New-DocsDirectory)) { return $false }
    if (-not (Remove-OldDocs)) { return $false }
    if (-not (Invoke-SwaggerGeneration)) { return $false }
    if (-not (Test-GeneratedDocs)) { return $false }
    
    Show-DocumentStats
    Write-Host ""
    Show-AccessInfo
    Write-Host ""
    Write-ColorText "=== Swagger 文档生成完成 ===" $Colors.Green
    
    return $true
}

# 主逻辑
try {
    if ($Help) {
        Show-Help
        exit 0
    }
    
    if ($Clean) {
        Write-Info "仅清理旧文档..."
        if (Remove-OldDocs) {
            Write-Success "清理完成"
            exit 0
        }
        else {
            Write-Error "清理失败"
            exit 1
        }
    }
    
    if ($Verify) {
        Write-Info "验证现有文档..."
        if (Test-GeneratedDocs) {
            Show-DocumentStats
            exit 0
        }
        else {
            Write-Error "验证失败"
            exit 1
        }
    }
    
    # 默认执行完整生成流程
    if (Invoke-MainProcess) {
        exit 0
    }
    else {
        Write-Error "文档生成失败"
        exit 1
    }
}
catch {
    Write-Error "脚本执行失败: $_"
    exit 1
}
