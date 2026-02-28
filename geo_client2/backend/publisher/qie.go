package publisher

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

type QiePublisher struct{ *BasePublisher }

func (p *QiePublisher) Publish(
	ctx context.Context,
	article Article,
	resume <-chan struct{},
	emit EventEmitter,
	aiConfig AIPublishConfig,
) error {
	log := p.logger

	emit("publish:progress", map[string]string{"platform": "qie", "message": "正在打开企鹅号编辑器..."})
	log.Info("[Qie] Starting publish: " + article.Title)

	base := p.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		log.Info("[Qie] launch browser failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	defer cleanup()
	defer base.Close()

	page := browser.MustPage("https://om.qq.com/main/creation/article")
	defer page.Close()
	page.MustWaitLoad()
	page.MustWaitIdle()

	emit("publish:progress", map[string]string{"platform": "qie", "message": "等待编辑器加载..."})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}

	_, err = page.Timeout(10*time.Second).ElementR("button", "发布")
	if err != nil {
		log.Info("[Qie] editor not loaded (publish button not found): " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	emit("publish:progress", map[string]string{"platform": "qie", "message": "填写标题..."})
	var titleOK bool
	for i := 0; i < 3; i++ {
		titleResult, evalErr := page.Eval(`(title) => {
		const isVisible = (el) => {
			if (!el) return false;
			const style = window.getComputedStyle(el);
			if (!style || style.visibility === 'hidden' || style.display === 'none') return false;
			const rect = el.getBoundingClientRect();
			return rect.width > 0 && rect.height > 0;
		};

		const markInput = (el, value) => {
			el.focus();
			el.innerText = '';
			el.dispatchEvent(new Event('input', { bubbles: true }));
			try {
				document.execCommand('insertText', false, value);
			} catch (e) {
				el.innerText = value;
			}
			if (!el.innerText || el.innerText.trim() === '') {
				el.innerText = value;
			}
			try {
				el.dispatchEvent(new InputEvent('input', { bubbles: true, data: value, inputType: 'insertText' }));
			} catch (e) {
				el.dispatchEvent(new Event('input', { bubbles: true }));
			}
			el.dispatchEvent(new Event('change', { bubbles: true }));
			el.dispatchEvent(new KeyboardEvent('keyup', { bubbles: true, key: 'a' }));
		};

		const candidates = Array.from(document.querySelectorAll('[contenteditable="true"], [contenteditable="plaintext-only"]')).filter(isVisible);
		for (const el of candidates) {
			const placeholder = ((el.getAttribute('placeholder') || '') + ' ' + (el.getAttribute('data-placeholder') || '') + ' ' + (el.getAttribute('aria-label') || '')).trim();
			const text = (el.textContent || '').trim();
			if (placeholder.includes('标题') || text.includes('请输入标题')) {
				markInput(el, title);
				return { success: true, method: 'contenteditable_placeholder_or_text' };
			}
		}

		const nearPrompt = Array.from(document.querySelectorAll('*')).find(el => {
			const txt = (el.textContent || '').replace(/\s+/g, '');
			return txt.includes('请输入标题');
		});
		if (nearPrompt) {
			const scoped = nearPrompt.closest('main, section, article, div') || nearPrompt.parentElement;
			const fallback = scoped ? scoped.querySelector('[contenteditable="true"]') : null;
			if (fallback && isVisible(fallback)) {
				markInput(fallback, title);
				return { success: true, method: 'near_prompt_fallback' };
			}
		}

		return { success: false, method: 'title_not_found' };
	}`, article.Title)
		if evalErr == nil && titleResult.Value.Get("success").Bool() {
			titleOK = true
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1200 * time.Millisecond):
		}
	}
	if !titleOK {
		log.Info("[Qie] title input failed")
		return emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "未能自动填写企鹅号标题，请手动补全后点击继续", resume, emit)
	}
	log.Info("[Qie] title filled: " + article.Title)

	emit("publish:progress", map[string]string{"platform": "qie", "message": "填写正文..."})
	var contentOK bool
	for i := 0; i < 3; i++ {
		contentResult, evalErr := page.Eval(`(content) => {
		const isVisible = (el) => {
			if (!el) return false;
			const style = window.getComputedStyle(el);
			if (!style || style.visibility === 'hidden' || style.display === 'none') return false;
			const rect = el.getBoundingClientRect();
			return rect.width > 0 && rect.height > 0;
		};

		const editors = Array.from(document.querySelectorAll('main [contenteditable="true"], [contenteditable="true"][role="textbox"]')).filter(isVisible);
		const score = (el) => {
			const placeholder = (el.getAttribute('placeholder') || '').trim();
			const text = (el.textContent || '').trim();
			if (placeholder.includes('标题') || text.includes('请输入标题')) return -1;
			let s = 0;
			if (el.closest('main')) s += 3;
			if (el.getAttribute('role') === 'textbox') s += 2;
			if ((el.getAttribute('data-placeholder') || '').length > 0) s += 1;
			const rect = el.getBoundingClientRect();
			s += Math.min(5, Math.floor(rect.height / 80));
			return s;
		};

		const target = editors
			.map(el => ({ el, score: score(el) }))
			.filter(item => item.score >= 0)
			.sort((a, b) => b.score - a.score)[0];

		if (!target) {
			return { success: false, method: 'content_not_found' };
		}

		const el = target.el;
		el.focus();
		el.innerHTML = '';
		el.dispatchEvent(new Event('input', { bubbles: true }));
		try {
			document.execCommand('insertHTML', false, content);
		} catch (e) {
			el.innerHTML = content;
		}
		if (!el.innerHTML || el.innerHTML.trim() === '') {
			el.innerHTML = content;
		}
		el.dispatchEvent(new Event('input', { bubbles: true }));
		el.dispatchEvent(new Event('change', { bubbles: true }));
		try {
			el.dispatchEvent(new InputEvent('input', { bubbles: true, data: content, inputType: 'insertFromPaste' }));
		} catch (e) {
			el.dispatchEvent(new Event('input', { bubbles: true }));
		}

		return { success: true, method: 'ranked_contenteditable' };
	}`, article.Content)
		if evalErr == nil && contentResult.Value.Get("success").Bool() {
			contentOK = true
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1200 * time.Millisecond):
		}
	}
	if !contentOK {
		log.Info("[Qie] content input failed")
		return emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "未能自动填写企鹅号正文，请手动补全后点击继续", resume, emit)
	}
	log.Info("[Qie] content filled")

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}

	if article.CoverImage != "" {
		emit("publish:progress", map[string]string{"platform": "qie", "message": "上传封面图..."})
		log.Info("[Qie] cover image upload not implemented, skipping")
	}

	emit("publish:progress", map[string]string{"platform": "qie", "message": "添加自主声明..."})
	declarationBtn, err := page.Timeout(3*time.Second).ElementR("button", "添加内容自主声明")
	if err != nil {
		log.Info("[Qie] declaration button not found: " + err.Error())
		return emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "未找到“添加内容自主声明”，请手动完成声明后点击继续", resume, emit)
	}
	if err := declarationBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[Qie] declaration button click failed: " + err.Error())
		return emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "无法打开内容声明弹框，请手动完成声明后点击继续", resume, emit)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1200 * time.Millisecond):
	}

	declareResult, err := page.Eval(`() => {
		const isVisible = (el) => {
			if (!el) return false;
			const style = window.getComputedStyle(el);
			if (!style || style.visibility === 'hidden' || style.display === 'none') return false;
			const rect = el.getBoundingClientRect();
			return rect.width > 0 && rect.height > 0;
		};
		const norm = (s) => (s || '').replace(/\s+/g, ' ').trim();

		const clickOption = (re) => {
			const candidates = Array.from(document.querySelectorAll('label, button, [role="button"], span, div, li, p'))
				.filter(isVisible);
			for (const el of candidates) {
				const text = norm(el.innerText || el.textContent || '');
				if (!re.test(text)) continue;
				const clickable = el.closest('label, button, [role="button"], li, div') || el;
				clickable.click();
				return true;
			}
			return false;
		};

		const aiSelected = clickOption(/AI(生成|创作)?声明|AIGC声明|AI声明/);
		const contentSelected = clickOption(/发布内容自主声明|内容自主声明|自主声明/);

		return { aiSelected, contentSelected, ok: aiSelected && contentSelected };
	}`)
	if err != nil {
		log.Info("[Qie] declaration selection eval failed: " + err.Error())
		return emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "声明选项自动勾选失败，请手动勾选 AI 生成声明和发布内容自主声明后点击继续", resume, emit)
	}

	aiSelected := declareResult.Value.Get("aiSelected").Bool()
	contentSelected := declareResult.Value.Get("contentSelected").Bool()
	if !aiSelected || !contentSelected {
		return emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "请在声明弹框中勾选 AI 生成声明和发布内容自主声明后点击继续", resume, emit)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(600 * time.Millisecond):
	}

	declConfirmed, _, err := clickVisibleConfirmButton(page, `确定|完成|提交`, `发布|取消|关闭|返回`, true)
	if err != nil || !declConfirmed {
		return emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "声明弹框确认失败，请手动点击“确定/完成”后点击继续", resume, emit)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(800 * time.Millisecond):
	}

	emit("publish:progress", map[string]string{"platform": "qie", "message": "点击发布按钮..."})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
	}

	publishResult, err := page.Eval(`() => {
		const isVisible = (el) => {
			if (!el) return false;
			const style = window.getComputedStyle(el);
			if (!style || style.visibility === 'hidden' || style.display === 'none') return false;
			const rect = el.getBoundingClientRect();
			return rect.width > 0 && rect.height > 0;
		};

		const norm = (s) => (s || '').replace(/\s+/g, ' ').trim();
		const candidates = Array.from(document.querySelectorAll('button, [role="button"], a')).filter(isVisible);
		for (const el of candidates) {
			const text = norm(el.innerText || el.textContent || el.getAttribute('aria-label') || '');
			if (text === '发布') {
				el.click();
				return { success: true, method: 'exact_text_publish' };
			}
		}

		const fallback = candidates.find(el => {
			const text = norm(el.innerText || el.textContent || '');
			return text.includes('发布') && !text.includes('草稿') && !text.includes('定时');
		});
		if (fallback) {
			fallback.click();
			return { success: true, method: 'fallback_contains_publish' };
		}

		return { success: false, method: 'publish_button_not_found' };
	}`)
	if err != nil || !publishResult.Value.Get("success").Bool() {
		log.Info("[Qie] click publish button failed")
		return emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "未找到企鹅号发布按钮，请手动点击发布后点击继续", resume, emit)
	}

	confirmedCount, popupErr := autoConfirmPlatformPopups(ctx, page, "qie", true, 6, 900*time.Millisecond)
	if popupErr == nil && confirmedCount > 0 {
		emit("publish:progress", map[string]string{"platform": "qie", "message": "检测到发布弹窗，已自动点击确定，继续处理..."})
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
	}

	emit("publish:progress", map[string]string{"platform": "qie", "message": "已点击发布，正在判断是否发布成功..."})

	articleURL := ""
	checkPublished := func() (bool, bool, error) {
		res, evalErr := page.Eval(`() => {
			const isVisible = (el) => {
				if (!el) return false;
				const style = window.getComputedStyle(el);
				if (!style || style.visibility === 'hidden' || style.display === 'none') return false;
				const rect = el.getBoundingClientRect();
				return rect.width > 0 && rect.height > 0;
			};
			const norm = (s) => (s || '').replace(/\s+/g, ' ').trim();
			const text = document.body ? norm(document.body.innerText || '') : '';
			const url = location.href || '';
			const successByText = /发布成功|已发布|提交成功|审核中|正在审核/.test(text);
			const stillEditor = url.includes('/main/creation/article');
			const hasSecondConfirm = Array.from(document.querySelectorAll('button, [role="button"], a'))
				.filter(isVisible)
				.some(el => /确认发布|确定发布|立即发布|提交发布/.test(norm(el.innerText || el.textContent || '')));
			return {
				successByText,
				stillEditor,
				hasSecondConfirm,
				url,
			};
		}`)
		if evalErr != nil {
			return false, false, evalErr
		}

		url := res.Value.Get("url").String()
		if url != "" {
			articleURL = url
		}

		successByText := res.Value.Get("successByText").Bool()
		stillEditor := res.Value.Get("stillEditor").Bool()
		hasSecondConfirm := res.Value.Get("hasSecondConfirm").Bool()
		successByURL := articleURL != "" && !stillEditor

		return successByText || successByURL, hasSecondConfirm, nil
	}

	published := false
	hasSecondConfirm := false
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if clickedCount, popupErr := autoConfirmPlatformPopups(ctx, page, "qie", true, 2, 300*time.Millisecond); popupErr == nil && clickedCount > 0 {
			emit("publish:progress", map[string]string{"platform": "qie", "message": "发布后弹窗已自动确认，继续检测结果..."})
		}

		ok, needConfirm, checkErr := checkPublished()
		if checkErr == nil {
			if ok {
				published = true
				break
			}
			hasSecondConfirm = needConfirm
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(800 * time.Millisecond):
		}
	}

	if !published && hasSecondConfirm {
		emit("publish:progress", map[string]string{"platform": "qie", "message": "检测到发布确认弹层，请手动完成最终确认"})
		if err := emitNeedsManual(ctx, "qie", fmt.Sprintf("qie-%d", time.Now().UnixNano()), "请在企鹅号页面完成最终发布确认，出现发布成功后点击继续", resume, emit); err != nil {
			return err
		}

		deadline = time.Now().Add(20 * time.Second)
		for time.Now().Before(deadline) {
			ok, _, checkErr := checkPublished()
			if checkErr == nil && ok {
				published = true
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(800 * time.Millisecond):
			}
		}
	}

	if !published {
		log.Info("[Qie] publish success not detected")
		return fmt.Errorf("企鹅号未检测到发布成功，请检查是否卡在二次确认或校验步骤")
	}

	log.Info("[Qie] publish succeeded")
	emit("publish:progress", map[string]string{"platform": "qie", "message": "企鹅号发布成功"})
	emit("publish:completed", map[string]interface{}{
		"platform":   "qie",
		"success":    true,
		"articleUrl": articleURL,
	})

	return errAlreadyEmitted
}
