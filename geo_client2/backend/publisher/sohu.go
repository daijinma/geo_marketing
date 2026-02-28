package publisher

import (
	"context"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// SohuPublisher handles publishing articles to 搜狐号.
type SohuPublisher struct{ *BasePublisher }

// Publish implements the Publisher interface for 搜狐号.
// Flow:
//  1. Open https://mp.sohu.com (home page)
//  2. Click "发布内容" button
//  3. Wait for redirect to publish page
//  4. Fill in title
//  5. Fill in content (Quill editor)
//  6. Click "生成摘要" button
//  7. Wait 5 s
//  8. Click "发布" <li> button
//  9. Wait up to 15 s for URL to change, then wait 5 s and emit completed
//
// Any hard-coded step failure falls back to runAIAssist.
func (p *SohuPublisher) Publish(
	ctx context.Context,
	article Article,
	resume <-chan struct{},
	emit EventEmitter,
	aiConfig AIPublishConfig,
) error {
	log := p.logger

	emit("publish:progress", map[string]string{"platform": "sohu", "message": "正在打开搜狐号..."})
	log.Info("[Sohu] Starting publish: " + article.Title)

	base := p.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		log.Info("[Sohu] launch browser failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	defer cleanup()
	defer base.Close()

	page := browser.MustPage("https://mp.sohu.com")
	defer page.Close()
	page.MustWaitLoad()
	page.MustWaitIdle()

	// ── Step 1: click "发布内容" button ─────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "点击发布内容..."})
	publishBtn, err := page.ElementR("button", "发布内容")
	if err != nil {
		log.Info("[Sohu] publish-content button not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := publishBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[Sohu] click publish-content button failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	// ── Step 2: wait for redirect to publish page ───────────────────────────
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "等待跳转至发布页..."})
	publishPageURL := ""
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		info := page.MustInfo()
		if info != nil && info.URL != "" && info.URL != "https://mp.sohu.com" && info.URL != "https://mp.sohu.com/" {
			publishPageURL = info.URL
			log.Info("[Sohu] redirected to publish page: " + publishPageURL)
			break
		}
	}
	if publishPageURL == "" {
		log.Info("[Sohu] redirect to publish page not detected within timeout")
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	page.MustWaitLoad()
	page.MustWaitIdle()

	// ── Step 3: fill title ──────────────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "填写标题..."})
	titleEl, err := page.Element(`.publish-title input[type="text"]`)
	if err != nil {
		log.Info("[Sohu] title input not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := titleEl.Input(article.Title); err != nil {
		_, evalErr := titleEl.Eval(
			`(el, value) => { el.value = value; el.dispatchEvent(new Event('input', { bubbles: true })); }`,
			article.Title,
		)
		if evalErr != nil {
			log.Info("[Sohu] title input failed: " + evalErr.Error())
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
	}

	// ── Step 4: fill content (direct innerHTML to preserve full HTML with styles) ─
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "填写正文..."})
	// Bypass Quill API entirely — set innerHTML directly to preserve all inline styles.
	// Quill's dangerouslyPasteHTML strips styles; innerHTML keeps them intact.
	contentResult, err := page.Eval(`(html) => {
		const editor = document.querySelector('#editor .ql-editor');
		if (!editor) {
			return { success: false, error: 'editor not found' };
		}
		
		editor.innerHTML = html;
		
		// Trigger events so Quill syncs its internal Delta state
		editor.dispatchEvent(new Event('input', { bubbles: true }));
		editor.dispatchEvent(new Event('change', { bubbles: true }));
		
		// Also try to update Quill's internal state if accessible
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
	if err != nil {
		log.Info("[Sohu] content eval failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if contentResult.Value.Get("success").Bool() {
		log.Info("[Sohu] content inserted via innerHTML, length=" + contentResult.Value.Get("length").String())
	} else {
		log.Info("[Sohu] content insertion failed: " + contentResult.Value.Get("error").Str())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	// ── Step 5: click "生成摘要" ─────────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "点击生成摘要..."})
	abstractBtn, err := page.ElementR("button", "生成摘要")
	if err != nil {
		log.Info("[Sohu] abstract button not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := abstractBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[Sohu] click abstract button failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	// ── Step 6: wait 5 s ────────────────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "等待 5s..."})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
	}

	// ── Step 7: click "发布" <li> ────────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "点击发布..."})
	submitBtn, err := page.Element(`li.publish-report-btn.active.positive-button`)
	if err != nil {
		// fallback: try by text content
		submitBtn, err = page.ElementR("li", "发布")
		if err != nil {
			log.Info("[Sohu] submit button not found: " + err.Error())
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
	}
	if err := submitBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[Sohu] click submit button failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	if confirmed, popupErr := autoConfirmPlatformPopups(ctx, page, "sohu", true, 6, 700*time.Millisecond); popupErr == nil && confirmed > 0 {
		emit("publish:progress", map[string]string{"platform": "sohu", "message": "检测到发布弹窗，已自动确认"})
	}

	log.Info("[Sohu] publish button clicked, waiting for page redirect...")
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "已点击发布，等待页面跳转..."})

	// ── Step 8: wait up to 15 s for URL to leave the publish page ───────────
	articleURL := ""
	deadline2 := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline2) {
		if confirmed, popupErr := autoConfirmPlatformPopups(ctx, page, "sohu", true, 1, 300*time.Millisecond); popupErr == nil && confirmed > 0 {
			emit("publish:progress", map[string]string{"platform": "sohu", "message": "发布后弹窗已自动确认，继续检测结果..."})
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		info := page.MustInfo()
		if info != nil && info.URL != "" && info.URL != publishPageURL {
			articleURL = info.URL
			log.Info("[Sohu] redirected to article page: " + articleURL)
			break
		}
	}
	if articleURL == "" {
		log.Info("[Sohu] redirect not detected within timeout, treating as success anyway")
	}

	// ── Step 9: wait 5 s then emit completed ────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "sohu", "message": "等待发布确认..."})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
	}

	log.Info("[Sohu] publish completed, article_url=" + articleURL)
	emit("publish:completed", map[string]interface{}{
		"platform":   "sohu",
		"success":    true,
		"articleUrl": articleURL,
	})

	return errAlreadyEmitted
}
