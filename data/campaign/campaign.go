// Package campaigndata embeds the campaign spines (data/campaign/*.json). The engine is internal/campaign.
package campaigndata

import _ "embed"

// NormalJSON is normal.json, the Act 1 Normal spine (docs/campaign.md).
//
//go:embed normal.json
var NormalJSON []byte
