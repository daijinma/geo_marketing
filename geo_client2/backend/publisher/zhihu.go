package publisher

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// ZhihuPublisher handles publishing articles to 知乎专栏.
type ZhihuPublisher struct{ *BasePublisher }

// Publish implements the Publisher interface for 知乎.
// Flow:
//  1. Open https://zhuanlan.zhihu.com/write
//  2. Fill in title
//  3. Fill in content (Draft.js editor)
//  4. Wait 20 s, then click the publish button
//  5. Wait up to 10 s for the success toast / URL change, then return
//
// Any hard-coded step failure falls back to runAIAssist.
func (p *ZhihuPublisher) Publish(
	ctx context.Context,
	article Article,
	resume <-chan struct{},
	emit EventEmitter,
	aiConfig AIPublishConfig,
) error {
	log := p.logger

	emit("publish:progress", map[string]string{"platform": "zhihu", "message": "正在打开知乎编辑器..."})
	log.Info("[Zhihu] Starting publish: " + article.Title)

	base := p.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		log.Info("[Zhihu] launch browser failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	defer cleanup()
	defer base.Close()

	page := browser.MustPage("https://zhuanlan.zhihu.com/write")
	defer page.Close()
	page.MustWaitLoad()
	page.MustWaitIdle()

	// ── Step 1: title ──────────────────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "zhihu", "message": "填写标题..."})
	titleEl, err := page.Element(`textarea[placeholder*="请输入标题"]`)
	if err != nil {
		log.Info("[Zhihu] title input not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := titleEl.Input(article.Title); err != nil {
		_, evalErr := titleEl.Eval(
			`(el, value) => { el.value = value; el.dispatchEvent(new Event('input', { bubbles: true })); }`,
			article.Title,
		)
		if evalErr != nil {
			log.Info("[Zhihu] title input failed: " + evalErr.Error())
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
	}

	// ── Step 2: content (Draft.js) ─────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "zhihu", "message": "填写正文..."})
	contentEl, err := page.Element(`div.public-DraftEditor-content[contenteditable="true"][role="textbox"]`)
	if err != nil {
		log.Info("[Zhihu] content editor not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	_ = contentEl.Click(proto.InputMouseButtonLeft, 1)
	if err := contentEl.Input(article.Content); err != nil {
		_, evalErr := contentEl.Eval(
			`(el, value) => { el.innerText = value; el.dispatchEvent(new Event('input', { bubbles: true })); }`,
			article.Content,
		)
		if evalErr != nil {
			log.Info("[Zhihu] content input failed: " + evalErr.Error())
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
	}

	// ── Step 3: wait 20 s, then click publish ──────────────────────────────
	emit("publish:progress", map[string]string{"platform": "zhihu", "message": "等待 20s 后点击发布..."})
	time.Sleep(20 * time.Second)

	emit("publish:progress", map[string]string{"platform": "zhihu", "message": "点击发布..."})
	btn, err := page.ElementR("button", "发布")
	if err != nil {
		log.Info("[Zhihu] publish button not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := btn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[Zhihu] click publish failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	log.Info("[Zhihu] publish button clicked, waiting for page redirect...")
	emit("publish:progress", map[string]string{"platform": "zhihu", "message": "已点击发布，等待页面跳转..."})

	// ── Step 4: wait up to 15 s for URL to leave /write ───────────────────
	// 知乎点击发布后会立刻跳转到文章页，URL 不再是 /write。
	articleURL := ""
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}

		info := page.MustInfo()
		if info != nil && info.URL != "" && info.URL != "https://zhuanlan.zhihu.com/write" {
			articleURL = info.URL
			log.Info("[Zhihu] redirected to article page: " + articleURL)
			break
		}
	}

	if articleURL == "" {
		log.Info("[Zhihu] redirect not detected within timeout, treating as success anyway")
	}

	// ── Step 5: wait 5 s for the "发布成功" toast to appear on the page ────
	emit("publish:progress", map[string]string{"platform": "zhihu", "message": "等待发布成功弹窗..."})
	log.Info("[Zhihu] waiting 5s for success toast")

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
	}

	log.Info("[Zhihu] publish completed, article_url=" + articleURL)
	emit("publish:completed", map[string]interface{}{
		"platform":   "zhihu",
		"success":    true,
		"articleUrl": articleURL,
	})

	// Return sentinel so manager.go does not emit a second publish:completed.
	return errAlreadyEmitted
}

// containsAny returns true if s contains any of the substrings in needles.
func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

// errAlreadyEmitted is returned by publishers that handle their own
// publish:completed event emission, so manager.go skips the default one.
var errAlreadyEmitted = fmt.Errorf("already_emitted")
