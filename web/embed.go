// Package web holds the dashboard: plain HTML, CSS and JavaScript with no build step and no external
// requests. internal/api serves it from this embedded filesystem.
package web

import "embed"

//go:embed index.html app.js panels.js style.css setup.html setup.js
var FS embed.FS
