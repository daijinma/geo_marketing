# RPA Publishing Skill - Multi-Platform Article Publishing

This document describes the RPA (Robotic Process Automation) capabilities for publishing articles to Chinese self-media platforms. It serves as a reference for implementing browser automation workflows using go-rod or similar tools.

## Overview

The publishing system supports multiple platforms with both hard-coded automation flows and AI-assisted fallback mechanisms.

### End-to-End Publishing Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        RPA PUBLISHING WORKFLOW                               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌────────────┐ │
│  │ 1. LOGIN     │───▶│ 2. NAVIGATE  │───▶│ 3. FILL      │───▶│ 4. PUBLISH │ │
│  │   CHECK      │    │   TO EDITOR  │    │   CONTENT    │    │   SUBMIT   │ │
│  └──────────────┘    └──────────────┘    └──────────────┘    └────────────┘ │
│         │                   │                   │                   │       │
│         ▼                   ▼                   ▼                   ▼       │
│  Provider.Check     browser.MustPage    Title + Content      Click 发布    │
│  LoginStatus()      (editor URL)        + Summary + Tags     Wait redirect │
│         │                                       │                           │
│         ▼                                       ▼                           │
│  ┌──────────────┐                      ┌──────────────────┐                 │
│  │ Not Logged   │                      │ Editor Types:    │                 │
│  │ In? ────────▶│ StartLogin()         │ • Draft.js       │                 │
│  │              │ Manual auth          │ • Quill.js       │                 │
│  └──────────────┘                      │ • ContentEditable│                 │
│                                        │ • iframe-based   │                 │
│                                        └──────────────────┘                 │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Component Mapping

| Step | Component | Code Location |
|------|-----------|---------------|
| Login Check | Provider | `backend/provider/{platform}.go` → `CheckLoginStatus()` |
| Login Execute | Provider | `backend/provider/{platform}.go` → `StartLogin()` |
| Navigate & Fill | Publisher | `backend/publisher/{platform}.go` → `Publish()` |
| AI Fallback | Publisher | `backend/publisher/publishers.go` → `AIAssistPublish()` |

### Supported Platforms

| Platform | Code | Home URL | Editor URL | Status |
|----------|------|----------|------------|--------|
| 知乎专栏 | `zhihu` | https://zhuanlan.zhihu.com | https://zhuanlan.zhihu.com/write | Implemented |
| 搜狐号 | `sohu` | https://mp.sohu.com | https://mp.sohu.com/mpfe/v4/contentManagement/news/addarticle | Implemented |
| CSDN | `csdn` | https://www.csdn.net | https://mp.csdn.net/mp_blog/creation/editor | Implemented |
| 企鹅号 | `qie` | https://om.qq.com | https://om.qq.com/main/creation/article | Implemented |
| 百家号 | `baijiahao` | https://baijiahao.baidu.com | https://baijiahao.baidu.com/builder/rc/edit?type=news | Implemented |
| 小红书 | `xiaohongshu` | https://creator.xiaohongshu.com | (via AI assist) | AI-assisted only |

### Quick Reference - Key Selectors

| Platform | Title Selector | Content Selector | Publish Button |
|----------|---------------|------------------|----------------|
| 知乎 | `textarea[placeholder*="请输入标题"]` | `div.public-DraftEditor-content[contenteditable="true"]` | `button` 含 "发布" |
| 搜狐号 | `.publish-title input[type="text"]` | `#editor .ql-editor` | `li.publish-report-btn` |
| CSDN | `textarea[placeholder*="请输入文章标题"]` | `main iframe` → `body[contenteditable]` | `button` 含 "发布博客" |
| 企鹅号 | `[contenteditable][placeholder*="请输入标题"]` | `main [contenteditable="true"]` | `button "发布"` |
| 百家号 | `[contenteditable][placeholder*="请输入标题"]` | `iframe` → `body[contenteditable]` | `button "发布"` |

### Quick Reference - Login Detection

| Platform | Logged In Indicator | Not Logged In Indicator |
|----------|--------------------|-----------------------|
| 知乎 | "写文章" 按钮, Avatar | "登录\|注册" 按钮 |
| 搜狐号 | "发布内容" 按钮, Avatar | "登录\|注册" 按钮 |
| CSDN | "创作" 链接 (href含mp.csdn.net), Avatar | "登录\|注册" 提示 |
| 企鹅号 | "发布" 按钮, Avatar | "QQ登录\|微信登录" 按钮 |
| 百家号 | "发布作品", Avatar | "登录\|百度账号登录" 按钮 |

---

## Login Flow & Detection

This section documents the login flow and status detection for each platform.

### Login Detection Architecture

