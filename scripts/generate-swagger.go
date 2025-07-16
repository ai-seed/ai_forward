package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// 颜色代码
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorPurple = "\033[35m"
	ColorCyan   = "\033[36m"
	ColorWhite  = "\033[37m"
)

// SwaggerGenerator Swagger文档生成器
type SwaggerGenerator struct {
	ProjectRoot string
	DocsDir     string
	MainFile    string
}

// NewSwaggerGenerator 创建新的生成器
func NewSwaggerGenerator() (*SwaggerGenerator, error) {
	// 获取当前脚本所在目录的上级目录作为项目根目录
	scriptDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, fmt.Errorf("获取脚本目录失败: %v", err)
	}
	
	projectRoot := filepath.Dir(scriptDir)
	docsDir := filepath.Join(projectRoot, "docs")
	mainFile := filepath.Join(projectRoot, "cmd", "server", "main.go")
	
	return &SwaggerGenerator{
		ProjectRoot: projectRoot,
		DocsDir:     docsDir,
		MainFile:    mainFile,
	}, nil
}

// colorPrint 彩色打印（Windows下可能不支持）
func colorPrint(color, text string) {
	if runtime.GOOS == "windows" {
		fmt.Print(text)
	} else {
		fmt.Print(color + text + ColorReset)
	}
}

// colorPrintln 彩色打印并换行
func colorPrintln(color, text string) {
	colorPrint(color, text)
	fmt.Println()
}

// info 打印信息
func (sg *SwaggerGenerator) info(text string) {
	colorPrintln(ColorBlue, "[信息] "+text)
}

// success 打印成功信息
func (sg *SwaggerGenerator) success(text string) {
	colorPrintln(ColorGreen, "[成功] "+text)
}

// warning 打印警告信息
func (sg *SwaggerGenerator) warning(text string) {
	colorPrintln(ColorYellow, "[警告] "+text)
}

// error 打印错误信息
func (sg *SwaggerGenerator) error(text string) {
	colorPrintln(ColorRed, "[错误] "+text)
}

// checkSwag 检查swag工具是否安装
func (sg *SwaggerGenerator) checkSwag() error {
	sg.info("检查 swag 工具...")
	
	cmd := exec.Command("swag", "--version")
	if err := cmd.Run(); err != nil {
		sg.warning("swag 工具未安装，正在安装...")
		
		installCmd := exec.Command("go", "install", "github.com/swaggo/swag/cmd/swag@latest")
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("swag 工具安装失败: %v", err)
		}
		
		sg.success("swag 工具安装成功")
	} else {
		sg.success("swag 工具已安装")
		
		// 显示版本信息
		versionCmd := exec.Command("swag", "--version")
		if output, err := versionCmd.Output(); err == nil {
			fmt.Printf("版本: %s", string(output))
		}
	}
	
	return nil
}

// checkMainFile 检查主文件是否存在
func (sg *SwaggerGenerator) checkMainFile() error {
	sg.info("检查主文件...")
	
	if _, err := os.Stat(sg.MainFile); os.IsNotExist(err) {
		return fmt.Errorf("主文件不存在: %s", sg.MainFile)
	}
	
	sg.success(fmt.Sprintf("主文件存在: %s", sg.MainFile))
	return nil
}

// createDocsDir 创建文档目录
func (sg *SwaggerGenerator) createDocsDir() error {
	sg.info("创建文档目录...")
	
	if err := os.MkdirAll(sg.DocsDir, 0755); err != nil {
		return fmt.Errorf("创建文档目录失败: %v", err)
	}
	
	sg.success(fmt.Sprintf("文档目录已准备: %s", sg.DocsDir))
	return nil
}

// cleanOldDocs 清理旧文档
func (sg *SwaggerGenerator) cleanOldDocs() error {
	sg.info("清理旧文档...")
	
	files := []string{"docs.go", "swagger.json", "swagger.yaml"}
	cleaned := false
	
	for _, file := range files {
		filePath := filepath.Join(sg.DocsDir, file)
		if _, err := os.Stat(filePath); err == nil {
			if err := os.Remove(filePath); err != nil {
				sg.warning(fmt.Sprintf("删除 %s 失败: %v", file, err))
			} else {
				sg.success(fmt.Sprintf("已删除旧的 %s", file))
				cleaned = true
			}
		}
	}
	
	if !cleaned {
		sg.info("没有需要清理的旧文档")
	}
	
	return nil
}

// generateSwagger 生成Swagger文档
func (sg *SwaggerGenerator) generateSwagger() error {
	sg.info("生成 Swagger 文档...")
	
	// 切换到项目根目录
	if err := os.Chdir(sg.ProjectRoot); err != nil {
		return fmt.Errorf("切换到项目根目录失败: %v", err)
	}
	
	// 运行 swag init 命令
	cmd := exec.Command("swag", "init", "-g", "cmd/server/main.go", "-o", "docs", "--parseDependency", "--parseInternal")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Swagger 文档生成失败: %v", err)
	}
	
	sg.success("Swagger 文档生成成功！")
	return nil
}

