// Package live embeds the static browser assets of the live display.
package live

import "embed"

//go:embed index.html app.js lib.js style.css panels
var FS embed.FS