Each platform has a corresponding Provider implementation in `geo_client2/backend/provider/` that handles:
1. **Login Status Detection** - `CheckLoginStatus()` method
2. **Browser Profile Isolation** - User data stored in `~/.geo_client2/browser_data/{platform}/{account_id}/`
3. **Login URL Navigation** - Configured in `config/platforms.go`

### Platform Login URLs

| Platform | Login URL | Home URL |
|----------|-----------|----------|
| 知乎 | `https://www.zhihu.com/signin` | `https://www.zhihu.com` |
| 搜狐号 | `https://mp.sohu.com/mpfe/v4/login` | `https://mp.sohu.com` |
| CSDN | `https://passport.csdn.net/login` | `https://www.csdn.net` |
| 企鹅号 | `https://om.qq.com/userAuth/index` | `https://om.qq.com` |
| 百家号 | `https://baijiahao.baidu.com/builder/theme/bjh/login` | `https://baijiahao.baidu.com` |

---

### 1. 知乎登录 (Zhihu)

**Login URL**: `https://www.zhihu.com/signin`

**Provider File**: `geo_client2/backend/provider/zhihu.go`

#### Login Methods
- 验证码登录 (SMS verification)
- 密码登录 (Password)
- QQ 登录 (OAuth)
- 微信登录 (WeChat OAuth)

#### Login Page DOM Elements

| Element | Selector/Description | Notes |
|---------|---------------------|-------|
| SMS Login Tab | `button "验证码登录"` | Default tab |
| Password Login Tab | `button "密码登录"` | Switch to password mode |
| Phone Number | `textbox` + `combobox "中国 +86"` | Country code + phone input |
| Verification Code | `textbox` | 6-digit SMS code |
| Password Input | `textbox` (when password mode) | Password field |
| Login Button | `button "登录/注册"` | Submit button |
| QQ Login | QQ icon button | OAuth redirect |
| WeChat Login | WeChat icon button | QR code scan |

#### Login Status Detection Logic

```go
// CheckLoginStatus in zhihu.go
// 1. Check for "写文章", "写回答", or "提问" buttons
hasWriteBtn, _, _ := page.HasR("button", "写文章|写回答|提问")

// 2. Check for user avatar
hasAvatar, _, _ := page.Has(".Avatar, img.Avatar, [class*='Avatar']")

// 3. Check for login/register button (indicates NOT logged in)
hasLoginBtn, _, _ := page.HasR("button, a", "登录|注册")
```

---

### 2. 搜狐号登录 (Sohu)

**Login URL**: `https://mp.sohu.com/mpfe/v4/login`

**Provider File**: `geo_client2/backend/provider/sohu.go`

#### Login Methods
- 手机号登录 (Phone + SMS)
- 密码登录 (Account + Password)
- 微信登录 (WeChat QR)
- QQ 登录 (OAuth)

#### Login Status Detection Logic

```go
// CheckLoginStatus in sohu.go
// 1. Check for "发布内容" button
hasPublishBtn, _, _ := page.HasR("button", "发布内容")

// 2. Check for user avatar
hasAvatar, _, _ := page.Has(".user-avatar, .avatar, [class*='avatar']")

// 3. Check for login/register button (indicates NOT logged in)
hasLoginBtn, _, _ := page.HasR("button, a", "登录|注册")
```

---

### 3. CSDN 登录

**Login URL**: `https://passport.csdn.net/login`

**Provider File**: `geo_client2/backend/provider/csdn.go`

#### Login Methods
- 账号密码登录
- 手机验证码登录
- 微信扫码登录
- QQ 登录
- GitHub 登录

#### Login Status Detection Logic

```go
// CheckLoginStatus in csdn.go
// 1. Check for "创作" link with correct href
hasCreateLink, createEl, _ := page.HasR("a", "创作")
if hasCreateLink && createEl != nil {
    href, err := createEl.Attribute("href")
    // href should contain "mp.csdn.net/edit" or "mp_blog/creation"
}

// 2. Check for login/register prompt (indicates NOT logged in)
hasLoginBtn, _, _ := page.HasR("a, button, div, span", "登录|注册|登录/注册")

// 3. Check for avatar
hasAvatar, _, _ := page.Has("img.avatar, .user-avatar, .toolbar-avatar, [class*='avatar']")
```

---

### 4. 企鹅号登录 (Penguin/Tencent)

**Login URL**: `https://om.qq.com/userAuth/index`

**Provider File**: `geo_client2/backend/provider/qie.go`

#### Login Methods
- QQ 登录 (Primary - OAuth)
- 微信登录 (WeChat QR code scan)
- 手机/邮箱登录 (Phone/Email fallback)

#### Login Page DOM Elements

