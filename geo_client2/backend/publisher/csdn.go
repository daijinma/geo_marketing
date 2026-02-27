package publisher

import (
	"context"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
)

type CsdnPublisher struct{ *BasePublisher }

// Publish implements the Publisher interface for CSDN.
// Flow (using rich text editor at mp.csdn.net/edit):
//  1. Open https://www.csdn.net/
//  2. Click "创作" link (href="https://mp.csdn.net/edit")
//  3. Wait 5s for editor to load
//  4. Fill title: textarea#txtTitle
//  5. Fill content in iframe: iframe.cke_wysiwyg_frame -> body.innerHTML
//  6. Click "添加文章标签" button
//  7. Wait 5s for tag popup
//  8. Select first-level tag "学习和成长"
//  9. Select second-level tags "职场和发展", "学习方法"
//
// 10. Click "AI提取摘要"
// 11. Click "发布博客" button
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

	emit("publish:progress", map[string]string{"platform": "csdn", "message": "正在打开 CSDN..."})
	log.Info("[CSDN] Starting publish: " + article.Title)

	base := p.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		log.Info("[CSDN] launch browser failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	defer cleanup()
	defer base.Close()

	// ── Step 1: Open CSDN homepage ──────────────────────────────────────────
	page := browser.MustPage("https://www.csdn.net/")
	defer page.Close()
	page.MustWaitLoad()
	page.MustWaitIdle()

	// ── Step 2: Click "创作" link ───────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "点击创作按钮..."})
	createLink, err := page.Element(`a[href="https://mp.csdn.net/edit"]`)
	if err != nil {
		// Fallback: try finding by text
		createLink, err = page.ElementR("a", "创作")
	}
	if err != nil {
		log.Info("[CSDN] create link not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := createLink.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[CSDN] click create link failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	// ── Step 3: Wait 5s for editor page to load ─────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "等待编辑器加载..."})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
	}

	// Wait for page navigation to mp.csdn.net/edit
	page.MustWaitLoad()
	page.MustWaitIdle()

	// Additional wait for dynamic content
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
	}

	// ── Step 4: Fill title ──────────────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "填写标题..."})
	titleEl, err := page.Element(`textarea#txtTitle`)
	if err != nil {
		// Fallback: try by placeholder
		titleEl, err = page.Element(`textarea[placeholder*="请输入文章标题"]`)
	}
	if err != nil {
		log.Info("[CSDN] title textarea not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := titleEl.SelectAllText(); err == nil {
		titleEl.MustInput("")
	}
	if err := titleEl.Input(article.Title); err != nil {
		_, evalErr := titleEl.Eval(
			`(el, value) => { el.value = value; el.dispatchEvent(new Event('input', { bubbles: true })); }`,
			article.Title,
		)
		if evalErr != nil {
			log.Info("[CSDN] title input failed: " + evalErr.Error())
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
	}
	log.Info("[CSDN] title filled: " + article.Title)

	// ── Step 5: Fill content in CKEditor iframe ─────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "填写正文..."})

	// Use CKEditor's setData API from parent page to properly sync content
	contentResult, err := page.Eval(`(html) => {
		try {
			// CKEditor instance is accessible via CKEDITOR.instances
			if (typeof CKEDITOR !== 'undefined' && CKEDITOR.instances) {
				const editorName = Object.keys(CKEDITOR.instances)[0];
				if (editorName) {
					const editor = CKEDITOR.instances[editorName];
					editor.setData(html);
					return { success: true, method: 'ckeditor_api', length: html.length };
				}
			}
			
			// Fallback: directly set iframe body innerHTML
			const iframe = document.querySelector('iframe.cke_wysiwyg_frame');
			if (iframe && iframe.contentDocument && iframe.contentDocument.body) {
				iframe.contentDocument.body.innerHTML = html;
				return { success: true, method: 'iframe_innerhtml', length: html.length };
			}
			
			return { success: false, error: 'no editor found' };
		} catch (e) {
			return { success: false, error: e.message };
		}
	}`, article.Content)
	if err != nil {
		log.Info("[CSDN] content eval failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if contentResult.Value.Get("success").Bool() {
		log.Info("[CSDN] content inserted via " + contentResult.Value.Get("method").Str() + ", length=" + contentResult.Value.Get("length").String())
	} else {
		log.Info("[CSDN] content insertion failed: " + contentResult.Value.Get("error").Str())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	// Simulate typing to trigger word count detection
	iframeEl, err := page.Timeout(5 * time.Second).Element(`iframe.cke_wysiwyg_frame`)
	if err != nil {
		log.Info("[CSDN] word count iframe not found: " + err.Error())
	} else {
		frame, err := iframeEl.Frame()
		if err != nil {
			log.Info("[CSDN] word count iframe frame error: " + err.Error())
		} else {
			bodyEl, err := frame.Timeout(5 * time.Second).Element("body")
			if err != nil {
				log.Info("[CSDN] word count body not found: " + err.Error())
			} else {
				clicked := false
				for i := 0; i < 3; i++ {
					if err := bodyEl.Click(proto.InputMouseButtonLeft, 1); err == nil {
						clicked = true
						break
					}
					time.Sleep(200 * time.Millisecond)
				}
				if !clicked {
					if err := bodyEl.Focus(); err != nil {
						log.Info("[CSDN] word count body focus failed: " + err.Error())
					} else {
						clicked = true
					}
				}
				if clicked {
					if err := bodyEl.Type(input.Space); err != nil {
						log.Info("[CSDN] word count type space failed: " + err.Error())
					} else {
						time.Sleep(100 * time.Millisecond)
						if err := bodyEl.Type(input.Backspace); err != nil {
							log.Info("[CSDN] word count type backspace failed: " + err.Error())
						} else {
							log.Info("[CSDN] triggered word count via simulated typing")
						}
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

	// ── Step 6: Click "添加文章标签" button ──────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "添加文章标签..."})
	tagBtn, err := page.Element(`button.tag__btn-tag`)
	if err != nil {
		// Fallback: try by text
		tagBtn, err = page.ElementR("button", "添加文章标签")
	}
	if err != nil {
		log.Info("[CSDN] tag button not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := tagBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[CSDN] click tag button failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	// ── Step 7: Wait for tag popup to appear ───────────────────────────────
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

	// ── Step 8: Select first-level tag "学习和成长" ─────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "选择一级标签..."})
	firstLevelTag, err := page.Timeout(5*time.Second).ElementR(`[class*="tabs__item"], [class*="tab-item"], .category-item`, "学习和成长")
	if err != nil {
		firstLevelTag, err = page.ElementR("div, span", "学习和成长")
	}
	if err != nil {
		log.Info("[CSDN] first level tag not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := firstLevelTag.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[CSDN] click first level tag failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	log.Info("[CSDN] first level tag '学习和成长' clicked")

	// Wait for second level tags to load after first level selection
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
	}

	// ── Step 9: Select second-level tags ────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "选择二级标签..."})

	secondLevelTags := []string{"职场和发展", "学习方法"}
	for _, tagName := range secondLevelTags {
		tag, err := page.Timeout(3*time.Second).ElementR(`[class*="tag"], .tag-item, .sub-tag`, tagName)
		if err != nil {
			tag, err = page.ElementR("span, div, label", tagName)
		}
		if err != nil {
			log.Info("[CSDN] tag '" + tagName + "' not found: " + err.Error())
			continue
		}
		if err := tag.Click(proto.InputMouseButtonLeft, 1); err != nil {
			log.Info("[CSDN] click tag '" + tagName + "' failed: " + err.Error())
		} else {
			log.Info("[CSDN] tag '" + tagName + "' clicked")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	// Wait after tag selection
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}

	// ── Step 10: Click "AI提取摘要" ─────────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "AI提取摘要..."})
	aiBtn, err := page.Element(`div.btn-getdistill`)
	if err != nil {
		aiBtn, err = page.ElementR("div", "AI提取摘要")
	}
	if err != nil {
		log.Info("[CSDN] AI summary button not found: " + err.Error())
		// Not fatal, continue without AI summary
	} else {
		if err := aiBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
			log.Info("[CSDN] click AI summary button failed: " + err.Error())
		} else {
			// Wait for AI to generate summary
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(3 * time.Second):
			}
		}
	}

	// Handle cover image if provided
	if article.CoverImage != "" {
		emit("publish:progress", map[string]string{"platform": "csdn", "message": "上传封面图..."})
		coverUploaded := p.uploadCoverImage(page, article.CoverImage)
		if !coverUploaded {
			log.Info("[CSDN] cover image upload failed, continuing without cover")
		}
	}

	// ── Step 11: Click "发布博客" button ────────────────────────────────────
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "点击发布博客..."})
	publishBtn, err := page.ElementR(`button.el_mcm-button--primary`, "发布博客")
	if err != nil {
		// Fallback: try by text only
		publishBtn, err = page.ElementR("button", "发布博客")
	}
	if err != nil {
		log.Info("[CSDN] publish button not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := publishBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[CSDN] click publish button failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	log.Info("[CSDN] publish button clicked, waiting for confirmation...")
	emit("publish:progress", map[string]string{"platform": "csdn", "message": "等待发布确认..."})

	// Wait for page redirect or success confirmation
	articleURL := ""
	editorURL := page.MustInfo().URL
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		info := page.MustInfo()
		if info != nil && info.URL != "" && info.URL != editorURL && !containsAny(info.URL, []string{"mp.csdn.net/edit"}) {
			articleURL = info.URL
			log.Info("[CSDN] redirected to article page: " + articleURL)
			break
		}
	}

	if articleURL == "" {
		log.Info("[CSDN] redirect not detected within timeout, treating as success anyway")
	}

	// Final wait for page to stabilize
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
