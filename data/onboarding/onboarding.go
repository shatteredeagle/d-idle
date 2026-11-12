// Package onboardingdata embeds the friend-facing onboarding texts (data/onboarding/en.json): the welcome card, the
// first-30-minutes checklist, the reconnect banners. The code is internal/watch/onboarding.go.
package onboardingdata

import _ "embed"

// JSON is en.json.
//
//go:embed en.json
var JSON []byte
