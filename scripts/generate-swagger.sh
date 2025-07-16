#!/bin/bash

# Swagger 文档生成脚本
# 用于生成 AI API Gateway 的 Swagger 文档

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DOCS_DIR="$PROJECT_ROOT/docs"
MAIN_FILE="$PROJECT_ROOT/cmd/server/main.go"

echo -e "${BLUE}=== AI API Gateway Swagger 文档生成器 ===${NC}"
echo -e "${BLUE}项目根目录: $PROJECT_ROOT${NC}"
echo

# 检查 swag 工具是否安装
check_swag() {
    echo -e "${YELLOW}检查 swag 工具...${NC}"
    if ! command -v swag &> /dev/null; then
        echo -e "${RED}错误: swag 工具未安装${NC}"
        echo -e "${YELLOW}正在安装 swag...${NC}"
        go install github.com/swaggo/swag/cmd/swag@latest
        if [ $? -eq 0 ]; then
            echo -e "${GREEN}swag 工具安装成功${NC}"
        else
            echo -e "${RED}swag 工具安装失败${NC}"
            exit 1
        fi
    else
        echo -e "${GREEN}swag 工具已安装${NC}"
        swag --version
    fi
    echo
}

# 检查主文件是否存在
check_main_file() {
    echo -e "${YELLOW}检查主文件...${NC}"
    if [ ! -f "$MAIN_FILE" ]; then
        echo -e "${RED}错误: 主文件不存在: $MAIN_FILE${NC}"
        exit 1
    fi
    echo -e "${GREEN}主文件存在: $MAIN_FILE${NC}"
    echo
}

# 创建文档目录
create_docs_dir() {
    echo -e "${YELLOW}创建文档目录...${NC}"
    if [ ! -d "$DOCS_DIR" ]; then
        mkdir -p "$DOCS_DIR"
        echo -e "${GREEN}文档目录已创建: $DOCS_DIR${NC}"
    else
        echo -e "${GREEN}文档目录已存在: $DOCS_DIR${NC}"
    fi
    echo
}

# 清理旧文档
clean_old_docs() {
    echo -e "${YELLOW}清理旧文档...${NC}"
    if [ -f "$DOCS_DIR/docs.go" ]; then
        rm -f "$DOCS_DIR/docs.go"
        echo -e "${GREEN}已删除旧的 docs.go${NC}"
    fi
    if [ -f "$DOCS_DIR/swagger.json" ]; then
        rm -f "$DOCS_DIR/swagger.json"
        echo -e "${GREEN}已删除旧的 swagger.json${NC}"
    fi
    if [ -f "$DOCS_DIR/swagger.yaml" ]; then
        rm -f "$DOCS_DIR/swagger.yaml"
        echo -e "${GREEN}已删除旧的 swagger.yaml${NC}"
    fi
    echo
}

# 生成 Swagger 文档
generate_swagger() {
    echo -e "${YELLOW}生成 Swagger 文档...${NC}"
    cd "$PROJECT_ROOT"
    
    # 运行 swag init 命令
    swag init -g cmd/server/main.go -o docs --parseDependency --parseInternal
    
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}Swagger 文档生成成功！${NC}"
    else
        echo -e "${RED}Swagger 文档生成失败${NC}"
        exit 1
    fi
    echo
}

# 验证生成的文档
verify_docs() {
    echo -e "${YELLOW}验证生成的文档...${NC}"
    
    local files=("docs.go" "swagger.json" "swagger.yaml")
    local all_exist=true
    
    for file in "${files[@]}"; do
        if [ -f "$DOCS_DIR/$file" ]; then
            local size=$(stat -f%z "$DOCS_DIR/$file" 2>/dev/null || stat -c%s "$DOCS_DIR/$file" 2>/dev/null)
            echo -e "${GREEN}✓ $file (${size} bytes)${NC}"
        else
            echo -e "${RED}✗ $file 不存在${NC}"
            all_exist=false
        fi
    done
    
    if [ "$all_exist" = true ]; then
        echo -e "${GREEN}所有文档文件生成成功${NC}"
    else
        echo -e "${RED}部分文档文件生成失败${NC}"
        exit 1
    fi
    echo
}

# 显示文档统计信息
show_stats() {
    echo -e "${YELLOW}文档统计信息:${NC}"
    
    if [ -f "$DOCS_DIR/swagger.json" ]; then
        # 统计 API 端点数量
        local endpoints=$(grep -o '"paths"' "$DOCS_DIR/swagger.json" | wc -l)
        local definitions=$(grep -o '"definitions"' "$DOCS_DIR/swagger.json" | wc -l)
        
        echo -e "${BLUE}• API 端点: 已生成${NC}"
        echo -e "${BLUE}• 数据模型: 已生成${NC}"
        
        # 检查是否包含 Midjourney 端点
        if grep -q "Midjourney" "$DOCS_DIR/swagger.json"; then
            echo -e "${GREEN}• Midjourney API: ✓ 已包含${NC}"
        else
            echo -e "${YELLOW}• Midjourney API: ⚠ 未找到${NC}"
        fi
        
        # 检查是否包含认证配置
        if grep -q "securityDefinitions" "$DOCS_DIR/swagger.json"; then
            echo -e "${GREEN}• 认证配置: ✓ 已包含${NC}"
        else
            echo -e "${YELLOW}• 认证配置: ⚠ 未找到${NC}"
        fi
    fi
    echo
}

# 显示访问信息
show_access_info() {
    echo -e "${GREEN}=== 文档访问信息 ===${NC}"
    echo -e "${BLUE}Swagger UI: ${NC}http://localhost:8080/swagger/index.html"
    echo -e "${BLUE}JSON 文档: ${NC}http://localhost:8080/swagger/doc.json"
    echo -e "${BLUE}本地文件: ${NC}"
    echo -e "  • JSON: $DOCS_DIR/swagger.json"
    echo -e "  • YAML: $DOCS_DIR/swagger.yaml"
    echo -e "  • Go:   $DOCS_DIR/docs.go"
    echo
    echo -e "${YELLOW}注意: 需要启动服务器才能访问 Swagger UI${NC}"
    echo -e "${YELLOW}启动命令: go run cmd/server/main.go${NC}"
    echo
}

# 主函数
main() {
    echo -e "${BLUE}开始生成 Swagger 文档...${NC}"
    echo
    
    check_swag
    check_main_file
    create_docs_dir
    clean_old_docs
    generate_swagger
    verify_docs
    show_stats
    show_access_info
    
    echo -e "${GREEN}=== Swagger 文档生成完成 ===${NC}"
}

# 帮助信息
show_help() {
    echo "AI API Gateway Swagger 文档生成器"
    echo
    echo "用法: $0 [选项]"
    echo
    echo "选项:"
    echo "  -h, --help     显示此帮助信息"
    echo "  -c, --clean    仅清理旧文档"
    echo "  -v, --verify   仅验证现有文档"
    echo
    echo "示例:"
    echo "  $0              # 生成完整的 Swagger 文档"
    echo "  $0 --clean     # 清理旧文档"
    echo "  $0 --verify    # 验证现有文档"
}

# 处理命令行参数
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    -c|--clean)
        echo -e "${YELLOW}仅清理旧文档...${NC}"
        clean_old_docs
        echo -e "${GREEN}清理完成${NC}"
        exit 0
        ;;
    -v|--verify)
        echo -e "${YELLOW}验证现有文档...${NC}"
        verify_docs
        show_stats
        exit 0
        ;;
    "")
        main
        ;;
    *)
        echo -e "${RED}未知选项: $1${NC}"
        echo "使用 --help 查看帮助信息"
        exit 1
        ;;
esac