| Element | UID/Selector | Notes |
|---------|-------------|-------|
| Page Title | `heading "企鹅号" level="1"` | Platform branding |
| QQ Login Tab | `StaticText "QQ登录"` | Default login method |
| WeChat Login Tab | `StaticText "微信登录"` | QR code mode |
| Login Button | `button "立即登录"` | Triggers OAuth flow |
| Agreement Checkbox | `checkbox "我已阅读并同意"` | Must check to proceed |
| Service Agreement | `link "《腾讯内容开放平台服务协议》"` | Terms link |
| Privacy Policy | `link "《隐私政策》"` | Privacy link |
| Register Link | `link "立即注册"` | New account creation |
| Phone/Email Login | `link "手机/邮箱登录"` | Alternate login method |
| Captcha | `iframe "验证码"` - drag slider | Tencent captcha |

#### Login Flow

```
1. Navigate to https://om.qq.com/userAuth/index
2. Check "我已阅读并同意" checkbox
3. Choose login method:
   - QQ登录: Click "立即登录" → QQ OAuth popup
   - 微信登录: Click tab → Scan QR code
   - 手机/邮箱: Click link → Enter credentials
4. Complete captcha (drag slider) if prompted
5. Wait for redirect to https://om.qq.com/ (logged in state)
```

#### Login Status Detection Logic

```go
// CheckLoginStatus in qie.go
// 1. Check for publish/create buttons
hasPublishBtn, _, _ := page.HasR("button, a", "发布|发表|创作")

// 2. Check for avatar
hasAvatar, _, _ := page.Has(".avatar, [class*='avatar'], [class*='user-info']")

// 3. Check for login buttons (indicates NOT logged in)
hasLoginBtn, _, _ := page.HasR("button, a, div", "登录|注册|QQ登录|微信登录")
```

---

### 5. 百家号登录 (Baidu Baijia)

**Login URL**: `https://baijiahao.baidu.com/builder/theme/bjh/login`

**Provider File**: `geo_client2/backend/provider/baijiahao.go`

#### Login Methods
- 百度账号登录 (Baidu account)
- 手机验证码登录
- 短信快捷登录

#### Login Page Elements (when logged out)

| Element | Selector | Notes |
|---------|----------|-------|
| Login Button | `button "登录"` | Primary login button |
| Join Button | `button "立即加入百家号"` | New registration |
| Login/Register | `button "登录/注册百家号"` | Combined action |

#### Home Page Elements (when logged in)

| Element | UID/Selector | Notes |
|---------|-------------|-------|
| Logo | `image "百家号Logo"` | Platform branding |
| Avatar | `image "头像"` | User profile image |
| Publish Button | `StaticText "发布作品"` | Entry to create content |
| Content Management | `StaticText "内容管理"` | Menu item |
| Data Center | `StaticText "数据中心"` | Analytics |

#### Login Status Detection Logic

```go
// CheckLoginStatus in baijiahao.go
// 1. Check for publish/create buttons
hasPublishBtn, _, _ := page.HasR("button, a, div", "发布|写文章|创作")

// 2. Check for avatar
hasAvatar, _, _ := page.Has(".avatar, [class*='avatar'], [class*='user']")

// 3. Check for login buttons (indicates NOT logged in)
hasLoginBtn, _, _ := page.HasR("button, a, div", "登录|注册|百度账号登录")
```

---

## Platform Details

### 1. 知乎专栏 (Zhihu)

**Editor URL**: `https://zhuanlan.zhihu.com/write`

**Editor Type**: Draft.js (React-based rich text editor)

#### DOM Elements

| Element | Selector | Description |
|---------|----------|-------------|
| Title | `textarea[placeholder*="请输入标题"]` | Title input area |
| Content | `div.public-DraftEditor-content[contenteditable="true"][role="textbox"]` | Draft.js content editor |
| Publish Button | `button` containing text "发布" | Submit button |

#### Publishing Flow

```
1. Navigate to https://zhuanlan.zhihu.com/write
2. Wait for page load
3. Fill title in textarea[placeholder*="请输入标题"]
   - Use Input() or eval with dispatchEvent
4. Click content editor to focus
5. Fill content (supports HTML input)
6. Wait 20 seconds (anti-bot measure)
7. Click "发布" button
8. Wait for URL change (leaves /write path)
9. Wait for success toast
```

#### Code Pattern

```go
// Title input
titleEl, _ := page.Element(`textarea[placeholder*="请输入标题"]`)
titleEl.Input(article.Title)

// Content input (Draft.js)
contentEl, _ := page.Element(`div.public-DraftEditor-content[contenteditable="true"][role="textbox"]`)
contentEl.Click(proto.InputMouseButtonLeft, 1)
contentEl.Input(article.Content)

// Publish
btn, _ := page.ElementR("button", "发布")
btn.Click(proto.InputMouseButtonLeft, 1)
```

