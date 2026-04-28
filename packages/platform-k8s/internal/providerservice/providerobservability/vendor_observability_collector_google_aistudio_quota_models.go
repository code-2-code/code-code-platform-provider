package providerobservability

import "strings"

type googleAIStudioQuotaModelMeta struct {
	ModelID      string
	Group        string
	CategoryCode int
	Preview      bool
}

func parseGoogleAIStudioQuotaModels(payload []any) (map[string]googleAIStudioQuotaModelMeta, error) {
	rows, err := googleAIStudioPayloadRows(payload)
	if err != nil {
		return nil, err
	}
	out := map[string]googleAIStudioQuotaModelMeta{}
	for _, item := range rows {
		row, ok := googleAIStudioPayloadRow(item)
		if !ok {
			continue
		}
		modelID := googleAIStudioStringAt(row, 0)
		if modelID == "" {
			continue
		}
		meta := googleAIStudioQuotaModelMeta{
			ModelID: modelID,
			Group:   googleAIStudioStringAt(row, 3),
		}
		meta.CategoryCode, _ = googleAIStudioIntAt(row, 2)
		previewTokens := []string{modelID, googleAIStudioStringAt(row, 5)}
		if aliases, ok := rowValueSlice(row, 4); ok {
			for _, aliasItem := range aliases {
				aliasRow, ok := googleAIStudioPayloadRow(aliasItem)
				if !ok {
					continue
				}
				aliasModelID := googleAIStudioStringAt(aliasRow, 0)
				previewTokens = append(previewTokens, strings.TrimPrefix(aliasModelID, "models/"))
			}
		}
		meta.Preview = googleAIStudioPreviewModel(previewTokens...)
		out[modelID] = meta
	}
	if len(out) == 0 {
		return nil, errGoogleAIStudioNoRows("ListQuotaModels")
	}
	return out, nil
}

func rowValueSlice(row []any, index int) ([]any, bool) {
	if index < 0 || index >= len(row) {
		return nil, false
	}
	values, ok := row[index].([]any)
	return values, ok
}

func googleAIStudioPreviewModel(values ...string) bool {
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if strings.Contains(normalized, "preview") ||
			strings.Contains(normalized, "-exp") ||
			strings.Contains(normalized, "_exp") ||
			strings.Contains(normalized, "experimental") {
			return true
		}
	}
	return false
}
