package watch

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	alertsdata "d2lod-bot/data/alerts"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
)

// Alerts and pings (TASK-162, docs/unlock-alerts.md). A rules table (data/alerts/rules.json) turns events the watch
// screen already sees into short alerts in the hero's voice. The in-page channel is GET /api/alerts plus a toast
// centre (web/alerts.js). The outbound channel is the Sender interface; it is off unless the profile config switches it
// on, and nothing in the repo holds a URL. Death and low life are free (rules with free=true): basic safety info is
// never behind the ladder row "alerts".

const (
	// AlertsRoute is GET (viewer); AlertsUIRoute serves web/alerts.js. AlertsID is the ladder row.
	AlertsRoute   = "/api/alerts"
	AlertsUIRoute = "/watch/alerts.js"
	AlertsID      = "alerts"

	alertsSchema      = "watch.alerts/1"
	alertsKeep        = 100
	alertsSendTimeout = 5 * time.Second
	alertsMaxInflight = 4
)

// Alert event kinds the code can raise. The table must use only these.
var alertEvents = map[string][]string{
	"drop":       {"item", "quality", "area", "hero"},
	"death":      {"monster", "area", "level", "hero"},
	"low_life":   {"pct", "area", "hero"},
	"kill":       {"monster", "area", "hero"},
	"level_up":   {"level", "hero"},
	"town":       {"area", "hero"},
	"pact_pause": {"reason", "hero"},
	"proposal":   {"from", "to", "hero"},
}

var alertSeverities = []string{"info", "good", "warn", "alarm"}

// alertClasses are the class keys classText may use (the seven classes of the game).
var alertClasses = []string{"amazon", "sorceress", "necromancer", "paladin", "barbarian", "druid", "assassin"}

// AlertFilters narrows a rule. Only the filters that make sense for the event may be set.
type AlertFilters struct {
	Qualities []string `json:"qualities,omitempty"` // drop: any of unique, set, rare, crafted, magic, rune
	Boss      bool     `json:"boss,omitempty"`      // kill: only boss kills
	MinLevel  int      `json:"minLevel,omitempty"`  // level_up
	BelowPct  int      `json:"belowPct,omitempty"`  // low_life: life percent under this
}

// AlertRule is one row of data/alerts/rules.json.
type AlertRule struct {
	ID           string            `json:"id"`
	Event        string            `json:"event"`
	Filters      AlertFilters      `json:"filters,omitempty"`
	CooldownSecs int               `json:"cooldownSecs"`
	Severity     string            `json:"severity"`
	Free         bool              `json:"free"`
	Text         string            `json:"text"`
	ClassText    map[string]string `json:"classText,omitempty"`
}

// AlertRules is the parsed table.
type AlertRules struct {
	Schema string      `json:"schema"`
	Doc    string      `json:"doc"`
	Note   string      `json:"note"`
	Rules  []AlertRule `json:"rules"`
}

