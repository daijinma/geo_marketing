package publisher

import (
	"context"
	"fmt"
	"time"
)

// FlowPublisher executes a config-driven publish flow (JSON action list).
// If flow loading/execution fails, it falls back to AI assist / manual intervention.
type FlowPublisher struct{ *BasePublisher }

func (p *FlowPublisher) Publish(
	ctx context.Context,
	article Article,
	resume <-chan struct{},
	emit EventEmitter,
	aiConfig AIPublishConfig,
) error {
	log := p.logger

	emit("publish:progress", map[string]string{"platform": p.platform, "message": "加载发布动作配置..."})
	flow, err := LoadPublishFlow(p.platform)
	if err != nil {
		log.Info(fmt.Sprintf("[FlowPublish] flow not available platform=%s: %v", p.platform, err))
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	base := p.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		log.Info(fmt.Sprintf("[FlowPublish] launch browser failed platform=%s: %v", p.platform, err))
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}
	defer cleanup()
	defer base.Close()

	page := browser.MustPage("about:blank")
	defer page.Close()
	page.MustWaitLoad()
	page.MustWaitIdle()

	runner := NewFlowRunner(log, p.platform, emit, resume)

	emit("publish:progress", map[string]string{"platform": p.platform, "message": "执行发布动作序列..."})
	articleURL, err := runner.Run(ctx, page, flow, article)
	if err != nil {
		log.Info(fmt.Sprintf("[FlowPublish] flow execution failed platform=%s: %v", p.platform, err))
		// Best-effort: try to surface any confirm dialogs before falling back.
		_, _ = autoConfirmPopupButtons(ctx, page, "^(确定|确认|继续|下一步)$", "取消", true, 2, 600*time.Millisecond)
		return p.runAIAssist(ctx, article, resume, emit, aiConfig)
	}

	emit("publish:completed", map[string]interface{}{
		"platform":   p.platform,
		"success":    true,
		"articleUrl": articleURL,
	})
	return errAlreadyEmitted
}