---

### 2. 搜狐号 (Sohu)

**Home URL**: `https://mp.sohu.com`
**Editor URL**: `https://mp.sohu.com/mpfe/v4/contentManagement/news/addarticle?contentStatus=1`

**Editor Type**: Quill.js (rich text editor)

#### DOM Elements (from live exploration)

| Element | Selector / Description | Notes |
|---------|------------------------|-------|
| "发布内容" Button | `button "发布内容"` | On home page |
| Title Input | `textbox "请输入标题（5-72字）"` | 5-72 character limit |
| Content Editor | `#editor .ql-editor` | Quill editor, use innerHTML |
| Generate Summary | `button "生成摘要"` | AI-generated summary |
| Summary Input | `textbox "请输入摘要" multiline` | 0-120 character limit |
| Cover Upload | "上传图片" area | Min 450x300px, non-GIF |
| Publish Button | `li.publish-report-btn.active.positive-button` or `li` containing "发布" | Bottom action bar |
| Save Draft | `StaticText "存草稿"` | |
| Schedule Publish | `StaticText "定时发布"` | |

#### Additional Options

| Option | Type | Values |
|--------|------|--------|
| 信息来源 (Source) | Radio | 无特别声明 / 引用声明 / 包含AI创作内容 / 包含虚构创作 |
| 话题 (Topics) | Text input | Keyword search |
| 栏目 (Column) | Selection | Link to account column |
| 可见范围 | Checkbox | "必须登录才能查看全文" (requires 100+ chars) |

#### Publishing Flow

```
1. Navigate to https://mp.sohu.com
2. Wait for page load
3. Click "发布内容" button
4. Wait for redirect to editor page (URL contains /addarticle)
5. Fill title in .publish-title input[type="text"]
6. Fill content via Quill editor:
   - Set #editor .ql-editor innerHTML directly
   - Dispatch input/change events
   - Optionally call quill.update('user')
7. Click "生成摘要" button
8. Wait 5 seconds for summary generation
9. Click "发布" button (li element)
10. Wait for URL change (redirect to article page)
```

#### Code Pattern

```go
// Navigate and click publish
page := browser.MustPage("https://mp.sohu.com")
publishBtn, _ := page.ElementR("button", "发布内容")
publishBtn.Click(proto.InputMouseButtonLeft, 1)

// Title input
titleEl, _ := page.Element(`.publish-title input[type="text"]`)
titleEl.Input(article.Title)

// Content via Quill (innerHTML method)
page.Eval(`(html) => {
    const editor = document.querySelector('#editor .ql-editor');
    if (!editor) return { success: false, error: 'editor not found' };
    
    editor.innerHTML = html;
    editor.dispatchEvent(new Event('input', { bubbles: true }));
    editor.dispatchEvent(new Event('change', { bubbles: true }));
    
    // Sync Quill internal state
    const container = document.querySelector('#editor .ql-container') || document.querySelector('#editor');
    let quill = container && (container.__quill || window.quill);
    if (!quill && typeof Quill !== 'undefined' && container) {
        quill = Quill.find(container);
    }
    if (quill && typeof quill.update === 'function') {
        quill.update('user');
    }
    
    return { success: true, length: editor.innerHTML.length };
}`, article.Content)

// Generate summary
abstractBtn, _ := page.ElementR("button", "生成摘要")
abstractBtn.Click(proto.InputMouseButtonLeft, 1)
time.Sleep(5 * time.Second)

// Publish
submitBtn, _ := page.Element(`li.publish-report-btn.active.positive-button`)
submitBtn.Click(proto.InputMouseButtonLeft, 1)
```

---

### 3. CSDN

**Home URL**: `https://www.csdn.net`
**Editor URL**: `https://mp.csdn.net/mp_blog/creation/editor`

**Editor Type**: New Rich Text Editor (iframe-based, NOT the legacy CKEditor)

> **Note**: CSDN has migrated from the legacy CKEditor (`mp.csdn.net/edit`) to a new editor. The new URL is `mp.csdn.net/mp_blog/creation/editor`.

#### DOM Elements

