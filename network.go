package main

import (
	"fmt"
	"log"
	"time"
)

// verbose = piszemy komunikaty?
var (
	RoundDelay time.Duration
	Verbose    bool
)

// wierzcholek z perspektywy sieci
type Process interface {
	Round(round int, wokenByAdversary bool, inbox []Incoming) []Outgoing
	Done() bool
}

type LeaderReporter interface {
	Leader() (rank uint64, ok bool)
}

type AwakeReporter interface {
	Awake() bool
}

type Adversary interface {
	// WakeUps zwraca wierzchołki, które przeciwnik budzi w rundzie round.
	WakeUps(round int) []int
	// Finished mówi, że po rundzie round przeciwnik nie obudzi już nikogo.
	Finished(round int) bool
}

// to jest po to zeby sam algorytm nigdy nie widzial tych id wierzcholkow
// zamiast from, do algorytmu przekazujemy Port(id, e.from)
type envelope struct {
	from int
	msg  Message
}

// zacznij runde
type roundCmd struct {
	round int
	woken bool
}

// dopoki Run nie zbierze n raportow, runda trwa
type report struct {
	sent   int              // ile wierzcholek wyslal w tej rundzie
	byType [numMsgTypes]int // ile kazdego typu
	done   bool             // Done()
}

type Stats struct {
	SimRounds int
	FirstWake int
	LastSend  int
	Messages  int
	ByType    [numMsgTypes]int
	PerRound  []int
	HitLimit  bool
}

func (s Stats) AlgorithmRounds() int {
	if s.LastSend < 0 {
		return 0
	}
	return s.LastSend - s.FirstWake + 1
}

type Network struct {
	n       int
	ports   *PortMap
	procs   []Process
	inbox   []chan envelope // wierzcholek -> wierzcholek v, bufor 1
	start   []chan roundCmd // Run -> v (zacznij runde)
	end     []chan struct{} // Run -> (wszyscy skonczyli wysylac, zamknij runde)
	reports chan report     // wszyscy -> Run (skonczylem wysylac)
	acks    chan struct{}   // wszyscy -> Run (przeczytalem caly inbox, nastepna runda moze byc)
	Stats   Stats
}

func NewNetwork(procs []Process, ports *PortMap) *Network {
	n := len(procs)
	net := &Network{
		n:       n,
		ports:   ports,
		procs:   procs,
		inbox:   make([]chan envelope, n),
		start:   make([]chan roundCmd, n),
		end:     make([]chan struct{}, n),
		reports: make(chan report),
		acks:    make(chan struct{}),
	}
	for i := range n {
		net.inbox[i] = make(chan envelope, 1)
		net.start[i] = make(chan roundCmd)
		net.end[i] = make(chan struct{})
	}
	return net
}

func (net *Network) runNode(id int) {
	p := net.procs[id]
	var inbox []Incoming

	// czekamy na wiadomosc (zacznij runde)
	// gdy zamykamy start (a zamykamy gdy petla w Run sie konczy)
	// ta petla tez sie konczy
	// czyli nic nie wisi
	for cmd := range net.start[id] {
		// wykonaj to co masz wykonac w tej rundzie
		out := p.Round(cmd.round, cmd.woken, inbox)

		var next []Incoming // wiadomości przychodzące w tej rundzie
		receive := func(e envelope) {
			next = append(next, Incoming{Port: net.ports.Port(id, e.from), Msg: e.msg})
		}

		var rep report
		for _, o := range out {
			if o.Port < 0 || o.Port >= net.n-1 {
				panic(fmt.Sprintf("wierzchołek %d: niepoprawny port %d (dozwolone 0..%d)", id, o.Port, net.n-2))
			}
			to := net.ports.Neighbor(id, o.Port)
			if Verbose {
				log.Printf("  %d wysłał %v do %d", id, o.Msg, to)
			}
			e := envelope{from: id, msg: o.Msg}
			for sent := false; !sent; {
				select {
				case net.inbox[to] <- e:
					sent = true
				case in := <-net.inbox[id]:
					receive(in)
				}
			}
			rep.sent++
			rep.byType[o.Msg.Type]++
		}
		rep.done = p.Done()

		for sent := false; !sent; {
			select {
			case net.reports <- rep:
				sent = true
			case in := <-net.inbox[id]:
				receive(in)
			}
		}
		for ended := false; !ended; {
			select {
			case <-net.end[id]:
				ended = true
			case in := <-net.inbox[id]:
				receive(in)
			}
		}
		for drained := false; !drained; {
			select {
			case in := <-net.inbox[id]:
				receive(in)
			default:
				drained = true
			}
		}
		inbox = next
		net.acks <- struct{}{}
	}
}

func (net *Network) Run(adv Adversary, maxRounds int) Stats {
	// odpalamy kazdy wierzch
	for id := range net.n {
		go net.runNode(id)
	}
	// gdy konczymy program, zamykamy wszystkie kanaly startu
	defer func() {
		for id := range net.n {
			close(net.start[id])
		}
	}()

	// init statystyk
	st := &net.Stats
	st.FirstWake = -1
	st.LastSend = -1
	woken := make([]bool, net.n)

	// jedna iteracja = jedna runda = jedna jednostka czasu
	for r := 0; ; r++ {
		if r >= maxRounds {
			st.HitLimit = true
			break
		}
		// kogo budzimy??
		wake := adv.WakeUps(r)

		if len(wake) > 0 && st.FirstWake < 0 {
			st.FirstWake = r
		}

		if Verbose {
			log.Printf("=== runda %d rozpoczęta ===", r)
		}

		for _, v := range wake {
			woken[v] = true
		}

		// czy adwersarz cie budzi?
		for id := range net.n {
			net.start[id] <- roundCmd{round: r, woken: woken[id]}
		}

		// reset
		for _, v := range wake {
			woken[v] = false
		}

		sent, allDone := 0, true

		for range net.n {
			// wierzcholek skonczyl runde, daje raport
			rep := <-net.reports
			sent += rep.sent

			for t := range rep.byType {
				st.ByType[t] += rep.byType[t]
			}

			allDone = allDone && rep.done
		}

		// kazdy juz dal raport -> koniec rundy
		for id := range net.n {
			net.end[id] <- struct{}{}
		}

		// gotowi na nastepna?
		for range net.n {
			<-net.acks
		}

		st.SimRounds = r + 1
		st.Messages += sent
		st.PerRound = append(st.PerRound, sent)

		if sent > 0 {
			st.LastSend = r
		}

		if Verbose {
			log.Printf("=== runda %d zakończona: wysłano %d wiadomości ===", r, sent)
		}

		time.Sleep(RoundDelay)

		if sent == 0 && allDone && adv.Finished(r) {
			break
		}
	}
	return *st
}
