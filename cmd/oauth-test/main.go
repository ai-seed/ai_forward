package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"ai-api-gateway/internal/infrastructure/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

func main() {
	fmt.Println("🔧 OAuth 配置验证工具")
	fmt.Println("====================")

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("❌ 加载配置失败: %v", err)
	}

	// 验证 Google OAuth 配置
	fmt.Println("\n📱 Google OAuth 配置:")
	if cfg.OAuth.Google.Enabled {
		fmt.Println("✅ Google OAuth 已启用")
		
		if cfg.OAuth.Google.ClientID == "" {
			fmt.Println("❌ Google Client ID 未配置")
		} else {
			fmt.Printf("✅ Google Client ID: %s...\n", cfg.OAuth.Google.ClientID[:10])
		}
		
		if cfg.OAuth.Google.ClientSecret == "" {
			fmt.Println("❌ Google Client Secret 未配置")
		} else {
			fmt.Println("✅ Google Client Secret 已配置")
		}
		
		fmt.Printf("✅ Google Redirect URL: %s\n", cfg.OAuth.Google.RedirectURL)
		
		// 测试 Google OAuth 配置
		googleConfig := &oauth2.Config{
			ClientID:     cfg.OAuth.Google.ClientID,
			ClientSecret: cfg.OAuth.Google.ClientSecret,
			RedirectURL:  cfg.OAuth.Google.RedirectURL,
			Scopes:       []string{"openid", "profile", "email"},
			Endpoint:     google.Endpoint,
		}
		
		authURL := googleConfig.AuthCodeURL("test-state")
		fmt.Printf("🔗 Google 认证 URL: %s\n", authURL)
		
	} else {
		fmt.Println("⚠️  Google OAuth 未启用")
	}

	// 验证 GitHub OAuth 配置
	fmt.Println("\n🐙 GitHub OAuth 配置:")
	if cfg.OAuth.GitHub.Enabled {
		fmt.Println("✅ GitHub OAuth 已启用")
		
		if cfg.OAuth.GitHub.ClientID == "" {
			fmt.Println("❌ GitHub Client ID 未配置")
		} else {
			fmt.Printf("✅ GitHub Client ID: %s...\n", cfg.OAuth.GitHub.ClientID[:10])
		}
		
		if cfg.OAuth.GitHub.ClientSecret == "" {
			fmt.Println("❌ GitHub Client Secret 未配置")
		} else {
			fmt.Println("✅ GitHub Client Secret 已配置")
		}
		
		fmt.Printf("✅ GitHub Redirect URL: %s\n", cfg.OAuth.GitHub.RedirectURL)
		
		// 测试 GitHub OAuth 配置
		githubConfig := &oauth2.Config{
			ClientID:     cfg.OAuth.GitHub.ClientID,
			ClientSecret: cfg.OAuth.GitHub.ClientSecret,
			RedirectURL:  cfg.OAuth.GitHub.RedirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     github.Endpoint,
		}
		
		authURL := githubConfig.AuthCodeURL("test-state")
		fmt.Printf("🔗 GitHub 认证 URL: %s\n", authURL)
		
	} else {
		fmt.Println("⚠️  GitHub OAuth 未启用")
	}

	// 环境变量检查
	fmt.Println("\n🌍 环境变量检查:")
	checkEnvVar("OAUTH_GOOGLE_ENABLED")
	checkEnvVar("OAUTH_GOOGLE_CLIENT_ID")
	checkEnvVar("OAUTH_GOOGLE_CLIENT_SECRET")
	checkEnvVar("OAUTH_GOOGLE_REDIRECT_URL")
	checkEnvVar("OAUTH_GITHUB_ENABLED")
	checkEnvVar("OAUTH_GITHUB_CLIENT_ID")
	checkEnvVar("OAUTH_GITHUB_CLIENT_SECRET")
	checkEnvVar("OAUTH_GITHUB_REDIRECT_URL")

	fmt.Println("\n🎉 配置验证完成！")
	
	// 如果配置不完整，提供帮助信息
	if !cfg.OAuth.Google.Enabled && !cfg.OAuth.GitHub.Enabled {
		fmt.Println("\n📋 配置帮助:")
		fmt.Println("请参考 docs/oauth-setup.md 文件进行 OAuth 配置")
	}
}

func checkEnvVar(name string) {
	value := os.Getenv(name)
	if value == "" {
		fmt.Printf("⚠️  %s 未设置\n", name)
	} else {
		if name == "OAUTH_GOOGLE_CLIENT_ID" || name == "OAUTH_GITHUB_CLIENT_ID" {
			fmt.Printf("✅ %s: %s...\n", name, value[:min(10, len(value))])
		} else if name == "OAUTH_GOOGLE_CLIENT_SECRET" || name == "OAUTH_GITHUB_CLIENT_SECRET" {
			fmt.Printf("✅ %s: [已设置]\n", name)
		} else {
			fmt.Printf("✅ %s: %s\n", name, value)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
