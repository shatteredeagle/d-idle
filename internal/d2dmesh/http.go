package d2dmesh

import (
	"archive/tar"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// handler serves gossip, the JSON API and the status page on the one TLS port. Every
// request already passed mutual TLS, so there is no second credential here.
func (n *Node) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /mesh/v1/push", n.handlePush)
	mux.HandleFunc("GET /api/v1/state", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, n.Snapshot()) })
	mux.HandleFunc("GET /api/v1/nodes", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, n.Snapshot().NodesSorted()) })
	mux.HandleFunc("GET /api/v1/jobs", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, n.Snapshot().JobsSorted()) })
	mux.HandleFunc("POST /api/v1/jobs", n.handleSubmit)
	mux.HandleFunc("POST /api/v1/jobs/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		fail(w, n.Cancel(r.PathValue("id")), http.StatusNotFound)
	})
	mux.HandleFunc("GET /api/v1/jobs/{id}/attempts/{n}/tar", n.handleTar)
	for _, verb := range []string{"pause", "resume", "drain", "undrain"} {
		verb := verb
		mux.HandleFunc("POST /api/v1/nodes/{name}/"+verb, func(w http.ResponseWriter, r *http.Request) {
			t, f := true, false
			var err error
			switch verb {
			case "pause":
				err = n.Control(r.PathValue("name"), &t, nil)
			case "resume":
				err = n.Control(r.PathValue("name"), &f, nil)
			case "drain":
				err = n.Control(r.PathValue("name"), nil, &t)
			case "undrain":
				err = n.Control(r.PathValue("name"), nil, &f)
			}
			fail(w, err, http.StatusNotFound)
		})
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, statusPage)
	})
	return mux
}

func (n *Node) handlePush(w http.ResponseWriter, r *http.Request) {
	n.mu.Lock()
	cut := n.cut
	n.mu.Unlock()
	if cut {
		http.Error(w, "partitioned (test hook)", http.StatusServiceUnavailable)
		return
	}
	theirs, err := DecodeState(r.Body)
	if err != nil {
		n.cfg.Logf("push from %s refused: %v", r.RemoteAddr, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	n.mergeIn(theirs)
	writeJSON(w, n.Snapshot())
}

type submitReq struct {
	Recipe string            `json:"recipe"`
	Args   map[string]string `json:"args"`
	Labels []string          `json:"labels"`
}

func (n *Node) handleSubmit(w http.ResponseWriter, r *http.Request) {
	var req submitReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := n.Submit(req.Recipe, req.Args, req.Labels)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	writeJSON(w, map[string]string{"id": id})
}

// handleTar streams an attempt's output directory (result collection). Only a directory
// recorded in the job's result on this node is served; nothing else on the host is reachable.
func (n *Node) handleTar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	attempt, _ := strconv.Atoi(r.PathValue("n"))
	st := n.Snapshot()
	j := st.Jobs[id]
	if j == nil || j.Result == nil || j.Result.Node != n.cfg.Name || j.Result.Attempt != attempt || j.Result.OutDir == "" {
		http.Error(w, "no output for that attempt on this node", http.StatusNotFound)
		return
	}
	root := filepath.Clean(j.Result.OutDir)
	w.Header().Set("Content-Type", "application/x-tar")
	tw := tar.NewWriter(w)
	defer tw.Close()
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		info, err := d.Info()
		if err != nil {
			return nil
		}
		f, err := os.Open(p)
		if err != nil {
			return nil
		}
		defer f.Close()
		if err := tw.WriteHeader(&tar.Header{Name: rel, Mode: 0o644, Size: info.Size(), ModTime: info.ModTime()}); err != nil {
			return err
		}
		_, err = io.Copy(tw, f)
		return err
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, err error, code int) {
	if err != nil {
		http.Error(w, err.Error(), code)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// statusPage is the embedded page every node serves at /: one table of nodes, one of jobs,
// polled from the JSON API; the verbs are the same POSTs the CLI uses.
const statusPage = `<!doctype html><meta charset="utf-8"><title>d2d mesh</title>
<style>body{font:14px system-ui;margin:1.5em}table{border-collapse:collapse}td,th{padding:.2em .6em;border-bottom:1px solid #ccc;text-align:left}button{font:inherit}</style>
<h1>d2d mesh</h1><h2>Nodes</h2><table id=n></table><h2>Jobs</h2><table id=j></table>
<script>
const post=(p)=>fetch(p,{method:'POST'}).then(load);
const esc=s=>String(s??'').replace(/[&<>]/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;'}[c]));
async function load(){const s=await (await fetch('/api/v1/state')).json();
const nodes=Object.values(s.nodes).filter(n=>!n.gone).sort((a,b)=>a.name<b.name?-1:1);
document.getElementById('n').innerHTML='<tr><th>node<th>addr<th>labels<th>run/cap<th>flags<th>seen<th>'+nodes.map(n=>'<tr><td>'+esc(n.name)+'<td>'+esc(n.addr)+'<td>'+esc((n.labels||[]).join(' '))+'<td>'+n.running+'/'+n.capacity+'<td>'+(n.ctl.paused?'paused ':'')+(n.ctl.draining?'draining':'')+'<td>'+new Date(n.last_seen*1000).toLocaleTimeString()+'<td><button onclick="post(\'/api/v1/nodes/'+esc(n.name)+'/'+(n.ctl.paused?'resume':'pause')+'\')">'+(n.ctl.paused?'resume':'pause')+'</button> <button onclick="post(\'/api/v1/nodes/'+esc(n.name)+'/'+(n.ctl.draining?'undrain':'drain')+'\')">'+(n.ctl.draining?'undrain':'drain')+'</button>').join('');
const jobs=Object.values(s.jobs).sort((a,b)=>a.submitted-b.submitted);
document.getElementById('j').innerHTML='<tr><th>id<th>recipe<th>state<th>owner<th>attempt<th>fence<th>result<th>'+jobs.map(j=>'<tr><td>'+esc(j.id)+'<td>'+esc(j.recipe)+'<td>'+esc(j.state)+'<td>'+esc(j.owner)+'<td>'+j.attempt+'<td>'+j.fence+'<td>'+esc(j.result?j.result.node+' exit '+j.result.exit+' '+(j.result.summary||''):'')+'<td>'+(j.state=='queued'||j.state=='running'?'<button onclick="post(\'/api/v1/jobs/'+esc(j.id)+'/cancel\')">cancel</button>':'')).join('');}
load();setInterval(load,3000);
</script>`

// SafeName keeps a mesh id safe for paths and logs (runners build output dirs from it).
func SafeName(s string) string {
	return strings.Map(func(r rune) rune {
		if ident.MatchString(string(r)) {
			return r
		}
		return '_'
	}, s)
}
