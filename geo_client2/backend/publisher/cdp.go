package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"geo_client2/backend/logger"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type CDPObserver struct {
	page          *rod.Page
	browser       *rod.Browser
	logger        *logger.Logger
	networkEvents []NetworkEvent
	consoleEvents []ConsoleEvent
	mu            sync.Mutex
	stopChan      chan struct{}
	isObserving   bool
}

type NetworkEvent struct {
	RequestID   string            `json:"request_id"`
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Status      int               `json:"status,omitempty"`
	Type        string            `json:"type"`
	Headers     map[string]string `json:"headers,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	IsCompleted bool              `json:"is_completed"`
	Error       string            `json:"error,omitempty"`
}

type ConsoleEvent struct {
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	URL       string    `json:"url,omitempty"`
	Line      int       `json:"line,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type FormField struct {
	Selector    string `json:"selector"`
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	ID          string `json:"id,omitempty"`
	Placeholder string `json:"placeholder,omitempty"`
	Label       string `json:"label,omitempty"`
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required"`
	Disabled    bool   `json:"disabled"`
	Visible     bool   `json:"visible"`
}

type InteractiveElement struct {
	Selector    string      `json:"selector"`
	Tag         string      `json:"tag"`
	Text        string      `json:"text"`
	Role        string      `json:"role,omitempty"`
	AriaLabel   string      `json:"aria_label,omitempty"`
	Href        string      `json:"href,omitempty"`
	Type        string      `json:"type,omitempty"`
	Visible     bool        `json:"visible"`
	Clickable   bool        `json:"clickable"`
	BoundingBox BoundingBox `json:"bounding_box"`
}

type BoundingBox struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type EnhancedObservation struct {
	URL                 string               `json:"url"`
	Title               string               `json:"title"`
	Timestamp           time.Time            `json:"timestamp"`
	FormFields          []FormField          `json:"form_fields"`
	InteractiveElements []InteractiveElement `json:"interactive_elements"`
	VisibleText         string               `json:"visible_text"`
	PendingRequests     []NetworkEvent       `json:"pending_requests,omitempty"`
	RecentResponses     []NetworkEvent       `json:"recent_responses,omitempty"`
	RecentConsole       []ConsoleEvent       `json:"recent_console,omitempty"`
	IsLoading           bool                 `json:"is_loading"`
	HasDialogs          bool                 `json:"has_dialogs"`
	DialogMessage       string               `json:"dialog_message,omitempty"`
	Screenshot          string               `json:"screenshot,omitempty"`
}

func NewCDPObserver(browser *rod.Browser, page *rod.Page) *CDPObserver {
	return &CDPObserver{
		page:          page,
		browser:       browser,
		logger:        logger.GetLogger(),
		networkEvents: make([]NetworkEvent, 0, 100),
		consoleEvents: make([]ConsoleEvent, 0, 50),
		stopChan:      make(chan struct{}),
	}
}

func (o *CDPObserver) StartObserving(ctx context.Context) error {
	o.mu.Lock()
	if o.isObserving {
		o.mu.Unlock()
		return nil
	}
	o.isObserving = true
	o.mu.Unlock()

	_ = proto.NetworkEnable{}.Call(o.page)

	go o.listenNetworkEvents(ctx)
	go o.listenConsoleEvents(ctx)

	return nil
}

func (o *CDPObserver) StopObserving() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.isObserving {
		return
	}

	close(o.stopChan)
	o.isObserving = false
	o.stopChan = make(chan struct{})
}