| Element | Selector | Notes |
|---------|----------|-------|
| "创作" Link | `a[href*="mp_blog/creation"]` or `a` containing "创作" | On home page |
| Title | `textarea[placeholder*="请输入文章标题"]` | 5-100 character limit |
| Content Editor | `main iframe` | New iframe-based editor |
| Content Body | `iframe` → `body[contenteditable="true"]` | Inside iframe |
| Add Tags Button | `button` containing "添加文章标签" | Opens tag selection popup |
| AI Summary Button | `StaticText "AI提取摘要"` or `div.btn-getdistill` | |
| Summary Input | `textbox "文章摘要" multiline` | Summary textarea |
| Publish Button | `button` containing "发布博客" | |
| Article Type | `radio "原创"`, `radio "转载"`, `radio "翻译"` | Default: 原创 |

#### Additional Options

| Option | Type | Values/Notes |
|--------|------|--------------|
| 文章类型 | Radio | 原创 (default) / 转载 / 翻译 |
| 发布形式 | Radio | 发布 / 仅发布到资源社区 |
| 可见范围 | Radio | 全部可见 / 仅VIP可见 / 仅粉丝可见 / 自己可见 |
| 个人分类 | Selection | Article category |
| 封面设置 | Button | 设置封面 |
| 精选内容 | Checkbox | 投稿精选内容 |
| 合集 | Button | 添加合集 |

#### Publishing Flow

```
1. Navigate to https://mp.csdn.net/mp_blog/creation/editor
2. Wait for page load (editor renders)
3. Fill title in textarea[placeholder*="请输入文章标题"]
4. Fill content in iframe editor:
   - Get main iframe element
   - Access iframe's body[contenteditable="true"]
   - Set innerHTML or use innerText + events
5. Click "添加文章标签" button
6. Wait for tag popup, select tags
7. Click "AI提取摘要" (optional)
8. Upload cover image (optional)
9. Click "发布博客" button
10. Wait for redirect to article page
```

#### Code Pattern

```go
// Title input
titleEl, _ := page.Timeout(5 * time.Second).Element(`textarea[placeholder*="请输入文章标题"]`)
titleEl.Input(article.Title)

// Content via new iframe editor
iframeEl, _ := page.Timeout(5 * time.Second).Element(`main iframe`)
frame, _ := iframeEl.Frame()
bodyEl, _ := frame.Timeout(5 * time.Second).Element("body")

// Set content via innerHTML
bodyEl.Eval(`(el, html) => {
    el.innerHTML = html;
    el.dispatchEvent(new Event('input', { bubbles: true }));
    el.dispatchEvent(new Event('change', { bubbles: true }));
}`, article.Content)

// Trigger word count via simulated typing
bodyEl.Click(proto.InputMouseButtonLeft, 1)
bodyEl.Type(input.Space)
bodyEl.Type(input.Backspace)

// Add tags
tagBtn, _ := page.ElementR("button", "添加文章标签")
tagBtn.Click(proto.InputMouseButtonLeft, 1)
time.Sleep(2 * time.Second)
// Select first-level tag, then second-level tags

// Publish
publishBtn, _ := page.ElementR("button", "发布博客")
publishBtn.Click(proto.InputMouseButtonLeft, 1)
```

---

### 4. 企鹅号 (Penguin/Tencent)

**Login URL**: `https://om.qq.com/userAuth/index`
**Home URL**: `https://om.qq.com`
**Editor URL**: `https://om.qq.com/main/creation/article`

**Status**: Implemented (hard-coded flow)

**Editor Type**: ContentEditable (rich text editor)

#### Login Methods
- QQ Login
- WeChat Login (scan QR code)
- Phone/Email Login

#### Key Features
- Multi-platform distribution (QQ Browser, Tencent News, Tencent Video, WeChat Look)
- Content management across platforms
- Data analytics dashboard

#### Login Page Elements

| Element | UID/Description |
|---------|-----------------|
| QQ Login Tab | `StaticText "QQ登录"` |
| WeChat Login Tab | `StaticText "微信登录"` |
| Login Button | `button "立即登录"` |
| Agreement Checkbox | `checkbox "我已阅读并同意"` |
| Register Link | `link "立即注册"` |

#### Editor DOM Elements

| Element | Selector / Description | Notes |
|---------|------------------------|-------|
| Title | `[contenteditable="true"]` with placeholder "请输入标题（5-64个字）" | 5-64 character limit |
| Content Editor | `main` area contenteditable | Rich text editor |
| Cover Type | `radio "单图" checked`, `radio "三图"` | Default: 单图 |
| Cover Upload | Image upload area | |
| Tags | `textbox` - "最多9个标签" | Up to 9 tags |
| Self-declaration | `button "添加内容自主声明"` | Content declaration |
| Publish Button | `button "发布"` | |
| Save Draft | `button "保存草稿"` | |

#### Additional Options

| Option | Type | Values/Notes |
|--------|------|--------------|
| 封面 (Cover) | Radio | 单图 (default) / 三图 |
| 标签 (Tags) | Text input | Up to 9 tags |
| 自主声明 | Button | Opens declaration popup |
| 发文身份 | Selection | Account identity selection |

