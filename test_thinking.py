#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
深度思考功能测试脚本
测试 /v1/chat/completions 接口的 thinking 功能
"""

import requests
import json
import time
import sys

# 配置
BASE_URL = "http://localhost:8080"
API_KEY = "ak_f246de96ea05dca6e3a1e4c82f7adb25a0a1b8156a737af9c03483aa31db6b15"  # 请替换为实际的API密钥

def test_non_streaming_thinking():
    """测试非流式深度思考"""
    print("🧠 测试非流式深度思考...")
    
    url = f"{BASE_URL}/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "gpt-3.5-turbo",
        "messages": [
            {
                "role": "user", 
                "content": "解释量子计算的基本原理，并分析其优势"
            }
        ],
        "thinking": {
            "enabled": True,
            "show_process": True,
            "language": "zh",
            "max_tokens": 1000
        }
    }
    
    try:
        response = requests.post(url, headers=headers, json=payload, timeout=30)
        
        print(f"状态码: {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            print("✅ 非流式请求成功")
            print(f"模型: {data.get('model', 'N/A')}")
            print(f"内容: {data['choices'][0]['message']['content'][:200]}...")
            
            # 检查是否包含思考过程
            content = data['choices'][0]['message']['content']
            if '<thinking>' in content or '【思考】' in content:
                print("✅ 检测到思考过程标签")
            else:
                print("ℹ️  响应中未发现思考标签")
                
        else:
            print(f"❌ 请求失败: {response.status_code}")
            print(f"错误信息: {response.text}")
            
    except requests.exceptions.RequestException as e:
        print(f"❌ 请求异常: {e}")

def test_streaming_thinking():
    """测试流式深度思考"""
    print("\n🚀 测试流式深度思考...")
    
    url = f"{BASE_URL}/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "gpt-3.5-turbo",
        "messages": [
            {
                "role": "user", 
                "content": "什么是人工智能？请详细解释其发展历程和未来前景"
            }
        ],
        "stream": True,
        "thinking": {
            "enabled": True,
            "show_process": True,
            "language": "zh",
            "max_tokens": 1500
        }
    }
    
    try:
        response = requests.post(url, headers=headers, json=payload, stream=True, timeout=60)
        
        print(f"状态码: {response.status_code}")
        if response.status_code == 200:
            print("✅ 流式请求开始")
            print("📡 接收流式数据:")
            print("-" * 50)
            
            thinking_chunks = []
            response_chunks = []
            
            for line in response.iter_lines():
                if line:
                    line_str = line.decode('utf-8')
                    if line_str.startswith('data: '):
                        data_str = line_str[6:].strip()
                        if data_str == '[DONE]':
                            print("\n✅ 流式传输完成")
                            break
                        
                        try:
                            chunk_data = json.loads(data_str)
                            
                            # 检查是否有内容类型字段
                            content_type = chunk_data.get('content_type', 'response')
                            content = chunk_data.get('content', '')
                            
                            if content_type == 'thinking':
                                thinking_chunks.append(content)
                                print(f"🧠 [思考] {content}", end='', flush=True)
                            elif content_type == 'response':
                                response_chunks.append(content)
                                print(f"💭 [回答] {content}", end='', flush=True)
                            else:
                                # 兼容旧格式
                                if 'choices' in chunk_data and len(chunk_data['choices']) > 0:
                                    delta = chunk_data['choices'][0].get('delta', {})
                                    if 'content' in delta:
                                        content = delta['content']
                                        response_chunks.append(content)
                                        print(f"📝 {content}", end='', flush=True)
                                        
                        except json.JSONDecodeError:
                            print(f"⚠️  无法解析JSON: {data_str}")
            
            print(f"\n\n📊 统计:")
            print(f"思考内容片段: {len(thinking_chunks)}")
            print(f"回答内容片段: {len(response_chunks)}")
            
            if thinking_chunks:
                print(f"✅ 成功接收到思考过程内容")
            if response_chunks:
                print(f"✅ 成功接收到回答内容")
                
        else:
            print(f"❌ 请求失败: {response.status_code}")
            print(f"错误信息: {response.text}")
            
    except requests.exceptions.RequestException as e:
        print(f"❌ 请求异常: {e}")

def test_thinking_disabled():
    """测试禁用思考模式"""
    print("\n🚫 测试禁用思考模式...")
    
    url = f"{BASE_URL}/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "gpt-3.5-turbo",
        "messages": [
            {
                "role": "user", 
                "content": "简单介绍一下机器学习"
            }
        ],
        "thinking": {
            "enabled": False
        }
    }
    
    try:
        response = requests.post(url, headers=headers, json=payload, timeout=30)
        
        print(f"状态码: {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            print("✅ 禁用思考模式请求成功")
            content = data['choices'][0]['message']['content']
            print(f"内容: {content[:200]}...")
            
            # 检查不应该包含思考过程
            if '<thinking>' not in content and '【思考】' not in content:
                print("✅ 确认未包含思考过程")
            else:
                print("⚠️  意外发现思考标签")
                
        else:
            print(f"❌ 请求失败: {response.status_code}")
            print(f"错误信息: {response.text}")
            
    except requests.exceptions.RequestException as e:
        print(f"❌ 请求异常: {e}")

def test_english_thinking():
    """测试英文思考模式"""
    print("\n🇺🇸 测试英文思考模式...")
    
    url = f"{BASE_URL}/v1/chat/completions"
    headers = {
        "Authorization": f"Bearer {API_KEY}",
        "Content-Type": "application/json"
    }
    
    payload = {
        "model": "gpt-3.5-turbo",
        "messages": [
            {
                "role": "user", 
                "content": "Explain the concept of blockchain technology"
            }
        ],
        "thinking": {
            "enabled": True,
            "show_process": True,
            "language": "en",
            "max_tokens": 800
        }
    }
    
    try:
        response = requests.post(url, headers=headers, json=payload, timeout=30)
        
        print(f"状态码: {response.status_code}")
        if response.status_code == 200:
            data = response.json()
            print("✅ 英文思考模式请求成功")
            content = data['choices'][0]['message']['content']
            print(f"内容: {content[:200]}...")
            
            # 检查是否包含英文思考提示
            if 'think step by step' in content.lower() or '<thinking>' in content:
                print("✅ 检测到英文思考过程")
            else:
                print("ℹ️  未明确检测到英文思考标记")
                
        else:
            print(f"❌ 请求失败: {response.status_code}")
            print(f"错误信息: {response.text}")
            
    except requests.exceptions.RequestException as e:
        print(f"❌ 请求异常: {e}")

def test_health_check():
    """测试服务健康状态"""
    print("🏥 检查服务健康状态...")
    
    try:
        # 尝试一个简单的请求来检查服务是否正常运行
        response = requests.get(f"{BASE_URL}/health", timeout=5)
        if response.status_code == 200:
            print("✅ 服务运行正常")
            return True
        else:
            print(f"⚠️  健康检查返回状态码: {response.status_code}")
    except requests.exceptions.RequestException:
        pass
    
    # 如果没有专门的健康检查端点，尝试一个简单的聊天请求
    try:
        url = f"{BASE_URL}/v1/chat/completions"
        headers = {
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json"
        }
        payload = {
            "model": "gpt-3.5-turbo",
            "messages": [{"role": "user", "content": "hello"}],
            "max_tokens": 10
        }
        response = requests.post(url, headers=headers, json=payload, timeout=10)
        if response.status_code in [200, 401, 403]:  # 200成功，401/403说明服务在运行但认证失败
            print("✅ 服务运行正常")
            return True
        else:
            print(f"❌ 服务可能异常，状态码: {response.status_code}")
            return False
    except requests.exceptions.RequestException as e:
        print(f"❌ 无法连接到服务: {e}")
        return False

def main():
    """主函数"""
    print("🎯 AI API Gateway 深度思考功能测试")
    print("=" * 60)
    
    # 健康检查
    if not test_health_check():
        print("\n⚠️  服务可能未运行或配置错误")
        print("请确保:")
        print("1. 服务运行在 http://localhost:8080")
        print("2. API密钥配置正确")
        print("3. 防火墙没有阻止连接")
        
        user_input = input("\n是否继续测试? (y/N): ")
        if user_input.lower() not in ['y', 'yes']:
            sys.exit(1)
    
    print(f"\n🔑 使用API密钥: {API_KEY[:10]}...")
    print("ℹ️  如果遇到认证错误，请修改脚本中的API_KEY变量\n")
    
    # 执行各项测试
    test_non_streaming_thinking()
    test_streaming_thinking()
    test_thinking_disabled()
    test_english_thinking()
    
    print("\n" + "=" * 60)
    print("🎉 所有测试完成!")
    print("\n💡 提示:")
    print("- 如果遇到401错误，请检查API密钥是否正确")
    print("- 如果遇到连接错误，请确认服务是否运行在8080端口")
    print("- 思考功能需要后端AI服务支持，请确认模型可用")

if __name__ == "__main__":
    main()