func (o *CDPObserver) listenNetworkEvents(ctx context.Context) {
	page := o.page

	go page.EachEvent(func(e *proto.NetworkRequestWillBeSent) {
		o.mu.Lock()
		defer o.mu.Unlock()

		event := NetworkEvent{
			RequestID: string(e.RequestID),
			URL:       e.Request.URL,
			Method:    e.Request.Method,
			Type:      string(e.Type),
			Timestamp: time.Now(),
		}

		if len(o.networkEvents) >= 100 {
			o.networkEvents = o.networkEvents[1:]
		}
		o.networkEvents = append(o.networkEvents, event)
	}, func(e *proto.NetworkResponseReceived) {
		o.mu.Lock()
		defer o.mu.Unlock()

		for i := len(o.networkEvents) - 1; i >= 0; i-- {
			if o.networkEvents[i].RequestID == string(e.RequestID) {
				o.networkEvents[i].Status = e.Response.Status
				o.networkEvents[i].IsCompleted = true
				break
			}
		}
	}, func(e *proto.NetworkLoadingFailed) {
		o.mu.Lock()
		defer o.mu.Unlock()

		for i := len(o.networkEvents) - 1; i >= 0; i-- {
			if o.networkEvents[i].RequestID == string(e.RequestID) {
				o.networkEvents[i].Error = e.ErrorText
				o.networkEvents[i].IsCompleted = true
				break
			}
		}
	})()
}

func (o *CDPObserver) listenConsoleEvents(ctx context.Context) {
	page := o.page

	go page.EachEvent(func(e *proto.RuntimeConsoleAPICalled) {
		o.mu.Lock()
		defer o.mu.Unlock()

		var text strings.Builder
		for _, arg := range e.Args {
			val := arg.Value.Val()
			if val != nil {
				text.WriteString(fmt.Sprintf("%v ", val))
			}
		}

		event := ConsoleEvent{
			Type:      string(e.Type),
			Text:      strings.TrimSpace(text.String()),
			Timestamp: time.Now(),
		}

		if len(o.consoleEvents) >= 50 {
			o.consoleEvents = o.consoleEvents[1:]
		}
		o.consoleEvents = append(o.consoleEvents, event)
	})()
}

