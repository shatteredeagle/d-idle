package watch

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

// Operator notices on the viewer: standing warnings the owner must not miss (a character file restored from a
// backup, low disk space). Read only; the lines come from the wiring (ServeOptions.Notices).

// NoticeRoutes are the GET routes of the notice banner.
var NoticeRoutes = []string{"/api/notices", "/watch/notices.js"}

// noticesJS polls the notices and shows them in a fixed bar at the top; no bar when there are none.
const noticesJS = `(function () {
  'use strict';
  var bar = null;
  function show(lines) {
    if (!bar) {
      bar = document.createElement('div');
      bar.id = 'notice-banner';
      bar.setAttribute('role', 'alert');
      bar.style.cssText = 'position:fixed;left:0;right:0;top:0;z-index:60;padding:8px 12px;background:#7a1f1f;color:#fff;font:14px sans-serif;white-space:pre-line';
      document.body.appendChild(bar);
    }
    bar.textContent = lines.join('\n');
    bar.style.display = lines.length ? 'block' : 'none';
  }
  function refresh() {
    fetch('/api/notices', { credentials: 'include' }).then(function (r) { return r.json(); })
      .then(function (d) { show((d && d.notices) || []); }).catch(function () {});
  }
  refresh();
  setInterval(refresh, 60000);
})();
`

// NoticesHandler serves NoticeRoutes.
func NoticesHandler(notices func() []string) http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("/api/notices", func(w http.ResponseWriter, r *http.Request) {
		n := notices()
		if n == nil {
			n = []string{}
		}
		writeJSON(w, 200, map[string][]string{"notices": n})
	})
	m.HandleFunc("/watch/notices.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write([]byte(noticesJS))
	})
	return m
}

// withScriptTag adds tag before </body> of the screens' HTML without touching the pages themselves.
func withScriptTag(next http.Handler, tag string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/watch") || strings.Contains(r.URL.Path[1:], ".") {
			next.ServeHTTP(w, r)
			return
		}
		rec := &bufWriter{h: http.Header{}, code: 200}
		next.ServeHTTP(rec, r)
		body := rec.buf.Bytes()
		if strings.HasPrefix(rec.h.Get("Content-Type"), "text/html") && bytes.Contains(body, []byte("</body>")) {
			body = bytes.Replace(body, []byte("</body>"), []byte(tag+"</body>"), 1)
		}
		for k, v := range rec.h {
			w.Header()[k] = v
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(rec.code)
		w.Write(body)
	})
}
