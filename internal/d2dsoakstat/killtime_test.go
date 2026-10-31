package d2dsoakstat

import (
	"strings"
	"testing"
)

func TestKillTimes(t *testing.T) {
	log := `{"msg":"minute","goal":"g","step":"s","rstate":"clearing"}
{"msg":"nav stuck"}
{"msg":"killtime","monster":"Zombie","area":2,"total":5.5,"walk":2,"cast":1,"wait":2.5,"gap_since_last_kill":6,"state":"farming"}
{"msg":"clear killed nothing"}
{"msg":"clear killed nothing"}
{"msg":"killtime","monster":"Fallen","area":2,"total":50,"walk":1,"cast":1,"wait":48,"gap_since_last_kill":60,"state":"farming"}
`
	ks, err := ParseKillTimes(strings.NewReader(log))
	if err != nil || len(ks) != 2 {
		t.Fatal(err, ks)
	}
	if ks[0].Msgs["nav stuck"] != 1 || ks[1].Msgs["clear killed nothing"] != 2 || ks[1].Goal != "g" {
		t.Fatalf("%+v", ks)
	}
	s := SummarizeKills(ks, 1)
	if s.Wait != 25.25 || s.Out != 5.25 || s.Top[0].Monster != "Fallen" || s.Hist[3].Kills != 1 {
		t.Fatalf("%+v", s)
	}
	if !strings.Contains(s.Text("x"), "clear killed nothing x2") {
		t.Fatal(s.Text("x"))
	}
}
