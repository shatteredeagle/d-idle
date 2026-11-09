package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Pages are the watch pages the launcher can open directly (--page NAME, .desktop actions, shell buttons).
var Pages = map[string]string{
	"watch":     "/watch",
	"inventory": "/watch/inventory",
	"policy":    "/watch/policy",
	"return":    "/watch/return",
}

// NormalizeAddress turns what the user typed ("host:8891", "http://host:8891/watch") into the origin
// scheme://host[:port] and nothing else. The login's same-origin rule compares the page's Origin with
// D2BOT_WATCH_ORIGIN, so the app must load the page through the exact origin the user configured: the host is only
// lowercased (browsers do the same), a port is never added or dropped. Credentials in the address are refused so a
// passphrase can never end up in the config file.
func NormalizeAddress(in string) (string, error) {
	s := strings.TrimSpace(in)
	if s == "" {
		return "", errors.New("the address is empty")
	}
	if !strings.Contains(s, "://") {
		s = "http://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("not a valid address: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("only http and https addresses work, not %q", u.Scheme)
	}
	if u.User != nil {
		return "", errors.New("do not put a user name or passphrase in the address")
	}
	if u.Hostname() == "" {
		return "", errors.New("the address has no host")
	}
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
			return "", fmt.Errorf("bad port %q", p)
		}
	}
	return u.Scheme + "://" + strings.ToLower(u.Host), nil
}

// PageURL joins an origin and a page name; an unknown name is an error, not a silent /watch.
func PageURL(origin, page string) (string, error) {
	p, ok := Pages[page]
	if !ok {
		return "", fmt.Errorf("unknown page %q (watch, inventory, policy, return)", page)
	}
	return origin + p, nil
}