#### Publishing Flow

```
1. Navigate to https://om.qq.com/main/creation/article
2. Wait for page load
3. Find and fill title in contenteditable element:
   - Locate element with placeholder "请输入标题（5-64个字）"
   - Click to focus, then input title
4. Fill content in main editor:
   - Locate contenteditable editor in main area
   - Set innerHTML or use innerText + events
5. Add tags (optional):
   - Find tag input field
   - Type tag and press Enter
6. Set cover (optional):
   - Click cover upload area
   - Select image file
7. Click "发布" button
8. Wait for success confirmation
```

#### Code Pattern

```go
// Navigate to editor
page := browser.MustPage("https://om.qq.com/main/creation/article")
page.MustWaitLoad()
page.MustWaitIdle()

// Title input - find contenteditable with placeholder
titleEl, _ := page.Eval(`() => {
    const els = document.querySelectorAll('[contenteditable="true"]');
    for (const el of els) {
        if (el.getAttribute('placeholder')?.includes('请输入标题')) {
            return el;
        }
    }
    return null;
}`)
// Or use specific selector if available
titleEl, _ := page.Element(`[contenteditable="true"][placeholder*="请输入标题"]`)
titleEl.Click(proto.InputMouseButtonLeft, 1)
titleEl.Input(article.Title)

// Content input
contentEl, _ := page.Element(`main [contenteditable="true"]:not([placeholder*="标题"])`)
contentEl.Eval(`(el, html) => {
    el.innerHTML = html;
    el.dispatchEvent(new Event('input', { bubbles: true }));
}`, article.Content)

// Publish
publishBtn, _ := page.ElementR("button", "发布")
publishBtn.Click(proto.InputMouseButtonLeft, 1)
```

---

### 5. 百家号 (Baidu Baijia)

**Login URL**: `https://baijiahao.baidu.com/builder/theme/bjh/login`
**Home URL**: `https://baijiahao.baidu.com`
**Editor URL**: `https://baijiahao.baidu.com/builder/rc/edit?type=news`

**Status**: Implemented (hard-coded flow)

**Editor Type**: iframe-based rich text editor

#### Key Features
- Multi-platform distribution within Baidu ecosystem
- AI creation tools (AI成片, AI生文, AI动漫, AI故事, 数字人, 高光剪辑)
- Monetization options (带货, 商单, 付费, 内容分销, 广告分成)

#### Login Page Elements

| Element | UID/Description |
|---------|-----------------|
| Login Button | `button "登录"` |
| Login/Register Button | `button "登录/注册百家号"` |
| Join Button | `button "立即加入百家号"` |

#### Entry Points

| Element | Selector | Notes |
|---------|----------|-------|
| Publish Button (Home) | `#home-publish-btn` | "发布作品" on home page |
| Create Article | Menu item in publish dropdown | Opens editor |

#### Editor DOM Elements

| Element | Selector / Description | Notes |
|---------|------------------------|-------|
| Title | `[contenteditable="true"]` with placeholder "请输入标题（2 - 64字）" | 2-64 character limit |
| Content Editor | `iframe` | iframe-based editor |
| Content Body | Inside iframe, contenteditable body | |
| Cover Type | `radio "三图"`, `radio "单图" checked` | Default: 单图 |
| Cover Upload | Image upload area | Min size requirements |
| Summary Input | `textbox "摘要" multiline` | Article summary |
| Auto Podcast | `checkbox "自动生成播客" checked` | Enabled by default |
| AI Declaration | `checkbox "AI创作声明"` | AI content declaration |
| Article to Dynamic | `checkbox "图文转动态"` | Convert to dynamic post |
| Publish Button | `button "发布"` | |
| Save Draft | `button "保存草稿"` | |

#### Additional Options

| Option | Type | Values/Notes |
|--------|------|--------------|
| 封面 (Cover) | Radio | 单图 (default) / 三图 |
| 摘要 (Summary) | Textarea | Article summary |
| 自动生成播客 | Checkbox | Generate podcast (default: on) |
| AI创作声明 | Checkbox | Declare AI-generated content |
| 图文转动态 | Checkbox | Convert to dynamic format |
| 添加标签 | Button | Add article tags |
| 添加话题 | Button | Add topics |

#### Publishing Flow