// verifyDocs 验证生成的文档
func (sg *SwaggerGenerator) verifyDocs() error {
	sg.info("验证生成的文档...")
	
	files := []string{"docs.go", "swagger.json", "swagger.yaml"}
	allExist := true
	
	for _, file := range files {
		filePath := filepath.Join(sg.DocsDir, file)
		if info, err := os.Stat(filePath); err == nil {
			sg.success(fmt.Sprintf("✓ %s (%d bytes)", file, info.Size()))
		} else {
			sg.error(fmt.Sprintf("✗ %s 不存在", file))
			allExist = false
		}
	}
	
	if !allExist {
		return fmt.Errorf("部分文档文件生成失败")
	}
	
	sg.success("所有文档文件生成成功")
	return nil
}

// showStats 显示文档统计信息
func (sg *SwaggerGenerator) showStats() {
	sg.info("文档统计信息:")
	
	swaggerJsonPath := filepath.Join(sg.DocsDir, "swagger.json")
	if content, err := os.ReadFile(swaggerJsonPath); err == nil {
		contentStr := string(content)
		
		fmt.Println("• API 端点: 已生成")
		fmt.Println("• 数据模型: 已生成")
		
		// 检查是否包含 Midjourney 端点
		if strings.Contains(contentStr, "Midjourney") {
			sg.success("• Midjourney API: ✓ 已包含")
		} else {
			sg.warning("• Midjourney API: ⚠ 未找到")
		}
		
		// 检查是否包含认证配置
		if strings.Contains(contentStr, "securityDefinitions") {
			sg.success("• 认证配置: ✓ 已包含")
		} else {
			sg.warning("• 认证配置: ⚠ 未找到")
		}
	}
}

// showAccessInfo 显示访问信息
func (sg *SwaggerGenerator) showAccessInfo() {
	colorPrintln(ColorGreen, "=== 文档访问信息 ===")
	colorPrintln(ColorBlue, "Swagger UI: http://localhost:8080/swagger/index.html")
	colorPrintln(ColorBlue, "JSON 文档:  http://localhost:8080/swagger/doc.json")
	colorPrintln(ColorBlue, "本地文件:")
	fmt.Printf("  • JSON: %s\n", filepath.Join(sg.DocsDir, "swagger.json"))
	fmt.Printf("  • YAML: %s\n", filepath.Join(sg.DocsDir, "swagger.yaml"))
	fmt.Printf("  • Go:   %s\n", filepath.Join(sg.DocsDir, "docs.go"))
	fmt.Println()
	sg.warning("注意: 需要启动服务器才能访问 Swagger UI")
	sg.info("启动命令: go run cmd/server/main.go")
}

// showHelp 显示帮助信息
func showHelp() {
	fmt.Println("AI API Gateway Swagger 文档生成器")
	fmt.Println()
	fmt.Println("用法: go run scripts/generate-swagger.go [选项]")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -h, --help     显示此帮助信息")
	fmt.Println("  -c, --clean    仅清理旧文档")
	fmt.Println("  -v, --verify   仅验证现有文档")
	fmt.Println()
	fmt.Println("示例:")
	fmt.Println("  go run scripts/generate-swagger.go              # 生成完整的 Swagger 文档")
	fmt.Println("  go run scripts/generate-swagger.go --clean     # 清理旧文档")
	fmt.Println("  go run scripts/generate-swagger.go --verify    # 验证现有文档")
}

// run 运行生成器
func (sg *SwaggerGenerator) run() error {
	colorPrintln(ColorBlue, "=== AI API Gateway Swagger 文档生成器 ===")
	fmt.Printf("项目根目录: %s\n", sg.ProjectRoot)
	fmt.Println()
	
	if err := sg.checkSwag(); err != nil {
		return err
	}
	
	if err := sg.checkMainFile(); err != nil {
		return err
	}
	
	if err := sg.createDocsDir(); err != nil {
		return err
	}
	
	if err := sg.cleanOldDocs(); err != nil {
		return err
	}
	
	if err := sg.generateSwagger(); err != nil {
		return err
	}
	
	if err := sg.verifyDocs(); err != nil {
		return err
	}
	
	sg.showStats()
	fmt.Println()
	sg.showAccessInfo()
	fmt.Println()
	colorPrintln(ColorGreen, "=== Swagger 文档生成完成 ===")
	
	return nil
}

func main() {
	// 处理命令行参数
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "-h", "--help":
			showHelp()
			return
		case "-c", "--clean":
			sg, err := NewSwaggerGenerator()
			if err != nil {
				fmt.Printf("初始化失败: %v\n", err)
				os.Exit(1)
			}
			sg.info("仅清理旧文档...")
			if err := sg.cleanOldDocs(); err != nil {
				sg.error(fmt.Sprintf("清理失败: %v", err))
				os.Exit(1)
			}
			sg.success("清理完成")
			return
		case "-v", "--verify":
			sg, err := NewSwaggerGenerator()
			if err != nil {
				fmt.Printf("初始化失败: %v\n", err)
				os.Exit(1)
			}
			sg.info("验证现有文档...")
			if err := sg.verifyDocs(); err != nil {
				sg.error(fmt.Sprintf("验证失败: %v", err))
				os.Exit(1)
			}
			sg.showStats()
			return
		default:
			fmt.Printf("未知选项: %s\n", os.Args[1])
			fmt.Println("使用 --help 查看帮助信息")
			os.Exit(1)
		}
	}
	
	// 创建生成器并运行
	sg, err := NewSwaggerGenerator()
	if err != nil {
		fmt.Printf("初始化失败: %v\n", err)
		os.Exit(1)
	}
	
	if err := sg.run(); err != nil {
		sg.error(fmt.Sprintf("生成失败: %v", err))
		os.Exit(1)
	}
}
