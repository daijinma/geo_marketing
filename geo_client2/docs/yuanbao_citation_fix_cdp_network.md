# 元宝引用URL抓取修复 - CDP Network完整响应方案

## 📋 问题总结
**原始问题**: 元宝(Yuanbao)平台的SSE流式响应中包含`searchGuid`类型的citations数据(32个引用,JSON长度25-29KB),但后端通过JavaScript console.log传输时被CDP截断,导致`unexpected end of JSON input`错误。

**根本原因**: Chrome DevTools Protocol的`RuntimeConsoleAPICalled`事件对单条console.log输出有长度限制,即使JavaScript代码中实现了分块逻辑(`MAX_CHUNK_SIZE`),但CDP在传输完整日志行时会整体截断。

## ✅ 解决方案: CDP Network.getResponseBody
**核心思路**: 完全绕过console.log传输通道,直接通过CDP的`Network.getResponseBody` API获取完整HTTP响应体。

### 实现架构
```
Yuanbao Server (SSE Stream)
         ↓
   Chrome Browser (rod-managed)
         ↓
  CDP Network Events (3-layer capture)
    ├─ NetworkResponseReceived → 标记SSE请求
    ├─ NetworkLoadingFinished  → 获取完整body
    └─ EventSourceMessageReceived → 实时fallback
         ↓
    parseSSEData() → Citations
```

### 关键代码变更
**文件**: `backend/provider/yuanbao.go` (line 366-467)

#### 变更1: 新增NetworkResponseReceived监听器
```go
// Track SSE ResponseReceived events
go func() {
    page.EachEvent(func(e *proto.NetworkResponseReceived) bool {
        // 通过MIMEType和URL模式识别SSE响应
        if e.Response.MIMEType == "text/event-stream" || 
           strings.Contains(e.Response.URL, "/chat") {
            sseRequestIDs.Store(string(e.RequestID), true)
        }
        return false
    })()
}()
```

#### 变更2: 新增NetworkLoadingFinished处理器
```go
// Track LoadingFinished to fetch complete response body
go func() {
    page.EachEvent(func(e *proto.NetworkLoadingFinished) bool {
        reqID := string(e.RequestID)
        if _, isSSE := sseRequestIDs.Load(reqID); isSSE {
            // 关键调用: 获取完整响应体
            body, err := proto.NetworkGetResponseBody{
                RequestID: e.RequestID
            }.Call(page)
            
            if err == nil {
                // 解析所有SSE行
                lines := strings.Split(body.Body, "\n")
                for _, line := range lines {
                    parseSSEData(line)
                }
            }
            sseRequestIDs.Delete(reqID)
        }
        return false
    })()
}()
```

#### 变更3: 保留EventSourceMessageReceived作为fallback
```go
// Keep original EventSourceMessageReceived as fallback
go func() {
    page.EachEvent(func(e *proto.NetworkEventSourceMessageReceived) bool {
        d.logger.DebugWithContext(ctx, "[YUANBAO-CDP-SSE-FALLBACK] ...")
        parseSSEData(e.Data)
        return false
    })()
}()
```

## 🔑 技术细节

### 为什么这个方案有效?
1. **完整数据**: `NetworkGetResponseBody`返回完整HTTP body,无长度限制
2. **时机正确**: `LoadingFinished`事件在完整响应接收后触发
3. **无依赖**: 不依赖JavaScript注入、console.log、或浏览器扩展

### 三层捕获策略
| 层级 | 事件类型 | 触发时机 | 数据完整性 | 用途 |
|-----|---------|---------|-----------|------|
| **1. NetworkLoadingFinished** | CDP Network | 响应完全加载后 | ✅ 完整 | **主要方案** |
| **2. EventSourceMessageReceived** | CDP Network | 实时SSE消息 | ⚠️ 可能截断 | 实时fallback |
| **3. JavaScript Hijack** | Runtime Console | 异步fetch读取 | ❌ CDP截断 | 已弃用 |

### sync.Map使用
```go
sseRequestIDs := &sync.Map{} // map[string]bool
```
- **并发安全**: 多个goroutine同时操作不会panic
- **临时存储**: ResponseReceived存入,LoadingFinished删除
- **作用**: 关联RequestID到SSE响应,避免处理非SSE请求

## 📊 预期效果

### 修复前日志 (典型失败)
```
[WARN] [YUANBAO-SSE] JSON unmarshal failed (CRITICAL)
data_len: 28959
data_suffix: "��,"海湾 G4"的差异化特点与适配企业类型,中企从 "抢大项目" 到 "本土"
error: "unexpected end of JSON input"
```