```
1. Navigate to https://baijiahao.baidu.com/builder/rc/edit?type=news
   - Or: Go to home page, click "#home-publish-btn", select article type
2. Wait for editor to load
3. Fill title in contenteditable element:
   - Find element with placeholder "请输入标题（2 - 64字）"
   - Click to focus, then input title
4. Fill content in iframe editor:
   - Get iframe element
   - Access contenteditable body inside iframe
   - Set innerHTML
5. Fill summary (optional):
   - Find summary textarea
   - Input summary text
6. Set cover (optional):
   - Click cover upload area
   - Select image file
7. Uncheck "自动生成播客" if not wanted
8. Click "发布" button
9. Wait for success confirmation or redirect
```

#### Code Pattern

```go
// Navigate directly to editor
page := browser.MustPage("https://baijiahao.baidu.com/builder/rc/edit?type=news")
page.MustWaitLoad()
page.MustWaitIdle()

// Title input
titleEl, _ := page.Eval(`() => {
    const els = document.querySelectorAll('[contenteditable="true"]');
    for (const el of els) {
        if (el.getAttribute('placeholder')?.includes('请输入标题')) {
            return el;
        }
    }
    return null;
}`)
titleEl.Click(proto.InputMouseButtonLeft, 1)
titleEl.Input(article.Title)

// Content via iframe
iframeEl, _ := page.Timeout(5 * time.Second).Element(`iframe`)
frame, _ := iframeEl.Frame()
bodyEl, _ := frame.Timeout(5 * time.Second).Element("body[contenteditable='true']")
bodyEl.Eval(`(el, html) => {
    el.innerHTML = html;
    el.dispatchEvent(new Event('input', { bubbles: true }));
}`, article.Content)

// Summary (optional)
summaryEl, _ := page.Element(`textarea[placeholder*="摘要"]`)
if summaryEl != nil {
    summaryEl.Input(article.Summary)
}

// Disable auto podcast if needed
podcastCheckbox, _ := page.Element(`input[type="checkbox"][checked]`) // Find auto podcast
if podcastCheckbox != nil {
    podcastCheckbox.Click(proto.InputMouseButtonLeft, 1) // Toggle off
}

// Publish
publishBtn, _ := page.ElementR("button", "发布")
publishBtn.Click(proto.InputMouseButtonLeft, 1)
```

---

## Common Patterns

### Browser Lifecycle

```go
base := provider.NewBaseProvider(platform, headless, timeout, accountID)
browser, cleanup, err := base.LaunchBrowser(false)
if err != nil {
    return err
}
defer cleanup()
defer base.Close()

page := browser.MustPage(url)
defer page.Close()
page.MustWaitLoad()
page.MustWaitIdle()
```

### Input Methods

```go
// Method 1: Direct Input (preferred)
element.Input(value)

// Method 2: Eval fallback for stubborn inputs
element.Eval(`(el, value) => { 
    el.value = value; 
    el.dispatchEvent(new Event('input', { bubbles: true })); 
}`, value)

// Method 3: For contenteditable elements
element.Eval(`(el, value) => { 
    el.innerText = value; 
    el.dispatchEvent(new Event('input', { bubbles: true })); 
}`, value)
```

### Click Methods

```go
// Standard click
element.Click(proto.InputMouseButtonLeft, 1)

// Double click
element.Click(proto.InputMouseButtonLeft, 2)
```

### Element Selection

```go
// By CSS selector
page.Element(`#myId`)
page.Element(`.myClass`)
page.Element(`button[type="submit"]`)

// By text content (regex)
page.ElementR("button", "发布")
page.ElementR("a", "创作")

// With timeout
page.Timeout(5 * time.Second).Element(selector)
```

### Wait Patterns

```go
// Page load
page.MustWaitLoad()
page.MustWaitIdle()

// Fixed delay
time.Sleep(5 * time.Second)

// Context-aware delay
select {
case <-ctx.Done():
    return ctx.Err()
case <-time.After(5 * time.Second):
}

