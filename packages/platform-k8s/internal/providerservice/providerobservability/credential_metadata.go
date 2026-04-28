package providerobservability

import (
	"time"
)

const (
	materialKeyAccountID = "account_id"
	materialKeyProjectID = "project_id"
	materialKeyTierName  = "tier_name"
)

func timePointerCopy(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}
