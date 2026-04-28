package providerservice

import (
	"testing"

	observabilityv1 "code-code.internal/go-contract/observability/v1"
)

func TestNormalizeObservabilityInputValuesUsesDeclaredStoredFields(t *testing.T) {
	values, required, err := normalizeObservabilityInputValues(testObservabilityInputForm(), map[string]string{
		"access_token": " token ",
	})
	if err != nil {
		t.Fatalf("normalizeObservabilityInputValues() error = %v", err)
	}
	if got, want := values["access_token"], "token"; got != want {
		t.Fatalf("access_token = %q, want %q", got, want)
	}
	if len(required) != 1 || required[0] != "access_token" {
		t.Fatalf("required = %#v, want access_token", required)
	}
}

func TestNormalizeObservabilityInputValuesRejectsUndeclaredFields(t *testing.T) {
	_, _, err := normalizeObservabilityInputValues(testObservabilityInputForm(), map[string]string{
		"authorization": "Bearer token",
	})
	if err == nil {
		t.Fatal("normalizeObservabilityInputValues() error = nil, want undeclared field error")
	}
}

func TestNormalizeObservabilityInputValuesMergesDeclaredSetCookie(t *testing.T) {
	form := &observabilityv1.ActiveQueryInputForm{
		SchemaId:    "google-ai-studio-session",
		Title:       "Update AI Studio Session",
		ActionLabel: "Update AI Studio Session",
		Fields: []*observabilityv1.ActiveQueryInputField{
			{
				FieldId:     "cookie",
				Label:       "Request Cookie",
				Required:    true,
				Control:     observabilityv1.ActiveQueryInputControl_ACTIVE_QUERY_INPUT_CONTROL_TEXTAREA,
				Persistence: observabilityv1.ActiveQueryInputPersistence_ACTIVE_QUERY_INPUT_PERSISTENCE_STORED_MATERIAL,
			},
			{
				FieldId:       "response_set_cookie",
				Label:         "Response Set-Cookie",
				Control:       observabilityv1.ActiveQueryInputControl_ACTIVE_QUERY_INPUT_CONTROL_TEXTAREA,
				Persistence:   observabilityv1.ActiveQueryInputPersistence_ACTIVE_QUERY_INPUT_PERSISTENCE_TRANSIENT,
				TargetFieldId: "cookie",
				Transform:     observabilityv1.ActiveQueryInputValueTransform_ACTIVE_QUERY_INPUT_VALUE_TRANSFORM_MERGE_SET_COOKIE,
			},
		},
	}

	values, _, err := normalizeObservabilityInputValues(form, map[string]string{
		"cookie":              "SID=old; HSID=old",
		"response_set_cookie": "Set-Cookie: SID=new; Path=/\nHSID=fresh; Path=/",
	})
	if err != nil {
		t.Fatalf("normalizeObservabilityInputValues() error = %v", err)
	}
	if got, want := values["cookie"], "HSID=fresh; SID=new"; got != want {
		t.Fatalf("cookie = %q, want %q", got, want)
	}
}

func TestNormalizeObservabilityInputValuesRequiresMergeTargetInSubmission(t *testing.T) {
	form := &observabilityv1.ActiveQueryInputForm{
		SchemaId:    "google-ai-studio-session",
		Title:       "Update AI Studio Session",
		ActionLabel: "Update AI Studio Session",
		Fields: []*observabilityv1.ActiveQueryInputField{
			{
				FieldId:     "cookie",
				Label:       "Request Cookie",
				Control:     observabilityv1.ActiveQueryInputControl_ACTIVE_QUERY_INPUT_CONTROL_TEXTAREA,
				Persistence: observabilityv1.ActiveQueryInputPersistence_ACTIVE_QUERY_INPUT_PERSISTENCE_STORED_MATERIAL,
			},
			{
				FieldId:       "response_set_cookie",
				Label:         "Response Set-Cookie",
				Control:       observabilityv1.ActiveQueryInputControl_ACTIVE_QUERY_INPUT_CONTROL_TEXTAREA,
				Persistence:   observabilityv1.ActiveQueryInputPersistence_ACTIVE_QUERY_INPUT_PERSISTENCE_TRANSIENT,
				TargetFieldId: "cookie",
				Transform:     observabilityv1.ActiveQueryInputValueTransform_ACTIVE_QUERY_INPUT_VALUE_TRANSFORM_MERGE_SET_COOKIE,
			},
		},
	}

	_, _, err := normalizeObservabilityInputValues(form, map[string]string{
		"response_set_cookie": "Set-Cookie: SID=new; Path=/",
	})
	if err == nil {
		t.Fatal("normalizeObservabilityInputValues() error = nil, want missing merge target error")
	}
}

func testObservabilityInputForm() *observabilityv1.ActiveQueryInputForm {
	return &observabilityv1.ActiveQueryInputForm{
		SchemaId:    "mistral-billing-session",
		Title:       "Update Mistral Session Token",
		ActionLabel: "Update Session Token",
		Fields: []*observabilityv1.ActiveQueryInputField{{
			FieldId:     "access_token",
			Label:       "Session token",
			Required:    true,
			Sensitive:   true,
			Control:     observabilityv1.ActiveQueryInputControl_ACTIVE_QUERY_INPUT_CONTROL_PASSWORD,
			Persistence: observabilityv1.ActiveQueryInputPersistence_ACTIVE_QUERY_INPUT_PERSISTENCE_STORED_MATERIAL,
		}},
	}
}