// Wait for URL change
deadline := time.Now().Add(15 * time.Second)
for time.Now().Before(deadline) {
    info := page.MustInfo()
    if info.URL != originalURL {
        break
    }
    time.Sleep(500 * time.Millisecond)
}
```

---

## AI-Assisted Fallback

When hard-coded flows fail, the system falls back to AI-assisted publishing using a Dify workflow.

### Observation Capture

```go
type Observation struct {
    Goal       string `json:"goal"`
    URL        string `json:"url"`
    Title      string `json:"title"`
    DOM        string `json:"dom"`        // [TEXT] + [HTML]
    Clickables string `json:"clickables"` // JSON array of {text, selector, tag}
    Screenshot string `json:"screenshot"` // Base64 PNG
}
```

### AI Decision Actions

| Action | Parameters | Description |
|--------|------------|-------------|
| `input` | selector, value | Type text into element |
| `click` | selector | Click element |
| `wait` | ms | Wait for milliseconds |
| `done` | - | Publishing complete |
| `request_manual` | reason | Request human intervention |

### Example Action Sequence (Cached)

The cached sequence is stored per platform and reused with template substitution.

```json
[
  {"action": "input", "selector": "textarea[placeholder*='标题']", "value": "{{title}}"},
  {"action": "click", "selector": "div.editor[contenteditable='true']"},
  {"action": "input", "selector": "div.editor[contenteditable='true']", "value": "{{content}}"},
  {"action": "wait", "ms": 1200},
  {"action": "click", "selector": "button:has-text('发布')"},
  {"action": "done"}
]
```

### Template Substitution

Template placeholders are replaced at runtime:

```
"{{title}}"   → article.Title
"{{content}}" → article.Content
```

If a cached sequence fails (selector missing, timeout, or navigation changes), the system falls back to:
1. Re-run AI planning with fresh DOM + screenshot
2. If still failing, emit `request_manual` for human intervention

### Caching

Successfully executed action sequences are cached per platform for future use, with template substitution for `{{title}}` and `{{content}}`.

---

## Anti-Bot Measures

### CAPTCHAs Encountered

| Platform | CAPTCHA Type | Solution |
|----------|--------------|----------|
| 搜狐号 | Tencent drag slider (腾讯验证码) | Manual intervention |
| 企鹅号 | Tencent drag slider | Manual intervention |
| CSDN | - | Usually none for logged-in users |
| 知乎 | 可能出现短信或滑块验证 | Manual intervention |

### Rate Limits

| Platform | Limit |
|----------|-------|
| 搜狐号 | 5 articles/day (shown on editor page) |

### Best Practices

1. **Use persistent browser profiles** - Store in `~/.geo_client2/browser_data/{platform}/{account_id}/`
2. **Add delays between actions** - 1-5 seconds between major operations
3. **Simulate human typing** - Use `Input()` instead of direct value assignment when possible
4. **Wait for network idle** - Use `page.MustWaitIdle()` after navigation
5. **Handle timeouts gracefully** - Fall back to AI-assisted or manual intervention

### StartLogin() Manual Flow

When `CheckLoginStatus()` returns false, the app triggers manual login with a headed browser.

```
1. LaunchBrowser(headless=false) with account-scoped user data dir
2. Navigate to platform LoginURL
3. User completes login (SMS/QR/password)
4. Persist cookies/session into user_data_dir
5. Next CheckLoginStatus() should return true
```

**Notes**:
- Manual steps are required for CAPTCHA and QR scan flows
- `user_data_dir` persistence is the primary mechanism for keeping login sessions
- If login expires, the flow repeats

---

## File References

### Publisher Files (Publishing Logic)

| File | Description |
|------|-------------|
| `geo_client2/backend/publisher/zhihu.go` | 知乎 publisher - Draft.js editor automation |
| `geo_client2/backend/publisher/sohu.go` | 搜狐号 publisher - Quill.js editor automation |
| `geo_client2/backend/publisher/csdn.go` | CSDN publisher - iframe-based editor automation |
| `geo_client2/backend/publisher/qie.go` | 企鹅号 publisher - contenteditable automation |
| `geo_client2/backend/publisher/baijiahao.go` | 百家号 publisher - iframe-based editor automation |
| `geo_client2/backend/publisher/publishers.go` | Base publisher, AI assist fallback, 小红书 |

### Provider Files (Login Detection)

| File | Description |
|------|-------------|
| `geo_client2/backend/provider/zhihu.go` | 知乎 login status detection |
| `geo_client2/backend/provider/sohu.go` | 搜狐号 login status detection |
| `geo_client2/backend/provider/csdn.go` | CSDN login status detection |
| `geo_client2/backend/provider/qie.go` | 企鹅号 login status detection |
| `geo_client2/backend/provider/baijiahao.go` | 百家号 login status detection |
| `geo_client2/backend/provider/base.go` | BaseProvider - shared browser lifecycle |

### Configuration Files

| File | Description |
|------|-------------|
| `geo_client2/backend/config/platforms.go` | Platform URLs (login, home, editor) |
| `RPA发布assistant.yml` | Dify workflow for AI-assisted publishing |

### Browser Data Storage

```
~/.geo_client2/
├── browser_data/
│   ├── zhihu/{account_id}/      # 知乎 persistent profile
│   ├── sohu/{account_id}/       # 搜狐号 persistent profile
│   ├── csdn/{account_id}/       # CSDN persistent profile
│   ├── qie/{account_id}/        # 企鹅号 persistent profile
│   └── baijiahao/{account_id}/  # 百家号 persistent profile
└── cache.db                     # SQLite database
```