// ParseAlertRules checks the table. Every error names the rule: an unknown event, severity, class or placeholder, a
// filter that does not fit the event, a negative cooldown, a duplicate id, an empty text.
func ParseAlertRules(b []byte) (*AlertRules, error) {
	var t AlertRules
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("alert rules: %w", err)
	}
	if t.Schema != "alerts.rules/1" {
		return nil, fmt.Errorf("alert rules: schema %q", t.Schema)
	}
	if len(t.Rules) == 0 {
		return nil, errors.New("alert rules: no rules")
	}
	seen := map[string]bool{}
	for _, r := range t.Rules {
		fail := func(f string, a ...any) error {
			return fmt.Errorf("alert rules: rule %q: %s", r.ID, fmt.Sprintf(f, a...))
		}
		allowed, ok := alertEvents[r.Event]
		switch {
		case r.ID == "":
			return nil, errors.New("alert rules: a rule has no id")
		case seen[r.ID]:
			return nil, fail("duplicate id")
		case !ok:
			return nil, fail("unknown event %q", r.Event)
		case !containsStr(alertSeverities, r.Severity):
			return nil, fail("unknown severity %q", r.Severity)
		case r.CooldownSecs < 0:
			return nil, fail("negative cooldown")
		case strings.TrimSpace(r.Text) == "":
			return nil, fail("empty text")
		}
		seen[r.ID] = true
		f := r.Filters
		switch {
		case len(f.Qualities) > 0 && r.Event != "drop":
			return nil, fail("qualities only fit a drop")
		case r.Event == "drop" && len(f.Qualities) == 0:
			return nil, fail("a drop rule needs qualities")
		case f.Boss && r.Event != "kill":
			return nil, fail("boss only fits a kill")
		case r.Event == "kill" && !f.Boss:
			return nil, fail("a kill rule must set boss (every kill would alert)")
		case f.MinLevel != 0 && r.Event != "level_up":
			return nil, fail("minLevel only fits level_up")
		case f.BelowPct != 0 && r.Event != "low_life":
			return nil, fail("belowPct only fits low_life")
		case r.Event == "low_life" && (f.BelowPct < 1 || f.BelowPct > 99):
			return nil, fail("belowPct %d must be 1..99", f.BelowPct)
		}
		if err := checkAlertText(r.Text, allowed); err != nil {
			return nil, fail("%v", err)
		}
		for c, txt := range r.ClassText {
			if !containsStr(alertClasses, c) {
				return nil, fail("classText for unknown class %q", c)
			}
			if err := checkAlertText(txt, allowed); err != nil {
				return nil, fail("class %s: %v", c, err)
			}
		}
	}
	return &t, nil
}

func checkAlertText(s string, allowed []string) error {
	for _, m := range voicePlaceholder.FindAllString(s, -1) {
		if !containsStr(allowed, strings.Trim(m, "{}")) {
			return fmt.Errorf("placeholder %s is not available for this event", m)
		}
	}
	if strings.ContainsAny(s, "<>") {
		return errors.New("text may not hold < or >")
	}
	return nil
}

// ShippedAlertRules parses the embedded table.
func ShippedAlertRules() (*AlertRules, error) { return ParseAlertRules(alertsdata.JSON) }

// AlertEvent is what the hooks hand to Emit.
type AlertEvent struct {
	Kind    string // a key of alertEvents
	Item    string
	Quality string
	Monster string
	Area    string
	Level   int
	Pct     int
	Boss    bool
	Reason  string
	From    string
	To      string
}

// Alert is one raised alert, as the API and the senders see it.
type Alert struct {
	Seq      uint64  `json:"seq"`
	T        float64 `json:"t"`
	Rule     string  `json:"rule"`
	Kind     string  `json:"kind"`
	Severity string  `json:"severity"`
	Free     bool    `json:"free"`
	Text     string  `json:"text"`
}

// Sender is an outbound channel. Send makes one attempt (no retry) and returns its error. Implementations are not used
// unless OutboundConfig switches them on.
type Sender interface {
	Name() string
	Send(ctx context.Context, a Alert) error
}

// OutboundConfig is the profile's outbound setting. The zero value is off. Enabling it is a deliberate edit of the
// profile config file; no route changes it.
type OutboundConfig struct {
	Enabled     bool   `json:"enabled"`
	Kind        string `json:"kind"`                  // "webhook" or "ntfy"
	URL         string `json:"url"`                   // empty: off
	Secret      string `json:"secret,omitempty"`      // webhook only: HMAC-SHA256 key for X-D2-Signature
	MinSeverity string `json:"minSeverity,omitempty"` // default "warn"
}

// ParseOutboundConfig reads the profile's outbound setting. Empty input is the off setting.
func ParseOutboundConfig(b []byte) (OutboundConfig, error) {
	var c OutboundConfig
	if len(bytes.TrimSpace(b)) == 0 {
		return c, nil
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return OutboundConfig{}, fmt.Errorf("alerts outbound config: %w", err)
	}
	return c, c.Check()
}

