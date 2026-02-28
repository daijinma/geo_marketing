package publisher

import (
	"context"
	"fmt"
	"time"

	"github.com/go-rod/rod"
)

// FlowPublisher executes a config-driven publish flow (JSON action list).
// Flow failures are returned as errors directly — no AI-assist fallback.
type FlowPublisher struct {
	*BasePublisher
	jobTaskID string
}

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
		emit("publish:failed", map[string]interface{}{"platform": p.platform, "error": err.Error()})
		return err
	}

	base := p.getBaseProvider()
	browser, cleanup, err := base.LaunchBrowser(false)
	if err != nil {
		log.Info(fmt.Sprintf("[FlowPublish] launch browser failed platform=%s: %v", p.platform, err))
		emit("publish:failed", map[string]interface{}{"platform": p.platform, "error": err.Error()})
		return err
	}
	defer cleanup()
	defer base.Close()

	page := browser.MustPage("about:blank")
	defer page.Close()
	_ = page.WaitLoad()
	rod.Try(func() { _ = page.Timeout(3000 * time.Millisecond).WaitStable(500 * time.Millisecond) })

	runner := NewFlowRunner(log, p.platform, p.jobTaskID, emit, resume).WithAIConfig(aiConfig)

	emit("publish:progress", map[string]string{"platform": p.platform, "message": "执行发布动作序列..."})
	articleURL, err := runner.Run(ctx, page, flow, article)
	if err != nil {
		log.Info(fmt.Sprintf("[FlowPublish] flow execution failed platform=%s: %v", p.platform, err))
		emit("publish:failed", map[string]interface{}{"platform": p.platform, "error": err.Error()})
		return err
	}

	emit("publish:completed", map[string]interface{}{
		"platform":   p.platform,
		"success":    true,
		"articleUrl": articleURL,
	})
	return errAlreadyEmitted
}
