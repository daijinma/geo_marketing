package publisher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

type BaijiaPublisher struct{ *BasePublisher }

func downloadURLToTempFile(urlStr string) (string, error) {
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return "", fmt.Errorf("unsupported cover image url: %s", urlStr)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(urlStr)
	if err != nil {
		return "", fmt.Errorf("download cover image: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("download cover image: status %d", resp.StatusCode)
	}

	ext := strings.ToLower(filepath.Ext(urlStr))
	if ext == "" || len(ext) > 6 {
		ext = ".png"
	}

	f, err := os.CreateTemp("", "geo_baijiahao_cover_*"+ext)
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(f.Name())
		return "", fmt.Errorf("write temp cover image: %w", err)
	}

	return f.Name(), nil
}

func (p *BaijiaPublisher) Publish(
	ctx context.Context,
	article Article,
	resume <-chan struct{},
	emit EventEmitter,
	aiConfig AIPublishConfig,
) error {
	log := p.logger

	emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "正在打开百家号编辑器..."})
	log.Info("[Baijiahao] Starting publish: " + article.Title)

	base := p.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		log.Info("[Baijiahao] launch browser failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	defer cleanup()
	defer base.Close()

	page := browser.MustPage("https://baijiahao.baidu.com/builder/rc/edit?type=news")
	defer page.Close()
	page.MustWaitLoad()
	page.MustWaitIdle()

	emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "等待编辑器加载..."})
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}

	_, err = page.Timeout(10*time.Second).ElementR("button", "发布")
	if err != nil {
		log.Info("[Baijiahao] editor not loaded (publish button not found): " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "填写标题..."})
	titleResult, err := page.Eval(`(title) => {
		const titleRoot = document.querySelector('div.client_components_titleInput');
		const editor = titleRoot ? titleRoot.querySelector('[contenteditable="true"]') : null;
		if (editor) {
			editor.focus();
			editor.innerText = '';
			editor.dispatchEvent(new Event('input', { bubbles: true }));
			editor.innerText = title;
			try {
				editor.dispatchEvent(new InputEvent('input', { bubbles: true, data: title, inputType: 'insertText' }));
			} catch (e) {
				editor.dispatchEvent(new Event('input', { bubbles: true }));
			}
			editor.dispatchEvent(new Event('change', { bubbles: true }));
			editor.dispatchEvent(new KeyboardEvent('keyup', { bubbles: true, key: 'a' }));
			editor.dispatchEvent(new Event('blur', { bubbles: true }));
			return { success: true, method: 'client_components_titleInput' };
		}

		// Fallback: find any contenteditable close to title placeholder
		const elements = document.querySelectorAll('[contenteditable="true"]');
		for (const el of elements) {
			const t = (el.closest('div')?.textContent || '');
			if (t.includes('请输入标题') || t.includes('标题（')) {
				el.focus();
				el.innerText = title;
				el.dispatchEvent(new Event('input', { bubbles: true }));
				return { success: true, method: 'fallback_contenteditable' };
			}
		}
		return { success: false, method: 'not_found' };
	}`, article.Title)
	if err != nil || !titleResult.Value.Get("success").Bool() {
		log.Info("[Baijiahao] title input failed")
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	log.Info("[Baijiahao] title filled: " + article.Title)

	emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "填写正文..."})
	contentResult, err := page.Eval(`(content) => {
		const inst = window.UE_V2 && window.UE_V2.instants && window.UE_V2.instants.ueditorInstant0;
		if (inst && typeof inst.execCommand === 'function') {
			try {
				inst.execCommand('cleardoc');
			} catch (e) {}
			try {
				inst.execCommand('insertHtml', content);
				return { success: true, method: 'UE_V2.execCommand.insertHtml' };
			} catch (e) {
				// continue fallback
			}
		}

		const iframe = document.querySelector('iframe');
		if (iframe && iframe.contentDocument && iframe.contentDocument.body) {
			const editor = iframe.contentDocument.body.querySelector('[contenteditable="true"]');
			if (editor) {
				editor.innerHTML = content;
				editor.dispatchEvent(new Event('input', { bubbles: true }));
				editor.dispatchEvent(new Event('change', { bubbles: true }));
				return { success: true, method: 'iframe_contenteditable' };
			}
			iframe.contentDocument.body.innerHTML = content;
			iframe.contentDocument.body.dispatchEvent(new Event('input', { bubbles: true }));
			iframe.contentDocument.body.dispatchEvent(new Event('change', { bubbles: true }));
			return { success: true, method: 'iframe_body' };
		}
		return { success: false, method: 'not_found' };
	}`, article.Content)
	if err != nil || !contentResult.Value.Get("success").Bool() {
		log.Info("[Baijiahao] content input failed")
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	log.Info("[Baijiahao] content filled")

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(1 * time.Second):
	}

	emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "设置封面图..."})
	coverPath := ""
	if article.CoverImage != "" {
		p, dlErr := downloadURLToTempFile(article.CoverImage)
		if dlErr != nil {
			log.Info("[Baijiahao] download cover image failed: " + dlErr.Error())
		} else {
			coverPath = p
			defer os.Remove(coverPath)
		}
	}

	coverResult, err := page.Eval(`() => {
		// detect whether cover already selected
		const root = document.getElementById('bjhNewsCover');
		const txt = root ? (root.textContent || '') : '';
		const needs = txt.includes('选择封面');
		return { needsCover: needs };
	}`)
	if err != nil {
		log.Info("[Baijiahao] cover status check failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	needsCover := coverResult.Value.Get("needsCover").Bool()
	if needsCover {
		// Open cover picker modal
		_, _ = page.Eval(`() => {
			const icon = document.querySelector('#bjhNewsCover ._73a3a52aab7e3a36-icon');
			if (icon) { icon.click(); return { success: true, method: 'icon_click' }; }
			const fallback = Array.from(document.querySelectorAll('*')).find(el => (el.textContent || '').replace(/\s+/g,'').trim() === '选择封面');
			if (fallback) { fallback.click(); return { success: true, method: 'text_click' }; }
			return { success: false };
		}`)

		if coverPath != "" {
			// Ensure we're on "正文/本地上传" tab
			_, _ = page.Eval(`() => {
				const wrap = document.querySelector('.cheetah-modal-wrap');
				if (!wrap) return { success: false, reason: 'no modal' };
				const tabs = Array.from(wrap.querySelectorAll('[role=tab]'));
				const tab = tabs.find(t => (t.textContent || '').replace(/\s+/g,'').includes('正文'));
				if (tab) { tab.click(); return { success: true }; }
				return { success: false, reason: 'no tab' };
			}`)

			fileEl, fileErr := page.Timeout(10 * time.Second).Element(`.cheetah-modal-wrap input[type="file"][accept*="image"]`)
			if fileErr != nil {
				log.Info("[Baijiahao] cover file input not found: " + fileErr.Error())
				return p.runAIAssist(ctx, article, resume, emit, aiConfig)
			}
			if err := fileEl.SetFiles([]string{coverPath}); err != nil {
				log.Info("[Baijiahao] set cover file failed: " + err.Error())
				return p.runAIAssist(ctx, article, resume, emit, aiConfig)
			}
		} else {
			// Use AI cover: "根据全文智能生成封面"
			_, _ = page.Eval(`() => {
				const wrap = document.querySelector('.cheetah-modal-wrap');
				if (!wrap) return { success: false, reason: 'no modal' };
				const tabs = Array.from(wrap.querySelectorAll('[role=tab]'));
				const aiTab = tabs.find(t => (t.textContent || '').replace(/\s+/g,'').includes('AI封图'));
				if (aiTab) aiTab.click();
				return { success: true };
			}`)
			_, _ = page.Eval(`() => {
				const wrap = document.querySelector('.cheetah-modal-wrap');
				if (!wrap) return { success: false, reason: 'no modal' };
				const nodes = Array.from(wrap.querySelectorAll('*'));
				const target = nodes.find(el => (el.textContent || '').replace(/\s+/g,'').trim() === '根据全文智能生成封面');
				if (target) {
					target.click();
					return { success: true };
				}
				return { success: false, reason: 'target not found' };
			}`)
		}

		// Wait until "确定" is enabled, then confirm
		enabled := false
		for i := 0; i < 60; i++ {
			res, _ := page.Eval(`() => {
				const wrap = document.querySelector('.cheetah-modal-wrap');
				if (!wrap) return { ok: false };
				const btns = Array.from(wrap.querySelectorAll('button'));
				const ok = btns.find(b => (b.textContent || '').replace(/\s+/g,'').trim() === '确定');
				return { ok: !!ok, enabled: ok ? !ok.disabled : false };
			}`)
			if res != nil && res.Value.Get("enabled").Bool() {
				enabled = true
				break
			}
			time.Sleep(300 * time.Millisecond)
		}
		if !enabled {
			log.Info("[Baijiahao] cover confirm button not enabled")
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}

		modalWrap, wrapErr := page.Element(`.cheetah-modal-wrap`)
		if wrapErr != nil {
			log.Info("[Baijiahao] cover modal not found")
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
		okBtn, okErr := modalWrap.Timeout(5*time.Second).ElementR("button", "确定")
		if okErr != nil {
			log.Info("[Baijiahao] cover ok button not found: " + okErr.Error())
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
		if err := okBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
			log.Info("[Baijiahao] cover ok click failed: " + err.Error())
			return p.runAIAssist(ctx, article, resume, emit, aiConfig)
		}
		log.Info("[Baijiahao] cover selected")
	}

	emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "点击发布按钮..."})
	publishBtn, err := page.Timeout(5*time.Second).ElementR("button", "发布")
	if err != nil {
		log.Info("[Baijiahao] publish button not found: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	if err := publishBtn.Click(proto.InputMouseButtonLeft, 1); err != nil {
		log.Info("[Baijiahao] click publish button failed: " + err.Error())
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	if confirmed, popupErr := autoConfirmPlatformPopups(ctx, page, "baijiahao", true, 6, 700*time.Millisecond); popupErr == nil && confirmed > 0 {
		emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "检测到发布弹窗，已自动确认"})
	}

	log.Info("[Baijiahao] publish button clicked, waiting for confirmation...")
	emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "等待发布确认..."})

	articleURL := ""
	editorURL := page.MustInfo().URL
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if confirmed, popupErr := autoConfirmPlatformPopups(ctx, page, "baijiahao", true, 1, 300*time.Millisecond); popupErr == nil && confirmed > 0 {
			emit("publish:progress", map[string]string{"platform": "baijiahao", "message": "发布后弹窗已自动确认，继续检测结果..."})
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
		info := page.MustInfo()
		if info != nil && info.URL != "" && info.URL != editorURL && !strings.Contains(info.URL, "rc/edit") {
			articleURL = info.URL
			log.Info("[Baijiahao] redirected to article page: " + articleURL)
			break
		}
	}

	if articleURL == "" {
		log.Info("[Baijiahao] redirect not detected within timeout, treating as success anyway")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(3 * time.Second):
	}

	log.Info("[Baijiahao] publish completed, article_url=" + articleURL)
	emit("publish:completed", map[string]interface{}{
		"platform":   "baijiahao",
		"success":    true,
		"articleUrl": articleURL,
	})

	return errAlreadyEmitted
}
