package elastic

import (
	"github.com/elastic/opentelemetry-lib/enrichments/config"
	"github.com/elastic/opentelemetry-lib/enrichments/internal/elastic"
	"github.com/ua-parser/uap-go/uaparser"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// EnrichSpan adds Elastic specific attributes to the OTel span.
// These attributes are derived from the base attributes and appended to
// the span attributes. The enrichment logic is performed by categorizing
// the OTel spans into 2 different categories:
//   - Elastic transactions, defined as spans which measure the highest
//     level of work being performed with a service.
//   - Elastic spans, defined as all spans (including transactions).
//     However, for the enrichment logic spans are treated as a separate
//     entity i.e. all transactions are not enriched as spans and vice versa.
func EnrichSpan(
	span ptrace.Span,
	cfg config.Config,
	userAgentParser *uaparser.Parser,
) {
	elastic.EnrichSpan(span, cfg, userAgentParser)
}
