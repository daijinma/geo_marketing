package publisher

import "sync"

type CachedDecision struct {
	Action   string
	Selector string
	Value    string
	MS       int
}

var (
	cacheMu               sync.Mutex
	platformDecisionCache = make(map[string][]CachedDecision)
)

func getCachedDecisions(platform string) ([]CachedDecision, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	decisions, ok := platformDecisionCache[platform]
	if !ok || len(decisions) == 0 {
		return nil, false
	}
	// Return a copy to avoid external mutation
	out := make([]CachedDecision, len(decisions))
	copy(out, decisions)
	return out, true
}

func setCachedDecisions(platform string, decisions []CachedDecision) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if len(decisions) == 0 {
		delete(platformDecisionCache, platform)
		return
	}
	copyDecisions := make([]CachedDecision, len(decisions))
	copy(copyDecisions, decisions)
	platformDecisionCache[platform] = copyDecisions
}

func clearCachedDecisions(platform string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	delete(platformDecisionCache, platform)
}
