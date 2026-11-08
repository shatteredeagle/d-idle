package access

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	const host = "play.example.test:8891"
	cases := []struct {
		name string
		h    map[string]string
		host string
		want bool
	}{
		{"no headers (curl)", nil, host, true},
		{"origin same", map[string]string{"Origin": "http://play.example.test:8891"}, host, true},
		{"origin same, case", map[string]string{"Origin": "http://PLAY.example.test:8891"}, host, true},
		{"origin foreign", map[string]string{"Origin": "http://evil.example"}, host, false},
		{"origin other port", map[string]string{"Origin": "http://play.example.test:9"}, host, false},
		{"origin null no referer", map[string]string{"Origin": "null"}, host, false},
		{"origin null same referer", map[string]string{"Origin": "null", "Referer": "http://play.example.test:8891/login"}, host, true},
		{"origin null foreign referer", map[string]string{"Origin": "null", "Referer": "http://evil.example/"}, host, false},
		{"no origin foreign referer", map[string]string{"Referer": "http://evil.example/"}, host, false},
		{"no origin same referer", map[string]string{"Referer": "http://play.example.test:8891/"}, host, true},
		{"sfs same-origin + null", map[string]string{"Origin": "null", "Sec-Fetch-Site": "same-origin"}, host, true},
		{"firefox: Origin null + Sec-Fetch-Site same-origin, no Referer", map[string]string{"Origin": "null", "Sec-Fetch-Site": "same-origin"}, host, true},
		{"sfs none", map[string]string{"Sec-Fetch-Site": "none"}, host, true},
		{"sfs cross-site", map[string]string{"Sec-Fetch-Site": "cross-site"}, host, false},
		{"sfs same-site", map[string]string{"Sec-Fetch-Site": "same-site"}, host, false},
		{"sfs cross-site beats same origin", map[string]string{"Origin": "http://play.example.test:8891", "Sec-Fetch-Site": "cross-site"}, host, false},
		{"default port 80 implicit", map[string]string{"Origin": "http://box.lan"}, "box.lan:80", true},
		{"default port 80 explicit", map[string]string{"Origin": "http://box.lan:80"}, "box.lan", true},
		{"https 443", map[string]string{"Origin": "https://box.lan"}, "box.lan:443", true},
		{"ipv6", map[string]string{"Origin": "http://[::1]:8891"}, "[::1]:8891", true},
		{"ipv6 default port", map[string]string{"Origin": "http://[::1]"}, "[::1]", true},
		{"garbage origin", map[string]string{"Origin": "::::"}, host, false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(http.MethodPost, "/login", nil)
		r.Host = c.host
		for k, v := range c.h {
			r.Header.Set(k, v)
		}
		if got := SameOrigin(r); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
