// inputcal: live measurement tool for internal/input (run via tools/live.sh input inputcal <sub> [args]).
// Subcommands: state, watch, sample N, hover.
package main

import (
	"fmt"
	"os"
	"strconv"

	"d2lod-bot/internal/input"
	"d2lod-bot/internal/mem"
)

type env struct {
	r  input.Reader
	cl *input.Client
	g  *input.Game
}

func main() {
	pid, err := input.FindGamePID()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	src, err := input.OpenProc(pid)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer src.Close()
	e := &env{r: input.Reader{Src: src}, cl: input.NewClient("")}
	e.g = input.NewGame(e.cl, e.r)
	sub := "state"
	if len(os.Args) > 1 {
		sub = os.Args[1]
	}
	arg := func(i, def int) int {
		if len(os.Args) > i {
			if n, err := strconv.Atoi(os.Args[i]); err == nil {
				return n
			}
		}
		return def
	}
	switch sub {
	case "state":
		e.state()
	case "dump": // dump RVA LEN: raw bytes as hex, one line
		rva, _ := strconv.ParseUint(os.Args[2], 0, 32)
		b := make([]byte, arg(3, 64))
		if err := src.ReadAt(b, uint32(0x400000+rva)); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Printf("%x\n", b)
	case "hover":
		e.hover(arg(2, 150))
	case "ui":
		e.ui()
	case "key": // key NAME
		fmt.Println(e.cl.Seq("key press " + os.Args[2] + " hold=60"))
	case "peek": // peek CLASS dy...
		var dys []int
		for _, a := range os.Args[3:] {
			n, _ := strconv.Atoi(a)
			dys = append(dys, n)
		}
		e.peek(arg(2, 179), dys)
	case "near":
		e.nearby()
	case "goto":
		e.g.T = input.Instant()
		fmt.Println(e.gotoNear(arg(2, 0), arg(3, 0), float64(arg(4, 6))))
		p, _ := e.r.Player()
		fmt.Printf("at (%.1f,%.1f)\n", p.X, p.Y)
	case "hovertest": // hovertest CLASS: sweep the pointer over a fresh-positioned NPC, print the hover vars
		e.hoverTest(arg(2, 150))
	case "selftest":
		e.selftest()
	case "sample":
		e.sample(arg(2, 30))
	default:
		fmt.Println("unknown subcommand")
	}
}

func (e *env) state() {
	fmt.Println("ping:", e.cl.Ping())
	fmt.Println(e.cl.Info())
	p, err := e.r.Player()
	fmt.Printf("player %+v err=%v\n", p, err)
	vx, vy, err := e.r.Viewport()
	fmt.Println("viewport", vx, vy, err)
	mx, my, err := e.r.Mouse()
	fmt.Println("mouse", mx, my, err)
	a, err := e.r.Area()
	fmt.Println("area", a, err)
}

// uiVars prints the nonzero entries of the UI-open array found by disassembling GetUiVar (RVA 0x538D0:
// mov eax,[ecx*4+0x7A27C0]), 0x26 entries.
func (e *env) ui() {
	for i := uint32(0); i < 0x26; i++ {
		v, _ := mem.U32(e.r.Src, mem.Base+0x3A27C0+i*4)
		if v != 0 {
			fmt.Printf("ui[%#x]=%d ", i, v)
		}
	}
	fmt.Println()
}