// Check validates a setting that is on; an off setting is always valid.
func (c OutboundConfig) Check() error {
	if !c.Enabled {
		return nil
	}
	if c.URL == "" {
		return errors.New("alerts outbound: enabled but no url")
	}
	u, err := url.Parse(c.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return errors.New("alerts outbound: url must be http or https with a host")
	}
	switch c.Kind {
	case "webhook", "ntfy":
	default:
		return fmt.Errorf("alerts outbound: unknown kind %q (want webhook or ntfy)", c.Kind)
	}
	if c.Secret != "" && c.Kind != "webhook" {
		return fmt.Errorf("alerts outbound: a secret only fits kind webhook, not %q", c.Kind)
	}
	if c.MinSeverity != "" && !containsStr(alertSeverities, c.MinSeverity) {
		return fmt.Errorf("alerts outbound: unknown minSeverity %q", c.MinSeverity)
	}
	return nil
}

// Target is the URL as the page may show it: no user info, no query, no fragment.
func (c OutboundConfig) Target() string {
	u, err := url.Parse(c.URL)
	if err != nil || c.URL == "" {
		return ""
	}
	u.User, u.RawQuery, u.Fragment = nil, "", ""
	return u.String()
}

// Consent is the line the settings show. When the channel is on it names the place the alert text is sent to.
func (c OutboundConfig) Consent() string {
	if !c.Enabled || c.URL == "" {
		return "Outbound alerts are off. Nothing leaves this machine."
	}
	return "Sends data to " + c.Target() + ": the text of each alert at or above severity " + c.minSeverity() + ". Switch off in the profile config."
}

func (c OutboundConfig) minSeverity() string {
	if c.MinSeverity == "" {
		return "warn"
	}
	return c.MinSeverity
}

// NewSender builds the sender of an enabled setting; nil, nil when the setting is off. client nil: a client with a
// 5 s timeout that does not follow redirects.
func NewSender(c OutboundConfig, client *http.Client) (Sender, error) {
	if !c.Enabled {
		return nil, nil
	}
	if err := c.Check(); err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: alertsSendTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	}
	if c.Kind == "ntfy" {
		return &NtfySender{URL: c.URL, Client: client}, nil
	}
	return &WebhookSender{URL: c.URL, Secret: c.Secret, Client: client}, nil
}

func post(ctx context.Context, client *http.Client, u string, body []byte, hdr map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("alert send: HTTP %d", resp.StatusCode)
	}
	return nil
}

// WebhookSender posts the alert as JSON. With a Secret the body is signed: header X-D2-Signature is
// "sha256=" and the hex HMAC-SHA256 of the exact body bytes.
type WebhookSender struct {
	URL    string
	Secret string
	Client *http.Client
}

func (*WebhookSender) Name() string { return "webhook" }

// WebhookPayload is the body a webhook receives.
type WebhookPayload struct {
	Schema   string  `json:"schema"`
	Seq      uint64  `json:"seq"`
	T        float64 `json:"t"`
	Rule     string  `json:"rule"`
	Kind     string  `json:"kind"`
	Severity string  `json:"severity"`
	Text     string  `json:"text"`
}

// Sign is the signature header value for body.
func Sign(secret string, body []byte) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write(body)
	return "sha256=" + hex.EncodeToString(m.Sum(nil))
}

func (w *WebhookSender) Send(ctx context.Context, a Alert) error {
	body, err := json.Marshal(WebhookPayload{Schema: "watch.alert/1", Seq: a.Seq, T: a.T, Rule: a.Rule, Kind: a.Kind, Severity: a.Severity, Text: a.Text})
	if err != nil {
		return err
	}
	h := map[string]string{"Content-Type": "application/json"}
	if w.Secret != "" {
		h["X-D2-Signature"] = Sign(w.Secret, body)
	}
	return post(ctx, w.Client, w.URL, body, h)
}

// NtfySender posts the alert text as the body of an ntfy-style topic URL (plain text, Title, Priority and Tags headers).
type NtfySender struct {
	URL    string
	Client *http.Client
}

func (*NtfySender) Name() string { return "ntfy" }

var ntfyPriority = map[string]string{"info": "2", "good": "3", "warn": "4", "alarm": "5"}

