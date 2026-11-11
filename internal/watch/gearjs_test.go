package watch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// gear.js in a stub DOM (node, skipped without it): a refused POST must stay on screen, a good one says requested.
func TestGearJSShowsFeedbackForRefusalAndHTTPError(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node")
	}
	js, _ := webFS.ReadFile("web/gear.js")
	harness := `
var posts=[],mode='refuse',els=[];
function mk(t){var e={tag:t,children:[],hidden:false,className:'',textContent:'',listeners:{},
 appendChild:function(c){this.children.push(c);return c;},setAttribute:function(){},addEventListener:function(n,f){this.listeners[n]=f;},
 insertAdjacentElement:function(w,c){top.after=c;}};els.push(e);return e;}
var top=mk('header');
global.document={querySelector:function(){return top;},getElementById:function(){return null;},createElement:mk};
global.location={protocol:'http:',hostname:'h'};
global.setInterval=function(){};
global.fetch=function(u){
 if(u==='/api/config')return Promise.resolve({json:function(){return Promise.resolve({answerBase:'http://a'});}});
 if(u==='/api/repair')return Promise.resolve({ok:true,json:function(){return Promise.resolve({gear:[{slot:'torso',name:'Plate',dur:0,max:50,pct:0,broken:true}],request:mode==='ok'?{state:'requested',short:41,need:116}:null});}});
 posts.push(u);
 if(mode==='refuse')return Promise.resolve({ok:false,status:409,json:function(){return Promise.resolve({error:'nothing worn is damaged'});}});
 if(mode==='html')return Promise.resolve({ok:false,status:403,json:function(){return Promise.reject(new Error('x'));}});
 return Promise.resolve({ok:true,status:200,json:function(){return Promise.resolve({ok:true});}});};
` + string(js) + `
function texts(e,o){o=o||[];if(e.textContent)o.push(e.textContent);(e.children||[]).forEach(function(c){texts(c,o);});return o;}
function find(e,tag){if(e.tag===tag)return e;for(var i=0;i<e.children.length;i++){var r=find(e.children[i],tag);if(r)return r;}return null;}
function tick(){return new Promise(function(r){setTimeout(r,20);});}
(async function(){
 await tick();
 var out={};
 for (const m of ['refuse','html','ok']){
  mode=m; await tick();
  var box=top.after, b=find(box,'button'); b.listeners.click(); await tick(); await tick();
  out[m]=texts(top.after).join(' | ');
 }
 console.log(JSON.stringify(out));
})();`
	f := filepath.Join(t.TempDir(), "h.js")
	os.WriteFile(f, []byte(harness), 0o600)
	b, err := exec.Command(node, f).CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, b)
	}
	s := string(b)
	for _, want := range []string{"Not sent: nothing worn is damaged", "Not sent: the answer route said HTTP 403", "Repair requested: the hero goes at the next safe moment", "waiting for gold: need 41 more"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in %s", want, s)
		}
	}
}