func (o *CDPObserver) CaptureEnhancedObservation(includeScreenshot bool) (*EnhancedObservation, error) {
	page := o.page

	info := page.MustInfo()
	obs := &EnhancedObservation{
		URL:       info.URL,
		Title:     page.MustEval("() => document.title").String(),
		Timestamp: time.Now(),
	}

	formFieldsJSON := page.MustEval(`() => {
		const isVisible = (el) => {
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			return rect.width > 0 && rect.height > 0 && 
				   style.visibility !== 'hidden' && 
				   style.display !== 'none' &&
				   style.opacity !== '0';
		};
		
		const getLabel = (el) => {
			if (el.id) {
				const label = document.querySelector('label[for="' + el.id + '"]');
				if (label) return label.innerText.trim();
			}
			const parentLabel = el.closest('label');
			if (parentLabel) return parentLabel.innerText.trim();
			if (el.getAttribute('aria-label')) return el.getAttribute('aria-label');
			if (el.placeholder) return el.placeholder;
			return '';
		};
		
		const cssPath = (el) => {
			if (!el || !el.tagName) return '';
			if (el.id) return '#' + el.id;
			
			const parts = [];
			let node = el;
			while (node && node.tagName && parts.length < 5) {
				let selector = node.tagName.toLowerCase();
				if (node.id) {
					parts.unshift('#' + node.id);
					break;
				}
				if (node.className && typeof node.className === 'string') {
					const cls = node.className.trim().split(/\s+/).filter(c => !c.includes(':'))[0];
					if (cls) selector += '.' + cls;
				}
				if (node.parentElement) {
					const siblings = Array.from(node.parentElement.children).filter(n => n.tagName === node.tagName);
					if (siblings.length > 1) {
						const index = siblings.indexOf(node) + 1;
						selector += ':nth-of-type(' + index + ')';
					}
				}
				parts.unshift(selector);
				node = node.parentElement;
			}
			return parts.join(' > ');
		};
		
		const fields = [];
		const selectors = [
			'input:not([type="hidden"]):not([type="submit"]):not([type="button"])',
			'textarea',
			'select',
			'[contenteditable="true"]',
			'[role="textbox"]',
			'[role="combobox"]'
		];
		
		const elements = document.querySelectorAll(selectors.join(','));
		for (const el of elements) {
			if (fields.length >= 30) break;
			
			const field = {
				selector: cssPath(el),
				type: el.type || el.tagName.toLowerCase(),
				name: el.name || '',
				id: el.id || '',
				placeholder: el.placeholder || '',
				label: getLabel(el),
				value: el.value || el.innerText || '',
				required: el.required || el.getAttribute('aria-required') === 'true',
				disabled: el.disabled || el.getAttribute('aria-disabled') === 'true',
				visible: isVisible(el)
			};
			
			if (el.getAttribute('contenteditable') === 'true' || el.getAttribute('role') === 'textbox') {
				field.type = 'contenteditable';
			}
			
			fields.push(field);
		}
		
		return JSON.stringify(fields);
	}`).String()

	if err := json.Unmarshal([]byte(formFieldsJSON), &obs.FormFields); err != nil {
		o.logger.Warn(fmt.Sprintf("[CDPObserver] Failed to parse form fields: %v", err))
		obs.FormFields = []FormField{}
	}

	interactiveJSON := page.MustEval(`() => {
		const isVisible = (el) => {
			const rect = el.getBoundingClientRect();
			const style = window.getComputedStyle(el);
			return rect.width > 0 && rect.height > 0 && 
				   style.visibility !== 'hidden' && 
				   style.display !== 'none';
		};
		
		const cssPath = (el) => {
			if (!el || !el.tagName) return '';
			if (el.id) return '#' + el.id;
			
			const parts = [];
			let node = el;
			while (node && node.tagName && parts.length < 5) {
				let selector = node.tagName.toLowerCase();
				if (node.id) {
					parts.unshift('#' + node.id);
					break;
				}
				if (node.className && typeof node.className === 'string') {
					const cls = node.className.trim().split(/\s+/).filter(c => !c.includes(':'))[0];
					if (cls) selector += '.' + cls;
				}
				if (node.parentElement) {
					const siblings = Array.from(node.parentElement.children).filter(n => n.tagName === node.tagName);
					if (siblings.length > 1) {
						const index = siblings.indexOf(node) + 1;
						selector += ':nth-of-type(' + index + ')';
					}
				}
				parts.unshift(selector);
				node = node.parentElement;
			}
			return parts.join(' > ');
		};
		
		const elements = [];
		const selectors = [
			'button',
			'a[href]',
			'[role="button"]',
			'[role="link"]',
			'[role="tab"]',
			'[role="menuitem"]',
			'input[type="submit"]',
			'input[type="button"]',
			'[onclick]',
			'[tabindex]:not([tabindex="-1"])'
		];
		
		const nodes = document.querySelectorAll(selectors.join(','));
		for (const el of nodes) {
			if (elements.length >= 50) break;
			if (!isVisible(el)) continue;
			
			const rect = el.getBoundingClientRect();
			const text = (el.innerText || el.value || el.getAttribute('aria-label') || '').trim().substring(0, 100);
			
			if (!text && !el.getAttribute('aria-label') && !el.title) continue;
			
			elements.push({
				selector: cssPath(el),
				tag: el.tagName.toLowerCase(),
				text: text,
				role: el.getAttribute('role') || '',
				aria_label: el.getAttribute('aria-label') || '',
				href: el.href || '',
				type: el.type || '',
				visible: true,
				clickable: true,
				bounding_box: {
					x: rect.x,
					y: rect.y,
					width: rect.width,
					height: rect.height
				}
			});
		}
		
		return JSON.stringify(elements);
	}`).String()

	if err := json.Unmarshal([]byte(interactiveJSON), &obs.InteractiveElements); err != nil {
		o.logger.Warn(fmt.Sprintf("[CDPObserver] Failed to parse interactive elements: %v", err))
		obs.InteractiveElements = []InteractiveElement{}
	}

	obs.VisibleText = limitString(page.MustEval("() => document.body ? document.body.innerText : ''").String(), 6000)

	o.mu.Lock()
	pendingRequests := []NetworkEvent{}
	recentResponses := []NetworkEvent{}
	for _, e := range o.networkEvents {
		if !e.IsCompleted {
			pendingRequests = append(pendingRequests, e)
		} else if time.Since(e.Timestamp) < 10*time.Second {
			recentResponses = append(recentResponses, e)
		}
	}
	obs.PendingRequests = pendingRequests
	if len(recentResponses) > 10 {
		recentResponses = recentResponses[len(recentResponses)-10:]
	}
	obs.RecentResponses = recentResponses

	if len(o.consoleEvents) > 10 {
		obs.RecentConsole = o.consoleEvents[len(o.consoleEvents)-10:]
	} else {
		obs.RecentConsole = o.consoleEvents
	}
	o.mu.Unlock()

	loadingState := page.MustEval(`() => {
		return {
			isLoading: document.readyState !== 'complete',
			hasDialogs: false
		};
	}`)
	obs.IsLoading = loadingState.Get("isLoading").Bool()

	if includeScreenshot {
		screenshot, err := page.Screenshot(true, &proto.PageCaptureScreenshot{Format: "png"})
		if err == nil {
			obs.Screenshot = fmt.Sprintf("data:image/png;base64,%s", string(screenshot))
		}
	}

	return obs, nil
}

