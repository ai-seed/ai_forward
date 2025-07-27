package email

import (
	"fmt"
	"strings"
)

// TemplateService 邮件模板服务
type TemplateService struct {
	companyName string
	logoURL     string
}

// NewTemplateService 创建邮件模板服务
func NewTemplateService() *TemplateService {
	return &TemplateService{
		companyName: "718AI",
		logoURL:     "https://718ai.cn/logo.png", // 占位符，实际使用时替换为真实logo URL
	}
}

// RenderVerificationCode 渲染验证码邮件模板
func (t *TemplateService) RenderVerificationCode(code string) (subject, htmlBody, textBody string) {
	subject = "【718AI】邮箱验证码"

	htmlBody = t.renderVerificationCodeHTML(code)
	textBody = t.renderVerificationCodeText(code)

	return subject, htmlBody, textBody
}

// RenderPasswordResetCode 渲染密码重置邮件模板
func (t *TemplateService) RenderPasswordResetCode(code string) (subject, htmlBody, textBody string) {
	subject = "【718AI】密码重置验证码"

	htmlBody = t.renderPasswordResetHTML(code)
	textBody = t.renderPasswordResetText(code)

	return subject, htmlBody, textBody
}

// renderVerificationCodeHTML 渲染注册验证码HTML模板
func (t *TemplateService) renderVerificationCodeHTML(code string) string {
	template := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>邮箱验证码</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            max-width: 120px;
            height: auto;
            margin-bottom: 20px;
        }
        .title {
            color: #2c3e50;
            font-size: 24px;
            font-weight: 600;
            margin: 0;
        }
        .content {
            margin: 30px 0;
        }
        .verification-code {
            background-color: #f8f9fa;
            border: 2px dashed #007bff;
            border-radius: 8px;
            padding: 20px;
            text-align: center;
            margin: 20px 0;
        }
        .code {
            font-size: 32px;
            font-weight: bold;
            color: #007bff;
            letter-spacing: 4px;
            font-family: 'Courier New', monospace;
        }
        .note {
            color: #666;
            font-size: 14px;
            margin-top: 20px;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            text-align: center;
            color: #999;
            font-size: 12px;
        }
        .warning {
            background-color: #fff3cd;
            border: 1px solid #ffeaa7;
            border-radius: 4px;
            padding: 15px;
            margin: 20px 0;
            color: #856404;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <img src="{{.LogoURL}}" alt="{{.CompanyName}}" class="logo">
            <h1 class="title">邮箱验证</h1>
        </div>
        
        <div class="content">
            <p>您好！</p>
            <p>感谢您注册{{.CompanyName}}账户。为了确保您的邮箱地址有效，请使用以下验证码完成注册：</p>
            
            <div class="verification-code">
                <div class="code">{{.Code}}</div>
            </div>
            
            <div class="warning">
                <strong>重要提醒：</strong>
                <ul style="margin: 10px 0; padding-left: 20px;">
                    <li>验证码有效期为10分钟</li>
                    <li>请勿将验证码告诉他人</li>
                    <li>如果您没有注册{{.CompanyName}}账户，请忽略此邮件</li>
                </ul>
            </div>
            
            <p class="note">
                如果您在使用过程中遇到任何问题，请联系我们的客服团队。
            </p>
        </div>
        
        <div class="footer">
            <p>此邮件由系统自动发送，请勿回复。</p>
            <p>&copy; 2024 {{.CompanyName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	// 替换模板变量
	template = strings.ReplaceAll(template, "{{.LogoURL}}", t.logoURL)
	template = strings.ReplaceAll(template, "{{.CompanyName}}", t.companyName)
	template = strings.ReplaceAll(template, "{{.Code}}", code)

	return template
}

// renderPasswordResetHTML 渲染密码重置HTML模板
func (t *TemplateService) renderPasswordResetHTML(code string) string {
	template := `
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>密码重置验证码</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
            line-height: 1.6;
            color: #333;
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            background-color: #ffffff;
            border-radius: 8px;
            padding: 40px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        }
        .header {
            text-align: center;
            margin-bottom: 30px;
        }
        .logo {
            max-width: 120px;
            height: auto;
            margin-bottom: 20px;
        }
        .title {
            color: #e74c3c;
            font-size: 24px;
            font-weight: 600;
            margin: 0;
        }
        .content {
            margin: 30px 0;
        }
        .verification-code {
            background-color: #f8f9fa;
            border: 2px dashed #e74c3c;
            border-radius: 8px;
            padding: 20px;
            text-align: center;
            margin: 20px 0;
        }
        .code {
            font-size: 32px;
            font-weight: bold;
            color: #e74c3c;
            letter-spacing: 4px;
            font-family: 'Courier New', monospace;
        }
        .note {
            color: #666;
            font-size: 14px;
            margin-top: 20px;
        }
        .footer {
            margin-top: 40px;
            padding-top: 20px;
            border-top: 1px solid #eee;
            text-align: center;
            color: #999;
            font-size: 12px;
        }
        .warning {
            background-color: #f8d7da;
            border: 1px solid #f5c6cb;
            border-radius: 4px;
            padding: 15px;
            margin: 20px 0;
            color: #721c24;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <img src="{{.LogoURL}}" alt="{{.CompanyName}}" class="logo">
            <h1 class="title">密码重置</h1>
        </div>
        
        <div class="content">
            <p>您好！</p>
            <p>我们收到了您的密码重置请求。请使用以下验证码来重置您的{{.CompanyName}}账户密码：</p>
            
            <div class="verification-code">
                <div class="code">{{.Code}}</div>
            </div>
            
            <div class="warning">
                <strong>安全提醒：</strong>
                <ul style="margin: 10px 0; padding-left: 20px;">
                    <li>验证码有效期为10分钟</li>
                    <li>请勿将验证码告诉他人</li>
                    <li>如果您没有申请密码重置，请立即联系客服</li>
                    <li>为了账户安全，建议设置复杂密码</li>
                </ul>
            </div>
            
            <p class="note">
                如果您在使用过程中遇到任何问题，请联系我们的客服团队。
            </p>
        </div>
        
        <div class="footer">
            <p>此邮件由系统自动发送，请勿回复。</p>
            <p>&copy; 2024 {{.CompanyName}}. All rights reserved.</p>
        </div>
    </div>
</body>
</html>`

	// 替换模板变量
	template = strings.ReplaceAll(template, "{{.LogoURL}}", t.logoURL)
	template = strings.ReplaceAll(template, "{{.CompanyName}}", t.companyName)
	template = strings.ReplaceAll(template, "{{.Code}}", code)

	return template
}

// renderVerificationCodeText 渲染注册验证码纯文本模板
func (t *TemplateService) renderVerificationCodeText(code string) string {
	return fmt.Sprintf(`
【%s】邮箱验证码

您好！

感谢您注册%s账户。为了确保您的邮箱地址有效，请使用以下验证码完成注册：

验证码：%s

重要提醒：
- 验证码有效期为10分钟
- 请勿将验证码告诉他人
- 如果您没有注册%s账户，请忽略此邮件

如果您在使用过程中遇到任何问题，请联系我们的客服团队。

此邮件由系统自动发送，请勿回复。
© 2024 %s. All rights reserved.
`, t.companyName, t.companyName, code, t.companyName, t.companyName)
}

// renderPasswordResetText 渲染密码重置纯文本模板
func (t *TemplateService) renderPasswordResetText(code string) string {
	return fmt.Sprintf(`
【%s】密码重置验证码

您好！

我们收到了您的密码重置请求。请使用以下验证码来重置您的%s账户密码：

验证码：%s

安全提醒：
- 验证码有效期为10分钟
- 请勿将验证码告诉他人
- 如果您没有申请密码重置，请立即联系客服
- 为了账户安全，建议设置复杂密码

如果您在使用过程中遇到任何问题，请联系我们的客服团队。

此邮件由系统自动发送，请勿回复。
© 2024 %s. All rights reserved.
`, t.companyName, t.companyName, code, t.companyName)
}
