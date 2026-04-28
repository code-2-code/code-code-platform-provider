package providerobservability

import (
	"net/http"
	"strings"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
)

func credentialBackfillValuesFromHTTPResponse(rules []CredentialBackfillRule, headers http.Header) map[string]string {
	if len(rules) == 0 || len(headers) == 0 {
		return nil
	}
	values := map[string]string{}
	for _, rule := range rules {
		sourceName := strings.TrimSpace(rule.SourceName)
		if sourceName == "" {
			continue
		}
		switch rule.Source {
		case observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_HTTP_RESPONSE_HEADER:
			value := strings.TrimSpace(headers.Get(sourceName))
			if value != "" {
				values[sourceName] = value
			}
		case observabilityv1.CredentialBackfillSource_CREDENTIAL_BACKFILL_SOURCE_HTTP_RESPONSE_COOKIE:
			if value := responseCookieValue(headers, sourceName); value != "" {
				values[sourceName] = value
			}
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func responseCookieValue(headers http.Header, name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	response := &http.Response{Header: headers}
	var out string
	for _, cookie := range response.Cookies() {
		if cookie == nil || strings.TrimSpace(cookie.Name) != name {
			continue
		}
		if value := strings.TrimSpace(cookie.Value); value != "" {
			out = value
		}
	}
	return out
}

func mergeCredentialBackfillValues(base map[string]string, next map[string]string) map[string]string {
	if len(base) == 0 && len(next) == 0 {
		return nil
	}
	out := map[string]string{}
	for key, value := range base {
		if key = strings.TrimSpace(key); key != "" {
			if value = strings.TrimSpace(value); value != "" {
				out[key] = value
			}
		}
	}
	for key, value := range next {
		if key = strings.TrimSpace(key); key != "" {
			if value = strings.TrimSpace(value); value != "" {
				out[key] = value
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
