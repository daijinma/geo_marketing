# Rod 网络监听成功要点

## 问题背景

使用 go-rod 的 `page.EachEvent` 监听 CDP 网络事件时，发现完全收不到任何事件。

## 根本原因

`EachEvent` **返回的是一个函数**，必须调用这个函数才会启动监听。

```go
// EachEvent 的签名
func (p *Page) EachEvent(callbacks ...interface{}) (wait func())
```

## 错误 vs 正确写法

### 错误写法（监听永远不会启动）

```go
go func() {
    page.EachEvent(func(e *proto.NetworkRequestWillBeSent) bool {
        fmt.Println("Request:", e.Request.URL)
        return false
    })  // ❌ 没有调用返回的 wait 函数
}()
```

### 正确写法

```go
go func() {
    page.EachEvent(func(e *proto.NetworkRequestWillBeSent) bool {
        fmt.Println("Request:", e.Request.URL)
        return false
    })()  // ✅ 加上 () 调用返回的 wait 函数
}()

// 或更简洁的写法
go page.EachEvent(func(e *proto.NetworkRequestWillBeSent) bool {
    fmt.Println("Request:", e.Request.URL)
    return false
})()
```

## 最小可运行模板

```go
page := browser.Context(ctx).MustPage()

// 1. 启用 Network 域
proto.NetworkEnable{}.Call(page)

// 2. 启动网络监听（goroutine + EachEvent()）
go page.EachEvent(
    func(e *proto.NetworkRequestWillBeSent) bool {
        fmt.Println("Request:", e.Request.URL)
        return false  // false = 继续监听
    },
    func(e *proto.NetworkResponseReceived) bool {
        fmt.Println("Response:", e.Response.URL, e.Response.Status)
        return false
    },
)()  // <-- 必须调用！

// 3. 导航到页面
page.MustNavigate("https://example.com")
```

## 成功三要素

| 步骤 | 说明 |
|------|------|
| `proto.NetworkEnable{}.Call(page)` | 告诉 Chrome 开始发送 Network 域事件 |
| `go page.EachEvent(...)()` | goroutine 里阻塞监听，末尾 `()` 是启动关键 |
| 在 `MustNavigate` 之前注册监听 | 确保不漏掉页面加载的请求 |

## 官方示例参考

来自 `page.go` 的注释：

```go
// Here's an example to dismiss all dialogs/alerts on the page:
//
//  go page.EachEvent(func(e *proto.PageJavascriptDialogOpening) {
//      _ = proto.PageHandleJavaScriptDialog{ Accept: false, PromptText: ""}.Call(page)
//  })()  // <-- 注意这里的 ()
```

## 总结

**`EachEvent` 返回一个阻塞函数，必须用 `()` 调用它才会开始监听事件。**