func (n *NtfySender) Send(ctx context.Context, a Alert) error {
	return post(ctx, n.Client, n.URL, []byte(a.Text), map[string]string{
		"Content-Type": "text/plain; charset=utf-8", "Title": "d.idle", "Priority": ntfyPriority[a.Severity], "Tags": a.Rule})
}

// AlertsConfig turns the alerts on in FunConfig.
type AlertsConfig struct {
	Rules    *AlertRules // nil: the shipped table
	Outbound OutboundConfig
	Client   *http.Client // for the sender (tests pass an httptest client); nil: the default
	Sender   Sender       // replaces the one built from Outbound (tests); still only used when Outbound.Enabled
}

// Alerts raises, keeps and (when switched on) sends the alerts. Safe for concurrent use.
type Alerts struct {
	u      *Unlocks // nil: only free rules fire
	rules  *AlertRules
	now    func() float64
	class  func() string // lowercase class of the hero, "" while unknown
	out    OutboundConfig
	sender Sender

	proposals func() []decide.Proposal
	incidents func() []JournalLine

	mu       sync.Mutex
	seq      uint64
	evSeq    uint64
	ring     []Alert
	last     map[string]float64
	lastArea int
	seenProp map[int]bool
	seenInc  map[string]bool
	errs     []string
	wg       sync.WaitGroup
	slots    chan struct{}
}

// NewAlerts builds Alerts. u may be nil (no ladder). A ladder that lacks the row "alerts" is an error naming it.
func NewAlerts(u *Unlocks, c AlertsConfig, now func() float64, class func() string) (*Alerts, error) {
	if u != nil {
		if _, err := u.Engine.Check(AlertsID); err != nil {
			return nil, err
		}
	}
	rules := c.Rules
	if rules == nil {
		var err error
		if rules, err = ShippedAlertRules(); err != nil {
			return nil, err
		}
	}
	if err := c.Outbound.Check(); err != nil {
		return nil, err
	}
	a := &Alerts{u: u, rules: rules, now: now, class: class, out: c.Outbound, last: map[string]float64{},
		seenProp: map[int]bool{}, seenInc: map[string]bool{}, slots: make(chan struct{}, alertsMaxInflight)}
	if c.Outbound.Enabled {
		a.sender = c.Sender
		if a.sender == nil {
			s, err := NewSender(c.Outbound, c.Client)
			if err != nil {
				return nil, err
			}
			a.sender = s
		}
	}
	return a, nil
}

// Open says whether the gated rules fire (the ladder row is open).
func (a *Alerts) Open() bool { return a.u != nil && a.u.Engine.Allowed(AlertsID) }

// Errors lists send failures, oldest first.
func (a *Alerts) Errors() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.errs...)
}

// Flush waits for the sends in flight (tests).
func (a *Alerts) Flush() { a.wg.Wait() }

func alertQuality(it goals.Item) string {
	if strings.EqualFold(it.Type, "rune") || strings.HasSuffix(it.Name, " Rune") {
		return "rune"
	}
	return strings.ToLower(it.Quality)
}

func (r AlertRule) matches(e AlertEvent) bool {
	if r.Event != e.Kind {
		return false
	}
	switch e.Kind {
	case "drop":
		return containsStr(r.Filters.Qualities, strings.ToLower(e.Quality))
	case "kill":
		return !r.Filters.Boss || e.Boss
	case "level_up":
		return e.Level >= r.Filters.MinLevel
	case "low_life":
		return e.Pct < r.Filters.BelowPct
	}
	return true
}

func (a *Alerts) render(r AlertRule, e AlertEvent) string {
	t := r.Text
	cls := ""
	if a.class != nil {
		cls = strings.ToLower(a.class())
	}
	if ct, ok := r.ClassText[cls]; ok {
		t = ct
	}
	hero := cls
	if hero == "" {
		hero = "the hero"
	}
	rep := strings.NewReplacer("{hero}", hero, "{item}", e.Item, "{quality}", e.Quality, "{monster}", e.Monster, "{area}", e.Area,
		"{level}", fmt.Sprint(e.Level), "{pct}", fmt.Sprint(e.Pct), "{reason}", e.Reason, "{from}", e.From, "{to}", e.To)
	return safeText(rep.Replace(t), 200)
}

