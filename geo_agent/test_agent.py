"""
Test script for geo_agent service

This script tests the OpenAI-compatible API endpoints
"""
from openai import OpenAI
import time
import sys

def test_basic_chat(client):
    """Test basic chat completion"""
    print("\n" + "="*50)
    print("测试 1: 基本聊天补全")
    print("="*50)
    
    try:
        start = time.time()
        response = client.chat.completions.create(
            model="qwen3-max",
            messages=[
                {"role": "user", "content": "1+1等于几？请简短回答"}
            ]
        )
        latency = (time.time() - start) * 1000
        
        print(f"✅ 成功")
        print(f"回答: {response.choices[0].message.content}")
        print(f"Tokens: {response.usage.total_tokens} (prompt: {response.usage.prompt_tokens}, completion: {response.usage.completion_tokens})")
        print(f"延迟: {latency:.2f}ms")
        return True
    except Exception as e:
        print(f"❌ 失败: {e}")
        return False

def test_system_prompt(client):
    """Test with system prompt"""
    print("\n" + "="*50)
    print("测试 2: 系统提示词")
    print("="*50)
    
    try:
        start = time.time()
        response = client.chat.completions.create(
            model="qwen3-max",
            messages=[
                {"role": "system", "content": "你是一个数学老师，回答要简洁清晰"},
                {"role": "user", "content": "什么是质数？"}
            ],
            temperature=0.7
        )
        latency = (time.time() - start) * 1000
        
        print(f"✅ 成功")
        print(f"回答: {response.choices[0].message.content[:200]}...")
        print(f"Tokens: {response.usage.total_tokens}")
        print(f"延迟: {latency:.2f}ms")
        return True
    except Exception as e:
        print(f"❌ 失败: {e}")
        return False

def test_streaming(client):
    """Test streaming response"""
    print("\n" + "="*50)
    print("测试 3: 流式响应")
    print("="*50)
    
    try:
        start = time.time()
        stream = client.chat.completions.create(
            model="qwen3-max",
            messages=[
                {"role": "user", "content": "从1数到5，每个数字一行"}
            ],
            stream=True
        )
        
        print("流式输出:")
        content = ""
        for chunk in stream:
            if chunk.choices[0].delta.content:
                content += chunk.choices[0].delta.content
                print(chunk.choices[0].delta.content, end="", flush=True)
        
        latency = (time.time() - start) * 1000
        print(f"\n✅ 成功")
        print(f"总长度: {len(content)} 字符")
        print(f"延迟: {latency:.2f}ms")
        return True
    except Exception as e:
        print(f"\n❌ 失败: {e}")
        return False

def test_multi_turn(client):
    """Test multi-turn conversation"""
    print("\n" + "="*50)
    print("测试 4: 多轮对话")
    print("="*50)
    
    try:
        start = time.time()
        response = client.chat.completions.create(
            model="qwen3-max",
            messages=[
                {"role": "user", "content": "我的名字是小明"},
                {"role": "assistant", "content": "你好，小明！很高兴认识你。"},
                {"role": "user", "content": "我叫什么名字？"}
            ]
        )
        latency = (time.time() - start) * 1000
        
        print(f"✅ 成功")
        print(f"回答: {response.choices[0].message.content}")
        print(f"延迟: {latency:.2f}ms")
        
        # Check if the model remembers the name
        if "小明" in response.choices[0].message.content:
            print("✅ 上下文理解正确")
        else:
            print("⚠️  上下文理解可能有问题")
        
        return True
    except Exception as e:
        print(f"❌ 失败: {e}")
        return False

def test_parameters(client):
    """Test different parameters"""
    print("\n" + "="*50)
    print("测试 5: 参数配置")
    print("="*50)
    
    try:
        # Test with low temperature
        start = time.time()
        response = client.chat.completions.create(
            model="qwen3-max",
            messages=[
                {"role": "user", "content": "说一个数字"}
            ],
            temperature=0.1,
            max_tokens=10
        )
        latency = (time.time() - start) * 1000
        
        print(f"✅ 成功")
        print(f"回答 (temperature=0.1): {response.choices[0].message.content}")
        print(f"Tokens: {response.usage.total_tokens}")
        print(f"延迟: {latency:.2f}ms")
        return True
    except Exception as e:
        print(f"❌ 失败: {e}")
        return False

def test_models_endpoint(client):
    """Test /v1/models endpoint"""
    print("\n" + "="*50)
    print("测试 6: 模型列表接口")
    print("="*50)
    
    try:
        models = client.models.list()
        print(f"✅ 成功")
        print(f"可用模型:")
        for model in models.data:
            print(f"  - {model.id} (owned by: {model.owned_by})")
        return True
    except Exception as e:
        print(f"❌ 失败: {e}")
        return False

def main():
    """Run all tests"""
    print("="*50)
    print("geo_agent API 测试套件")
    print("="*50)
    print("服务地址: http://localhost:8100")
    print()
    
    # Initialize client
    client = OpenAI(
        base_url="http://localhost:8100/v1",
        api_key="test"  # Dummy API key
    )
    
    # Run tests
    tests = [
        ("基本聊天", test_basic_chat),
        ("系统提示词", test_system_prompt),
        ("流式响应", test_streaming),
        ("多轮对话", test_multi_turn),
        ("参数配置", test_parameters),
        ("模型列表", test_models_endpoint),
    ]
    
    results = []
    for name, test_func in tests:
        try:
            result = test_func(client)
            results.append((name, result))
        except Exception as e:
            print(f"\n❌ 测试异常: {e}")
            results.append((name, False))
        
        # Wait a bit between tests
        time.sleep(1)
    
    # Summary
    print("\n" + "="*50)
    print("测试摘要")
    print("="*50)
    
    passed = sum(1 for _, result in results if result)
    total = len(results)
    
    for name, result in results:
        status = "✅ 通过" if result else "❌ 失败"
        print(f"{name}: {status}")
    
    print(f"\n总计: {passed}/{total} 通过")
    
    if passed == total:
        print("\n🎉 所有测试通过!")
        return 0
    else:
        print(f"\n⚠️  {total - passed} 个测试失败")
        return 1

if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print("\n\n测试被中断")
        sys.exit(1)
    except Exception as e:
        print(f"\n\n测试异常: {e}")
        sys.exit(1)