### 修复后日志 (预期成功)
```
[INFO] [YUANBAO-CDP-NETWORK] ✅ Complete SSE body fetched
request_id: "12345.67"
body_len: 32768
preview: "data: {\"type\":\"searchGuid\",\"docs\":[{\"url\":\"https://...\",\"title\":\"...\"..."

[INFO] [YUANBAO-SSE] searchGuid packet received
guid: "abc-def-123"
total_docs: 32
new_urls: 27

[INFO] [YUANBAO-SSE] Added citation
url: "https://example.com/article1"
title: "专家网络服务商深度分析"
```

## 🧪 测试方法

### 1. 编译验证
```bash
cd /Users/cow/Desktop/p_space/geo_marketing/geo_client2
go build ./backend/provider/...
# ✅ 无输出 = 编译成功
```

### 2. 启动应用
```bash
# 停止旧进程
pkill -f "wails dev" && sleep 2

# 启动dev server
make dev
```

### 3. 执行搜索测试
- **搜索关键词**: "专家网络服务商" / "凯盛融英" / "跨境电商平台"
- **选择账号**: 任意已登录元宝账号
- **点击**: "开始搜索"

### 4. 实时日志监控
```bash
tail -f ~/.duanjiegeo/logs/app.log | grep -E "(CDP-NETWORK|searchGuid|Citations captured|Added citation)"
```

### 成功标志 (MUST SEE)
```
✅ [YUANBAO-CDP-NETWORK] SSE Response detected
✅ [YUANBAO-CDP-NETWORK] SSE LoadingFinished, fetching body
✅ [YUANBAO-CDP-NETWORK] ✅ Complete SSE body fetched | body_len: 30000+
✅ [YUANBAO-SSE] searchGuid packet received | total_docs: 32
✅ [YUANBAO-SSE] ✅ Citations captured | new_citations: 27+
✅ [YUANBAO-SSE] Added citation | url: https://...
```

### 失败标志 (需要调查)
```
❌ [YUANBAO-CDP-NETWORK] Failed to get response body | error: ...
❌ [YUANBAO-SSE] JSON unmarshal failed | error: "unexpected end of JSON input"
❌ No "Citations captured" log after 10 seconds
```

## 🔍 Troubleshooting

### 问题1: 仍然看到 "unexpected end of JSON input"
**原因**: `NetworkGetResponseBody`未触发,可能URL过滤不匹配
**解决**:
1. 检查日志中的`[YUANBAO-CDP-NETWORK] SSE Response detected`
2. 如果没有,说明URL过滤条件不匹配
3. 修改line 384的条件:
```go
if e.Response.MIMEType == "text/event-stream" || 
   strings.Contains(e.Response.URL, "yuanbao.tencent.com") {
```

### 问题2: body_len显示为0或很小
**原因**: 响应可能是chunked编码,或LoadingFinished触发过早
**解决**:
1. 增加延迟: 在`NetworkGetResponseBody`调用前`time.Sleep(500*time.Millisecond)`
2. 检查`e.Response.Headers`中的`Transfer-Encoding`

### 问题3: 日志显示 "Failed to enable network"
**原因**: CDP连接问题
**解决**:
```go
// 在LaunchBrowser后立即验证
if err := page.Browser().Context(ctx); err != nil {
    d.logger.Error("Browser context lost: " + err.Error())
}
```

## 📁 相关文件
- **主要修改**: `/Users/cow/Desktop/p_space/geo_marketing/geo_client2/backend/provider/yuanbao.go` (line 366-467)
- **日志路径**: `~/.duanjiegeo/logs/app.log`
- **数据库**: `~/.duanjiegeo/cache.db` (表: `citations`)

## 🚀 后续优化建议
1. **性能**: 如果单次搜索有多个SSE请求,考虑限制sseRequestIDs大小
2. **错误恢复**: 添加body fetch重试逻辑(当前仅尝试1次)
3. **监控**: 添加Prometheus metrics跟踪截断率和成功率
4. **测试**: 编写单元测试mock CDP events

## 🎯 总结
这个方案通过**直接使用CDP Network API获取完整响应体**,彻底绕过了console.log的长度限制问题,是修复元宝citations抓取失败的**最佳解决方案**。相比JavaScript分块方案,它更可靠、更简洁、更易维护。

---
**修复时间**: 2026-02-01  
**影响范围**: 元宝Provider的所有搜索任务  
**向后兼容**: 是 (保留原有fallback逻辑)
