package providerconnect

import (
	"regexp"
	"strings"

	"code-code.internal/go-contract/domainerror"
)

var surfaceAPIKeyBaseURLTemplateTokenPattern = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

func resolveSurfaceAPIKeyBaseURL(templateBaseURL, providedBaseURL string) (string, bool, error) {
	templateBaseURL = strings.TrimSpace(templateBaseURL)
	providedBaseURL = strings.TrimSpace(providedBaseURL)
	if templateBaseURL == "" {
		return "", false, domainerror.NewValidation("platformk8s/providerconnect: provider surface api endpoint base_url is empty")
	}
	templateFields := surfaceAPIKeyBaseURLTemplateFields(templateBaseURL)
	if len(templateFields) == 0 {
		if providedBaseURL != "" && providedBaseURL != templateBaseURL {
			return "", false, domainerror.NewValidation("platformk8s/providerconnect: provider surface API key connect does not accept base_url override")
		}
		return templateBaseURL, false, nil
	}
	if providedBaseURL == "" {
		return "", false, domainerror.NewValidation(
			"platformk8s/providerconnect: provider surface API key connect requires base_url with fields %q",
			strings.Join(templateFields, ", "),
		)
	}
	matches := surfaceAPIKeyBaseURLTemplateValuePattern(templateBaseURL).FindStringSubmatch(providedBaseURL)
	if len(matches) != len(templateFields)+1 {
		return "", false, domainerror.NewValidation("platformk8s/providerconnect: provider surface API key connect base_url does not match required template")
	}
	for index, name := range templateFields {
		fieldValue := strings.TrimSpace(matches[index+1])
		if fieldValue == "" {
			return "", false, domainerror.NewValidation("platformk8s/providerconnect: provider surface API key connect base_url field %q is empty", name)
		}
		if strings.ContainsAny(fieldValue, "{}") {
			return "", false, domainerror.NewValidation("platformk8s/providerconnect: provider surface API key connect base_url field %q is unresolved", name)
		}
	}
	return providedBaseURL, true, nil
}

func surfaceAPIKeyBaseURLTemplateFields(template string) []string {
	matches := surfaceAPIKeyBaseURLTemplateTokenPattern.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		name := strings.TrimSpace(match[1])
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func surfaceAPIKeyBaseURLTemplateValuePattern(template string) *regexp.Regexp {
	indexes := surfaceAPIKeyBaseURLTemplateTokenPattern.FindAllStringSubmatchIndex(template, -1)
	if len(indexes) == 0 {
		return regexp.MustCompile("^" + regexp.QuoteMeta(template) + "$")
	}
	var pattern strings.Builder
	pattern.WriteString("^")
	last := 0
	for _, match := range indexes {
		pattern.WriteString(regexp.QuoteMeta(template[last:match[0]]))
		pattern.WriteString("([^/?#]+)")
		last = match[1]
	}
	pattern.WriteString(regexp.QuoteMeta(template[last:]))
	pattern.WriteString("$")
	return regexp.MustCompile(pattern.String())
}
