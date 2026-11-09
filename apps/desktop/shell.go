package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// shell is the local helper page shown instead of the game: the first-run address prompt and the "server is
// warming up" screen. It only listens on 127.0.0.1 behind a random token and checks the Host header (DNS
// rebinding). Once the server answers, the page navigates the window to the real, exactly configured origin; the
// game page itself is never proxied or framed, so the watch login sees the right Origin.
type shell struct {
	dir    string
	page   string
	token  string
	addr   string
	client *http.Client

	mu       sync.Mutex
	cfg      Config
	lastPoll time.Time
}

func newShell(dir string, cfg Config, page string) (*shell, net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, err
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		ln.Close()
		return nil, nil, err
	}
	s := &shell{dir: dir, page: page, token: hex.EncodeToString(b), addr: ln.Addr().String(), client: NewProbeClient(), cfg: cfg, lastPoll: time.Now()}
	return s, ln, nil
}

func (s *shell) URL() string { return "http://" + s.addr + "/t/" + s.token + "/" }

func (s *shell) Handler() http.Handler {
	base := "/t/" + s.token + "/"
	m := http.NewServeMux()
	m.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != base {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'")
		w.Write([]byte(shellHTML))
	})
	m.HandleFunc(base+"status", func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		cfg := s.cfg
		s.lastPoll = time.Now()
		s.mu.Unlock()
		out := map[string]string{"address": cfg.Address}
		if cfg.Address != "" {
			out["state"] = string(Probe(r.Context(), s.client, cfg.Address))
			if u, err := PageURL(cfg.Address, s.page); err == nil {
				out["target"] = u
			}
		}
		json.NewEncoder(w).Encode(out)
	})
	m.HandleFunc(base+"save", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		addr, err := NormalizeAddress(r.FormValue("address"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		s.mu.Lock()
		s.cfg.Address = addr
		cfg := s.cfg
		s.mu.Unlock()
		if err := SaveConfig(s.dir, cfg); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"address": addr})
	})
	m.HandleFunc(base+"change", func(w http.ResponseWriter, r *http.Request) { // forget the address, back to the prompt
		s.mu.Lock()
		s.cfg.Address = ""
		s.mu.Unlock()
		w.Write([]byte("{}"))
	})
	return s.guard(m)
}

// guard rejects any request whose Host is not exactly the loopback listener (DNS rebinding) and any non-GET from
// another origin.
func (s *shell) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != s.addr {
			http.Error(w, "forbidden host", http.StatusForbidden)
			return
		}
		if o := r.Header.Get("Origin"); o != "" && o != "http://"+s.addr {
			http.Error(w, "forbidden origin", http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost && !strings.HasPrefix(r.URL.Path, "/t/"+s.token+"/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Serve runs until the window stops polling (closed, or navigated to the game) for idle, or ctx ends.
func (s *shell) Serve(ctx context.Context, ln net.Listener, idle time.Duration) {
	srv := &http.Server{Handler: s.Handler()}
	go srv.Serve(ln)
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
		case <-t.C:
			s.mu.Lock()
			quiet := time.Since(s.lastPoll) > idle
			s.mu.Unlock()
			if !quiet {
				continue
			}
		}
		break
	}
	shut, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	srv.Shutdown(shut)
}

const shellHTML = `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>d.idle watch</title>
<style>
body{margin:0;min-height:100vh;display:grid;place-items:center;background:#14110f;color:#e8dcc8;font:16px system-ui,sans-serif}
main{max-width:30rem;padding:2rem;text-align:center}h1{font-size:1.4rem;margin:0 0 1rem}
p{line-height:1.5;color:#c9bca4}input{width:100%;box-sizing:border-box;padding:.6rem;font-size:1rem;margin:.5rem 0}
button{padding:.55rem 1.1rem;font-size:1rem;margin:.3rem;cursor:pointer}.err{color:#ff8a7a}code{color:#e8dcc8}
</style></head><body><main>
<h1 id="h">d.idle watch</h1><p id="m">Checking the server...</p>
<form id="f" hidden><input id="a" name="address" placeholder="host:port" autocomplete="off" spellcheck="false" autofocus>
<p id="e" class="err" role="alert"></p><button type="submit">Connect</button></form>
<div id="r" hidden><button id="retry">Retry now</button><button id="chg">Change address</button></div>
</main><script>
var base=location.pathname,tries=0,timer=null,$=function(i){return document.getElementById(i)};
function show(prompt){$("f").hidden=!prompt;$("r").hidden=prompt}
function tick(){clearTimeout(timer);fetch(base+"status",{cache:"no-store"}).then(function(r){return r.json()}).then(function(s){
 if(!s.address){$("h").textContent="Where is the game?";$("m").innerHTML="Enter the address of the d.idle watch page, for example <code>host:8891</code>.";show(true);return}
 show(false);
 if(s.state==="up"){$("h").textContent="Connected";$("m").textContent="Opening the watch page...";location.replace(s.target);return}
 tries++;
 $("h").textContent=s.state==="warming"?"The game is warming up":"The server is not reachable";
 $("m").textContent=(s.state==="warming"?"The server answered but is not ready yet. ":"")+"The container takes one to three minutes to start. Trying "+s.address+" again every few seconds (attempt "+tries+").";
 timer=setTimeout(tick,3000)}).catch(function(){timer=setTimeout(tick,3000)})}
$("retry").onclick=tick;
$("chg").onclick=function(){fetch(base+"change",{method:"POST"}).then(tick)};
$("f").onsubmit=function(ev){ev.preventDefault();fetch(base+"save",{method:"POST",body:new URLSearchParams({address:$("a").value})}).then(function(r){return r.json()}).then(function(j){
 if(j.error){$("e").textContent=j.error;return}$("e").textContent="";tries=0;tick()})};
tick();
</script></body></html>`