// Emit raises every rule that matches. Rules that are not free are skipped while the ladder row is locked (nothing is
// stored for them, so buying later does not flood the centre). Returns the alerts raised.
func (a *Alerts) Emit(e AlertEvent) []Alert {
	open := a.Open()
	t := a.now()
	var out []Alert
	a.mu.Lock()
	for _, r := range a.rules.Rules {
		if !r.matches(e) || (!r.Free && !open) {
			continue
		}
		if l, ok := a.last[r.ID]; ok && r.CooldownSecs > 0 && t-l < float64(r.CooldownSecs) {
			continue
		}
		a.last[r.ID] = t
		a.seq++
		al := Alert{Seq: a.seq, T: t, Rule: r.ID, Kind: r.Event, Severity: r.Severity, Free: r.Free, Text: a.render(r, e)}
		a.ring = append(a.ring, al)
		if len(a.ring) > alertsKeep {
			a.ring = append([]Alert(nil), a.ring[len(a.ring)-alertsKeep:]...)
		}
		out = append(out, al)
	}
	a.mu.Unlock()
	for _, al := range out {
		a.send(al)
	}
	return out
}

func severityRank(s string) int {
	for i, v := range alertSeverities {
		if v == s {
			return i
		}
	}
	return -1
}

func (a *Alerts) send(al Alert) {
	if a.sender == nil || severityRank(al.Severity) < severityRank(a.out.minSeverity()) {
		return
	}
	select {
	case a.slots <- struct{}{}:
	default:
		a.fail(errors.New("alert send: too many in flight, one dropped"))
		return
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer func() { <-a.slots }()
		ctx, cancel := context.WithTimeout(context.Background(), alertsSendTimeout)
		defer cancel()
		if err := a.sender.Send(ctx, al); err != nil {
			a.fail(fmt.Errorf("%s: %w", a.sender.Name(), err))
		}
	}()
}

func (a *Alerts) fail(err error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.errs) < 20 {
		a.errs = append(a.errs, err.Error())
	}
}

// OnGoalEvent derives alerts from the goals stream: a drop, a death, a level up, a boss kill and the hero reaching a
// town. A replayed stream (Seq not above the highest seen) raises nothing.
func (a *Alerts) OnGoalEvent(ev goals.Event) {
	a.mu.Lock()
	if ev.Seq != 0 {
		if ev.Seq <= a.evSeq {
			a.mu.Unlock()
			return
		}
		a.evSeq = ev.Seq
	}
	prev := a.lastArea
	if ev.Type == goals.EvAreaChange {
		a.lastArea = ev.AreaID
	}
	a.mu.Unlock()
	switch ev.Type {
	case goals.EvDrop:
		if ev.Item != nil {
			a.Emit(AlertEvent{Kind: "drop", Item: safeText(ev.Item.Name, 80), Quality: alertQuality(*ev.Item), Area: ev.Area})
		}
	case goals.EvDeath:
		a.Emit(AlertEvent{Kind: "death", Monster: monsterOr(ev.Monster), Area: areaOr(ev.Area), Level: ev.Level})
	case goals.EvLevelUp:
		a.Emit(AlertEvent{Kind: "level_up", Level: ev.Level})
	case goals.EvKill:
		a.Emit(AlertEvent{Kind: "kill", Monster: safeText(ev.Monster, 60), Boss: ev.Boss})
	case goals.EvAreaChange:
		// only a move from a known field into a town: the first area seen at start is not a return
		if data.IsTown(ev.AreaID) && prev != 0 && !data.IsTown(prev) {
			a.Emit(AlertEvent{Kind: "town", Area: areaOr(ev.Area)})
		}
	}
}

func monsterOr(s string) string {
	if s = safeText(s, 60); s != "" {
		return s
	}
	return "something unseen"
}

