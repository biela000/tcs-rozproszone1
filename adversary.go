package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type InteractiveAdversary struct {
	in      *bufio.Scanner
	out     io.Writer
	procs   []Process
	stopped bool // użytkownik zakończył budzenie ('q' albo koniec wejścia)
}

func NewInteractiveAdversary(in io.Reader, out io.Writer, procs []Process) *InteractiveAdversary {
	return &InteractiveAdversary{in: bufio.NewScanner(in), out: out, procs: procs}
}

func (a *InteractiveAdversary) awake(v int) bool {
	ar, ok := a.procs[v].(AwakeReporter)
	return ok && ar.Awake()
}

func (a *InteractiveAdversary) asleepCount() int {
	c := 0
	for v := range a.procs {
		if !a.awake(v) {
			c++
		}
	}
	return c
}

func (a *InteractiveAdversary) Finished(int) bool {
	return a.stopped || a.asleepCount() == 0
}

func (a *InteractiveAdversary) WakeUps(round int) []int {
	if a.stopped {
		return nil
	}
	n := len(a.procs)
	for {
		asleep := a.asleepCount()
		if asleep == 0 {
			return nil
		}
		fmt.Fprintf(a.out, "\n[przeciwnik] runda %d, śpi %d z %d wierzchołków.\n", round, asleep, n)
		fmt.Fprint(a.out, "Które obudzić? (np. \"0 3 5\"; Enter = żadnego, s = stan, q = koniec budzenia): ")

		if !a.in.Scan() {
			fmt.Fprintln(a.out, "\n(koniec wejścia: przeciwnik nie budzi już nikogo)")
			a.stopped = true
			return nil
		}
		line := strings.TrimSpace(a.in.Text())
		switch line {
		case "":
			return nil
		case "s":
			a.printState()
			continue
		case "q":
			a.stopped = true
			return nil
		}

		ids, err := parseIDs(line, n)
		if err != nil {
			fmt.Fprintln(a.out, "Błąd:", err, "- spróbuj jeszcze raz.")
			continue
		}
		var wake []int
		chosen := map[int]bool{}
		for _, v := range ids {
			switch {
			case chosen[v]:
				// powtorzony w tej samej linii, pomijamy po cichu
			case a.awake(v):
				fmt.Fprintf(a.out, "Wierzchołek %d jest już obudzony (%v), pomijam.\n", v, a.procs[v])
			default:
				chosen[v] = true
				wake = append(wake, v)
			}
		}
		if len(wake) > 0 {
			fmt.Fprintf(a.out, "Budzę: %v\n", wake)
		}
		return wake
	}
}

func (a *InteractiveAdversary) printState() {
	for v, p := range a.procs {
		fmt.Fprintf(a.out, "  %d: %v\n", v, p)
	}
}

func parseIDs(line string, n int) ([]int, error) {
	fields := strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == ',' || r == '\t' })
	ids := make([]int, 0, len(fields))
	for _, f := range fields {
		v, err := strconv.Atoi(f)
		if err != nil || v < 0 || v >= n {
			return nil, fmt.Errorf("%q nie jest numerem wierzchołka (dozwolone 0..%d)", f, n-1)
		}
		ids = append(ids, v)
	}
	return ids, nil
}