func (o *CDPObserver) WaitForNetworkIdle(ctx context.Context, idleDuration time.Duration) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	idleStart := time.Time{}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			o.mu.Lock()
			hasPending := false
			for _, e := range o.networkEvents {
				if !e.IsCompleted && time.Since(e.Timestamp) < 30*time.Second {
					hasPending = true
					break
				}
			}
			o.mu.Unlock()

			if !hasPending {
				if idleStart.IsZero() {
					idleStart = time.Now()
				} else if time.Since(idleStart) >= idleDuration {
					return nil
				}
			} else {
				idleStart = time.Time{}
			}
		}
	}
}

func (o *CDPObserver) WaitForElement(ctx context.Context, selector string, timeout time.Duration) (*rod.Element, error) {
	return o.page.Timeout(timeout).Element(selector)
}

func (o *CDPObserver) WaitForElementText(ctx context.Context, selector, text string, timeout time.Duration) (*rod.Element, error) {
	return o.page.Timeout(timeout).ElementR(selector, text)
}

func (o *CDPObserver) GetPendingRequestCount() int {
	o.mu.Lock()
	defer o.mu.Unlock()

	count := 0
	for _, e := range o.networkEvents {
		if !e.IsCompleted && time.Since(e.Timestamp) < 30*time.Second {
			count++
		}
	}
	return count
}

func (o *CDPObserver) ClearEvents() {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.networkEvents = make([]NetworkEvent, 0, 100)
	o.consoleEvents = make([]ConsoleEvent, 0, 50)
}

func (o *CDPObserver) FindFormFieldByLabel(obs *EnhancedObservation, labelText string) *FormField {
	labelLower := strings.ToLower(labelText)
	for i := range obs.FormFields {
		f := &obs.FormFields[i]
		if strings.Contains(strings.ToLower(f.Label), labelLower) ||
			strings.Contains(strings.ToLower(f.Placeholder), labelLower) ||
			strings.Contains(strings.ToLower(f.Name), labelLower) {
			return f
		}
	}
	return nil
}

func (o *CDPObserver) FindButtonByText(obs *EnhancedObservation, buttonText string) *InteractiveElement {
	textLower := strings.ToLower(buttonText)
	for i := range obs.InteractiveElements {
		el := &obs.InteractiveElements[i]
		if strings.Contains(strings.ToLower(el.Text), textLower) ||
			strings.Contains(strings.ToLower(el.AriaLabel), textLower) {
			return el
		}
	}
	return nil
}
