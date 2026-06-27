package llm

import (
	"net/http"

	"github.com/Ayush10212/receipts/core/config"
)

// NewRouterWithURLsForTest exposes the internal constructor for unit tests.
func NewRouterWithURLsForTest(cfg config.LLMConfig, mistralKey, grokKey, mistralURL, grokURL string, httpClient *http.Client) *Router {
	return newRouterWithURLs(cfg, mistralKey, grokKey, mistralURL, grokURL, httpClient)
}
