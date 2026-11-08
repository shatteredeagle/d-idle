package access

import (
	"net/http"
	"net/url"
	"strings"
)

// SameOrigin decides whether a state-changing request comes from a page of this server. The rule (docs/watch-origin.md):
//  1. Sec-Fetch-Site, when the browser sends it, is decisive: "same-origin" and "none" pass, anything else fails.
//  2. Otherwise a real Origin must match the request host (case-insensitive, default port 80/443 equals no port).
//  3. Origin "null" (WebKit after a no-referrer form POST) is judged by the Referer instead: same host passes.
//  4. No Origin: a Referer, if present, must match; with neither header (curl, native clients) the request passes,
//     because it cannot be a browser CSRF (a browser always sends Origin or Sec-Fetch-Site on a cross-site POST).
func SameOrigin(r *http.Request) bool {
	if s := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site"))); s != "" {
		return s == "same-origin" || s == "none"
	}
	o := strings.TrimSpace(r.Header.Get("Origin"))
	if o != "" && o != "null" {
		return hostMatches(o, r.Host)
	}
	if ref := r.Header.Get("Referer"); ref != "" {
		return hostMatches(ref, r.Host)
	}
	return o == ""
}

func hostMatches(raw, reqHost string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	return normHost(u.Host, u.Scheme) == normHost(reqHost, u.Scheme)
}

// normHost lowercases and appends the scheme's default port when none is given.
func normHost(h, scheme string) string {
	h = strings.ToLower(h)
	i := strings.LastIndex(h, ":")
	if i >= 0 && !strings.HasSuffix(h, "]") {
		return h
	}
	if scheme == "https" {
		return h + ":443"
	}
	return h + ":80"
}