func areaOr(s string) string {
	if s = safeText(s, 60); s != "" {
		return s
	}
	return "the wilds"
}

// OnLife is the hook for the hero's life percent (0..100). U: no emitter feeds it yet; the status poll that learns the
// life should call it. The rule's cooldown keeps a long low stretch from repeating.
func (a *Alerts) OnLife(pct int, area string) {
	a.Emit(AlertEvent{Kind: "low_life", Pct: pct, Area: areaOr(area)})
}

// Tick looks for the two signals that have no event: a new pending proposal and a Pact pause. Call it from Settle.
func (a *Alerts) Tick() {
	if a.proposals != nil {
		for _, p := range a.proposals() {
			a.mu.Lock()
			seen := a.seenProp[p.ID]
			a.seenProp[p.ID] = true
			a.mu.Unlock()
			if !seen {
				a.Emit(AlertEvent{Kind: "proposal", From: safeText(p.From, 60), To: safeText(p.To, 60)})
			}
		}
	}
	if a.incidents != nil {
		for _, l := range a.incidents() {
			reason, ok := strings.CutPrefix(l.Text, "Pact: ")
			if !ok || strings.Contains(strings.ToLower(reason), "resum") {
				continue
			}
			a.mu.Lock()
			seen := a.seenInc[l.ID]
			a.seenInc[l.ID] = true
			a.mu.Unlock()
			if !seen {
				a.Emit(AlertEvent{Kind: "pact_pause", Reason: safeText(reason, 120)})
			}
		}
	}
}

// AlertsView is GET /api/alerts. A locked row still carries the free alerts; the gated ones are not raised at all.
type AlertsView struct {
	Schema        string       `json:"schema"`
	Open          bool         `json:"open"`
	Title         string       `json:"title,omitempty"`
	Teaser        string       `json:"teaser,omitempty"`
	Price         int64        `json:"price,omitempty"`
	Seq           uint64       `json:"seq"`
	Alerts        []Alert      `json:"alerts"`
	Outbound      OutboundView `json:"outbound"`
	Notifications NotifyView   `json:"notifications"`
}

// OutboundView is the settings block. It never carries the URL's query, user info or the secret.
type OutboundView struct {
	Enabled bool   `json:"enabled"`
	Kind    string `json:"kind,omitempty"`
	Target  string `json:"target,omitempty"`
	Consent string `json:"consent"`
}

// NotifyView says what the browser channel can and cannot do.
type NotifyView struct {
	Browser    string `json:"browser"`
	Background string `json:"background"`
}

// View returns the alerts with Seq above since.
func (a *Alerts) View(since uint64) AlertsView {
	v := AlertsView{Schema: alertsSchema, Alerts: []Alert{}, Open: a.Open(),
		Outbound: OutboundView{Enabled: a.sender != nil, Consent: a.out.Consent()},
		Notifications: NotifyView{
			Browser:    "Desktop notifications work while this tab is open; the browser asks for permission only when you press the button.",
			Background: "Background push (tab closed) needs a service worker and web push. It is not built.",
		}}
	if a.sender != nil {
		v.Outbound.Kind, v.Outbound.Target = a.out.Kind, a.out.Target()
	}
	if a.u != nil {
		if st, ok := a.u.state(AlertsID); ok {
			v.Title = st.Title
			if !st.Open {
				v.Teaser, v.Price = st.Teaser, st.Price
			}
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	v.Seq = a.seq
	for _, al := range a.ring {
		if al.Seq > since {
			v.Alerts = append(v.Alerts, al)
		}
	}
	return v
}

// Handler serves GET /api/alerts?since=N (viewer).
func (a *Alerts) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		var since uint64
		if s := r.URL.Query().Get("since"); s != "" {
			if _, err := fmt.Sscanf(s, "%d", &since); err != nil {
				http.Error(w, "since must be a number", http.StatusBadRequest)
				return
			}
		}
		writeJSON(w, 200, a.View(since))
	})
}

func alertsUIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile("web/alerts.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", pageCSP)
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})
}
