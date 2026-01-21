#!/usr/bin/env python3
"""
测试日志增强功能
"""
import asyncio
import httpx
import json

# 测试配置
BASE_URL = "http://localhost:8001"
API_KEY = "your-api-key-here"  # 实际使用时需要配置真实的 API Key

async def test_non_streaming():
    """测试非流式请求的日志输出"""
    print("\n" + "="*80)
    print("🧪 测试非流式请求")
    print("="*80 + "\n")
    
    url = f"{BASE_URL}/v1/chat/completions"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_KEY}"
    }
    
    data = {
        "model": "qwen-plus",
        "messages": [
            {
                "role": "system",
                "content": "你是一个有帮助的助手。"
            },
            {
                "role": "user",
                "content": "你好，今天天气怎么样？"
            }
        ],
        "temperature": 0.7,
        "max_tokens": 100,
        "stream": False
    }
    
    async with httpx.AsyncClient() as client:
        try:
            response = await client.post(url, json=data, headers=headers, timeout=30.0)
            print(f"\n响应状态码: {response.status_code}")
            
            if response.status_code == 200:
                result = response.json()
                print(f"响应内容: {json.dumps(result, ensure_ascii=False, indent=2)}")
            else:
                print(f"错误: {response.text}")
                
        except Exception as e:
            print(f"请求失败: {str(e)}")

async def test_streaming():
    """测试流式请求的日志输出"""
    print("\n" + "="*80)
    print("🧪 测试流式请求")
    print("="*80 + "\n")
    
    url = f"{BASE_URL}/v1/chat/completions"
    headers = {
        "Content-Type": "application/json",
        "Authorization": f"Bearer {API_KEY}"
    }
    
    data = {
        "model": "qwen-plus",
        "messages": [
            {
                "role": "user",
                "content": "写一首关于春天的短诗"
            }
        ],
        "temperature": 0.8,
        "stream": True
    }
    
    async with httpx.AsyncClient() as client:
        try:
            async with client.stream("POST", url, json=data, headers=headers, timeout=30.0) as response:
                print(f"\n响应状态码: {response.status_code}")
                
                if response.status_code == 200:
                    print("\n开始接收流式响应:")
                    async for line in response.aiter_lines():
                        if line.startswith("data: "):
                            chunk_data = line[6:]
                            if chunk_data != "[DONE]":
                                try:
                                    chunk = json.loads(chunk_data)
                                    content = chunk.get("choices", [{}])[0].get("delta", {}).get("content", "")
                                    if content:
                                        print(content, end="", flush=True)
                                except json.JSONDecodeError:
                                    pass
                    print("\n\n流式响应结束")
                else:
                    error_text = await response.aread()
                    print(f"错误: {error_text.decode()}")
                    
        except Exception as e:
            print(f"请求失败: {str(e)}")

async def test_health():
    """测试健康检查端点"""
    print("\n" + "="*80)
    print("🧪 测试健康检查")
    print("="*80 + "\n")
    
    url = f"{BASE_URL}/health"
    
    async with httpx.AsyncClient() as client:
        try:
            response = await client.get(url, timeout=5.0)
            print(f"响应状态码: {response.status_code}")
            print(f"响应内容: {response.text}")
        except Exception as e:
            print(f"请求失败: {str(e)}")

async def main():
    """主测试函数"""
    print("\n" + "🎯"*40)
    print("geo_agent 日志增强功能测试")
    print("🎯"*40)
    
    print("\n提示: 请确保以下配置正确：")
    print(f"1. geo_agent 服务运行在 {BASE_URL}")
    print(f"2. DASHSCOPE_API_KEY 已配置")
    print("\n按回车继续...", end="")
    # input()  # 取消注释以等待用户输入
    
    # 测试健康检查
    await test_health()
    
    # 等待一下
    await asyncio.sleep(1)
    
    # 测试非流式请求
    # 注意: 需要配置真实的 API Key 才能测试
    # await test_non_streaming()
    
    # 等待一下
    # await asyncio.sleep(2)
    
    # 测试流式请求
    # await test_streaming()
    
    print("\n" + "="*80)
    print("✅ 测试完成")
    print("="*80)
    print("\n请检查控制台输出，应该看到以下日志信息：")
    print("1. 🔵 HTTP 请求信息")
    print("2. 📨 OpenAI 格式请求")
    print("3. 📤 中转请求详情")
    print("4. 🚀 DashScope API 调用")
    print("5. ✅ DashScope API 响应")
    print("6. 📥 中转响应详情")
    print("7. ✨ OpenAI 格式响应")
    print("8. 🟢 HTTP 响应信息")
    print("\n同时检查 logs/ 目录下的日志文件：")
    print("- logs/access.log")
    print("- logs/error.log")
    print("- logs/qwen_calls.log")
    print()

if __name__ == "__main__":
    asyncio.run(main())
