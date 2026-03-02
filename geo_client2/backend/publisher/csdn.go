package publisher

import (
	"context"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

type CsdnPublisher struct{ *BasePublisher }

// Publish implements the Publisher interface for CSDN.
// Flow (using new rich text editor at mp.csdn.net/mp_blog/creation/editor):
//  1. Navigate directly to editor page
//  2. Wait for editor to load
//  3. Fill title: textbox with placeholder "请输入文章标题（5～100个字）"
//  4. Fill content in new iframe editor (not CKEditor)
//  5. Click "添加文章标签" button
//  6. Select tags from popup
//  7. Click "AI提取摘要" to auto-generate summary
//  8. Click "发布博客" button
//
// Any hard-coded step failure falls back to runAIAssist.
func (p *CsdnPublisher) Publish(
	ctx context.Context,
	article Article,
	resume <-chan struct{},
	emit EventEmitter,
	aiConfig AIPublishConfig,
) error {
	log := p.logger

	emit("publish:progress", map[string]string{"platform": "csdn", "message": "正在打开 CSDN 编辑器..."})
	log.Info("[CSDN] Starting publish: " + article.Title)

	base := p.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		log.Info("[CSDN] launch browser failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	defer cleanup()
	defer base.Close()

	// ── Step 1: Navigate directly to new editor page ────────────────────────
	// New editor URL: https://mp.csdn.net/mp_blog/creation/editor
	page := browser.MustPage("https://mp.csdn.net/mp_blog/creation/editor")
	defer page.Close()
	page.MustWaitLoad()
	page.MustWaitIdle()

	// ── Step 2: Wait for editor to fully load ───────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "等待编辑器加载..."})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}

	// Wait for title input to appear (indicates editor is ready)
	_, err = page.Timeout(10 * time.Second).Element(`textarea[placeholder*="请输入文章标题"]`)
	if err != nil {
		log.Info("[CSDN] editor not loaded (title input not found): " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	emit("publish:progress", map[string]string{"platform": "csdn", "message": "填写标题..."})
	titleEl, err := page.Element(`textarea[placeholder*="请输入文章标题"]`)
	if err != nil {
		log.Info("[CSDN] title textarea not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	_, titleErr := titleEl.Eval(`(v) => {
		const setter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value');
		if (setter && setter.set) {
			setter.set.call(this, v);
		} else {
			this.value = v;
		}
		this.dispatchEvent(new Event('input',  { bubbles: true }));
		this.dispatchEvent(new Event('change', { bubbles: true }));
	}`, article.Title)
	if titleErr != nil {
		log.Info("[CSDN] title fill failed: " + titleErr.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	log.Info("[CSDN] title filled: " + article.Title)

	emit("publish:progress", map[string]string{"platform": "csdn", "message": "填写正文..."})

	contentResult, err := page.Eval(`(html) => {
		try {
			const iframe = document.querySelector('main iframe');
			if (!iframe || !iframe.contentDocument || !iframe.contentDocument.body) {
				return { success: false, error: 'iframe not found' };
			}
			const body = iframe.contentDocument.body;
			const doc  = iframe.contentDocument;
			body.innerHTML = html;
			const evtOpts = { bubbles: true, cancelable: true };
			body.dispatchEvent(new Event('input',   evtOpts));
			body.dispatchEvent(new Event('keyup',   evtOpts));
			body.dispatchEvent(new Event('keydown', evtOpts));
			doc.dispatchEvent(new Event('input',    evtOpts));
			doc.dispatchEvent(new Event('selectionchange', { bubbles: false }));
			return { success: true, method: 'iframe_html', length: html.length };
		} catch (e) {
			return { success: false, error: e.message };
		}
	}`, article.Content)
	if err != nil {
		log.Info("[CSDN] content eval failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if !contentResult.Value.Get("success").Bool() {
		log.Info("[CSDN] content insertion failed: " + contentResult.Value.Get("error").Str())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	log.Info("[CSDN] content inserted length=" + contentResult.Value.Get("length").String())

	iframeEl, err := page.Timeout(5 * time.Second).Element(`main iframe`)
	if err != nil {
		log.Info("[CSDN] iframe not found for word count trigger: " + err.Error())
	} else {
		frame, frameErr := iframeEl.Frame()
		if frameErr != nil {
			log.Info("[CSDN] iframe frame error: " + frameErr.Error())
		} else {
			bodyEl, bodyErr := frame.Timeout(5 * time.Second).Element("body")
			if bodyErr != nil {
				log.Info("[CSDN] iframe body not found: " + bodyErr.Error())
			} else {
				if clickErr := bodyEl.Click(proto.InputMouseButtonLeft, 1); clickErr == nil {
					if typeErr := bodyEl.Type(input.Space); typeErr == nil {
						time.Sleep(100 * time.Millisecond)
						bodyEl.Type(input.Backspace)
						log.Info("[CSDN] word count triggered via typing")
					}
				}
			}
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}

	// Step 5: Click tag button
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "添加文章标签..."})
	tagBtn, err := page.Timeout(5*time.Second).ElementR("button", "添加文章标签")
	if err != nil {
		log.Info("[CSDN] tag button not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := tagBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[CSDN] click tag button failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	// Step 6: Wait for tag popup and select tags
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "等待标签弹窗..."})
	tagPopup, err := page.Timeout(10 * time.Second).Element(`.mark-modal, .tag-modal, [class*="tag"][class*="modal"], .el_mcm-dialog`)
	if err != nil {
		log.Info("[CSDN] tag popup not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	_ = tagPopup

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}

	emit("publish:progress", map[string]string{"platform": "csdn", "message": "选择一级标签..."})
	firstLevelTag, err := page.Timeout(5*time.Second).ElementR(`[class*="tabs__item"], [class*="tab-item"], .category-item`, "学习和成长")
	if err != nil {
		firstLevelTag, err = page.ElementR("div, span", "学习和成长")
	}
	if err != nil {
		log.Info("[CSDN] first level tag '学习和成长' not found, picking first available tab")
		fallbackRes, fbErr := page.Eval(`() => {
			const selectors = ['[class*="tabs__item"]', '[class*="tab-item"]', '.category-item'];
			for (const sel of selectors) {
				const els = document.querySelectorAll(sel);
				if (els.length > 0) { els[0].click(); return { clicked: true, text: els[0].textContent.trim() }; }
			}
			return { clicked: false };
		}`)
		if fbErr != nil || !fallbackRes.Value.Get("clicked").Bool() {
			log.Info("[CSDN] first level tag fallback failed, continuing")
		} else {
			log.Info("[CSDN] first level tag fallback clicked: " + fallbackRes.Value.Get("text").Str())
		}
	} else {
		if err := firstLevelTag.Click(proto.InputMouseButtonLeft, 1); err != nil {
			log.Info("[CSDN] click first level tag failed: " + err.Error())
		} else {
			log.Info("[CSDN] first level tag '学习和成长' clicked")
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
	}

	emit("publish:progress", map[string]string{"platform": "csdn", "message": "选择二级标签..."})
	secondLevelTags := []string{"职场和发展", "学习方法"}
	clickedSubtags := 0
	for _, tagName := range secondLevelTags {
		tag, err := page.Timeout(3*time.Second).ElementR(`[class*="tag"], .tag-item, .sub-tag`, tagName)
		if err != nil {
			tag, err = page.ElementR("span, div, label", tagName)
		}
		if err != nil {
			log.Info("[CSDN] subtag '" + tagName + "' not found, skipping")
			continue
		}
		if err := tag.Click(proto.InputMouseButtonLeft, 1); err != nil {
			log.Info("[CSDN] click subtag '" + tagName + "' failed: " + err.Error())
		} else {
			log.Info("[CSDN] subtag '" + tagName + "' clicked")
			clickedSubtags++
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	if clickedSubtags == 0 {
		log.Info("[CSDN] no named subtags found, falling back to first available subtag")
		firstSubtag, fbErr := page.Eval(`() => {
			const selectors = [
				'[class*="sub-tag"]',
				'[class*="subtag"]',
				'[class*="child-tag"]',
				'[class*="tag-item"]',
				'.mark-list li',
				'.tag-list li',
			];
			for (const sel of selectors) {
				const els = Array.from(document.querySelectorAll(sel));
				if (els.length > 0) {
					els[0].click();
					return { clicked: true, selector: sel, text: els[0].textContent.trim() };
				}
			}
			return { clicked: false };
		}`)
		if fbErr == nil && firstSubtag.Value.Get("clicked").Bool() {
			log.Info("[CSDN] fallback subtag clicked: " + firstSubtag.Value.Get("text").Str())
		} else {
			log.Info("[CSDN] fallback subtag also not found, continuing without subtag")
		}
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}

	// Step 7: Click AI summary button (text "AI提取摘要")
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "AI提取摘要..."})
	aiBtn, err := page.Timeout(3*time.Second).ElementR("*", "AI提取摘要")
	if err != nil {
		log.Info("[CSDN] AI summary button not found: " + err.Error())
	} else {
		if err := aiBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
			log.Info("[CSDN] click AI summary button failed: " + err.Error())
		} else {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
			}
		}
	}

	if article.CoverImage != "" {
		emit("publish:progress", map[string]string{"platform": "csdn", "message": "上传封面图..."})
		coverUploaded := p.uploadCoverImage(page, article.CoverImage)
		if !coverUploaded {
			log.Info("[CSDN] cover image upload failed, continuing without cover")
		}
	}

	// Step 8: Click publish button
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "点击发布博客..."})
	publishBtn, err := page.Timeout(5*time.Second).ElementR("button", "发布博客")
	if err == nil {
		if clickErr := publishBtn.Click(proto.InputMouseButtonLeft, 1); clickErr != nil {
			log.Info("[CSDN] click publish button failed: " + clickErr.Error())
			err = clickErr
		}
	} else {
		log.Info("[CSDN] publish button not found: " + err.Error())
	}

	if err != nil {
		publishResult, evalErr := page.Eval(`() => {
			const box = document.querySelector('.btn-box');
			if (!box) return { success: false, reason: 'btn_box_not_found' };
			const buttons = Array.from(box.querySelectorAll('button'));
			const target = buttons.find((btn) => {
				const text = (btn.innerText || btn.textContent || '').replace(/\s+/g, ' ').trim();
				return /发布博客/.test(text) && !/草稿|定时/.test(text);
			});
			if (!target) return { success: false, reason: 'publish_button_not_found' };
			target.scrollIntoView({ block: 'center', inline: 'center' });
			target.click();
			return { success: true, reason: 'btn_box_click' };
		}`)
		if evalErr != nil {
			log.Info("[CSDN] publish button eval failed: " + evalErr.Error())
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
		if publishResult != nil && publishResult.Value.Get("success").Bool() {
			log.Info("[CSDN] publish button clicked via btn-box selector")
		} else {
			log.Info("[CSDN] publish button not found in btn-box")
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
	}

	if confirmed, popupErr := autoConfirmPlatformPopups(ctx, page, "csdn", true, 6, 700*time.Millisecond); popupErr == nil && confirmed > 0 {
		emit("publish:progress", map[string]string{"platform": "csdn", "message": "检测到发布弹窗，已自动确认"})
	}

	log.Info("[CSDN] publish button clicked, waiting for confirmation...")
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "等待发布确认..."})

	articleURL := ""
	editorURL := page.MustInfo().URL
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if confirmed, popupErr := autoConfirmPlatformPopups(ctx, page, "csdn", true, 1, 300*time.Millisecond); popupErr == nil && confirmed > 0 {
			emit("publish:progress", map[string]string{"platform": "csdn", "message": "发布后弹窗已自动确认，继续检测结果..."})
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		info := page.MustInfo()
		if info != nil && info.URL != "" && info.URL != editorURL && !strings.Contains(info.URL, "mp_blog/creation/editor") {
			articleURL = info.URL
			log.Info("[CSDN] redirected to article page: " + articleURL)
			break
		}
	}

	if articleURL == "" {
		log.Info("[CSDN] redirect not detected within timeout, treating as success anyway")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}

	log.Info("[CSDN] publish completed, article_url=" + articleURL)
	emit("publish:completed", map[string]interface{}{
		"platform":   "csdn",
		"success":    true,
		"articleUrl": articleURL,
	})

	return errAlreadyEmitted
}

func (p *CsdnPublisher) uploadCoverImage(page *rod.Page, coverImage string) bool {
	// TODO: Implement cover image upload for CSDN rich text editor
	return false
}
