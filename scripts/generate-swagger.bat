@echo off
setlocal enabledelayedexpansion

REM Swagger 文档生成脚本 (Windows 版本)
REM 用于生成 AI API Gateway 的 Swagger 文档

title AI API Gateway - Swagger 文档生成器

REM 设置项目根目录
set "PROJECT_ROOT=%~dp0.."
set "DOCS_DIR=%PROJECT_ROOT%\docs"
set "MAIN_FILE=%PROJECT_ROOT%\cmd\server\main.go"

echo ========================================
echo AI API Gateway Swagger 文档生成器
echo ========================================
echo 项目根目录: %PROJECT_ROOT%
echo.

REM 检查参数
if "%1"=="--help" goto :show_help
if "%1"=="-h" goto :show_help
if "%1"=="--clean" goto :clean_only
if "%1"=="-c" goto :clean_only
if "%1"=="--verify" goto :verify_only
if "%1"=="-v" goto :verify_only

goto :main

:show_help
echo AI API Gateway Swagger 文档生成器
echo.
echo 用法: %0 [选项]
echo.
echo 选项:
echo   -h, --help     显示此帮助信息
echo   -c, --clean    仅清理旧文档
echo   -v, --verify   仅验证现有文档
echo.
echo 示例:
echo   %0              # 生成完整的 Swagger 文档
echo   %0 --clean     # 清理旧文档
echo   %0 --verify    # 验证现有文档
goto :end

:clean_only
echo [清理] 仅清理旧文档...
call :clean_old_docs
echo [完成] 清理完成
goto :end

:verify_only
echo [验证] 验证现有文档...
call :verify_docs
call :show_stats
goto :end

:main
echo [开始] 生成 Swagger 文档...
echo.

call :check_swag
if errorlevel 1 goto :error

call :check_main_file
if errorlevel 1 goto :error

call :create_docs_dir
if errorlevel 1 goto :error

call :clean_old_docs
if errorlevel 1 goto :error

call :generate_swagger
if errorlevel 1 goto :error

call :verify_docs
if errorlevel 1 goto :error

call :show_stats

call :show_access_info

echo.
echo ========================================
echo Swagger 文档生成完成
echo ========================================
goto :end

:check_swag
echo [检查] 检查 swag 工具...
swag --version >nul 2>&1
if errorlevel 1 (
    echo [错误] swag 工具未安装
    echo [安装] 正在安装 swag...
    go install github.com/swaggo/swag/cmd/swag@latest
    if errorlevel 1 (
        echo [错误] swag 工具安装失败
        exit /b 1
    )
    echo [成功] swag 工具安装成功
) else (
    echo [成功] swag 工具已安装
    swag --version
)
echo.
exit /b 0

:check_main_file
echo [检查] 检查主文件...
if not exist "%MAIN_FILE%" (
    echo [错误] 主文件不存在: %MAIN_FILE%
    exit /b 1
)
echo [成功] 主文件存在: %MAIN_FILE%
echo.
exit /b 0

:create_docs_dir
echo [创建] 创建文档目录...
if not exist "%DOCS_DIR%" (
    mkdir "%DOCS_DIR%"
    echo [成功] 文档目录已创建: %DOCS_DIR%
) else (
    echo [成功] 文档目录已存在: %DOCS_DIR%
)
echo.
exit /b 0

:clean_old_docs
echo [清理] 清理旧文档...
if exist "%DOCS_DIR%\docs.go" (
    del "%DOCS_DIR%\docs.go"
    echo [删除] 已删除旧的 docs.go
)
if exist "%DOCS_DIR%\swagger.json" (
    del "%DOCS_DIR%\swagger.json"
    echo [删除] 已删除旧的 swagger.json
)
if exist "%DOCS_DIR%\swagger.yaml" (
    del "%DOCS_DIR%\swagger.yaml"
    echo [删除] 已删除旧的 swagger.yaml
)
echo.
exit /b 0

:generate_swagger
echo [生成] 生成 Swagger 文档...
cd /d "%PROJECT_ROOT%"

REM 运行 swag init 命令
swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal

if errorlevel 1 (
    echo [错误] Swagger 文档生成失败
    exit /b 1
)

echo [成功] Swagger 文档生成成功！
echo.
exit /b 0

:verify_docs
echo [验证] 验证生成的文档...

set "all_exist=true"

if exist "%DOCS_DIR%\docs.go" (
    for %%A in ("%DOCS_DIR%\docs.go") do set "size=%%~zA"
    echo [成功] ✓ docs.go (!size! bytes^)
) else (
    echo [错误] ✗ docs.go 不存在
    set "all_exist=false"
)

if exist "%DOCS_DIR%\swagger.json" (
    for %%A in ("%DOCS_DIR%\swagger.json") do set "size=%%~zA"
    echo [成功] ✓ swagger.json (!size! bytes^)
) else (
    echo [错误] ✗ swagger.json 不存在
    set "all_exist=false"
)

if exist "%DOCS_DIR%\swagger.yaml" (
    for %%A in ("%DOCS_DIR%\swagger.yaml") do set "size=%%~zA"
    echo [成功] ✓ swagger.yaml (!size! bytes^)
) else (
    echo [错误] ✗ swagger.yaml 不存在
    set "all_exist=false"
)

if "!all_exist!"=="true" (
    echo [成功] 所有文档文件生成成功
) else (
    echo [错误] 部分文档文件生成失败
    exit /b 1
)
echo.
exit /b 0

:show_stats
echo [统计] 文档统计信息:

if exist "%DOCS_DIR%\swagger.json" (
    echo [信息] • API 端点: 已生成
    echo [信息] • 数据模型: 已生成
    
    REM 检查是否包含 Midjourney 端点
    findstr /C:"Midjourney" "%DOCS_DIR%\swagger.json" >nul 2>&1
    if not errorlevel 1 (
        echo [成功] • Midjourney API: ✓ 已包含
    ) else (
        echo [警告] • Midjourney API: ⚠ 未找到
    )
    
    REM 检查是否包含认证配置
    findstr /C:"securityDefinitions" "%DOCS_DIR%\swagger.json" >nul 2>&1
    if not errorlevel 1 (
        echo [成功] • 认证配置: ✓ 已包含
    ) else (
        echo [警告] • 认证配置: ⚠ 未找到
    )
)
echo.
exit /b 0

:show_access_info
echo ========================================
echo 文档访问信息
echo ========================================
echo Swagger UI: http://localhost:8080/swagger/index.html
echo JSON 文档:  http://localhost:8080/swagger/doc.json
echo 本地文件:
echo   • JSON: %DOCS_DIR%\swagger.json
echo   • YAML: %DOCS_DIR%\swagger.yaml
echo   • Go:   %DOCS_DIR%\docs.go
echo.
echo 注意: 需要启动服务器才能访问 Swagger UI
echo 启动命令: go run cmd/server/main.go
echo.
exit /b 0

:error
echo.
echo [错误] 文档生成过程中出现错误
exit /b 1

:end
pause
