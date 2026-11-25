package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"time"
)

func main() {
	n := flag.Int("n", 16, "liczba wierzchołków (min 2, max 65535)")
	seed := flag.Uint64("seed", 0, "ziarno losowości (0 = losowe)")
	wake := flag.String("wake", "random", "tryb budzenia przez adwersarza: all | one | random")
	k := flag.Int("k", 4, "tryb random: ilu wierzchołków budzi adwersarz")
	T := flag.Int("T", 5, "tryb random: budzenie w losowych rundach z [0, T)")
	maxRounds := flag.Int("maxrounds", 100, "limit rund symulacji")
	delay := flag.Duration("delay", 0, "opóźnienie między rundami, np. 500ms")
	verbose := flag.Bool("v", true, "logi rund i wiadomości (dla dużych n użyj -v=false)")
	interactive := flag.Bool("interactive", false, "przeciwnik interaktywny: przed każdą rundą pyta, kogo obudzić (ignoruje -wake)")
	flag.Parse()

	log.SetFlags(log.Lmicroseconds)
	Verbose = *verbose
	RoundDelay = *delay

	if *n < 2 || *n > 65535 {
		fmt.Fprintln(os.Stderr, "n musi być z zakresu 2..65535")
		os.Exit(2)
	}
	if *seed == 0 {
		*seed = uint64(time.Now().UnixNano())
	}

	adversary := rand.New(rand.NewPCG(*seed, 0))
	ports := NewPortMap(*n, adversary)

	nodes := make([]*LeaderNode, *n)
	procs := make([]Process, *n)
	for id := range *n {
		nodes[id] = NewLeaderNode(id, *n, rand.New(rand.NewPCG(*seed, uint64(id)+1)))
		procs[id] = nodes[id]
	}

	var adv Adversary
	switch {
	case *interactive:
		adv = NewInteractiveAdversary(os.Stdin, os.Stdout, procs)
	default:
		schedule, err := GenerateSchedule(*wake, *n, *k, *T, adversary)
		if err != nil {
			fmt.Fprintln(os.Stderr, "błąd planu budzenia:", err)
			os.Exit(2)
		}
		adv = schedule
	}

	net := NewNetwork(procs, ports)
	st := net.Run(adv, *maxRounds)

	printSummary(*n, *seed, st, procs)
}

func printSummary(n int, seed uint64, st Stats, procs []Process) {
	fmt.Println()
	fmt.Println("=== Podsumowanie ===")
	fmt.Printf("n = %d, seed = %d\n", n, seed)
	fmt.Printf("rundy symulacji: %d", st.SimRounds)
	if st.HitLimit {
		fmt.Print(" (PRZERWANO: osiągnięto limit rund)")
	}
	fmt.Println()
	fmt.Printf("czas algorytmu (od pierwszego obudzenia do ostatniej wiadomości): %d rund\n", st.AlgorithmRounds())
	fmt.Printf("wiadomości: %d (%.2f na wierzchołek)\n", st.Messages, float64(st.Messages)/float64(n))
	for t := range numMsgTypes {
		fmt.Printf("  %-17v %d\n", t, st.ByType[t])
	}
	fmt.Printf("wiadomości w kolejnych rundach: %v\n", st.PerRound)

	leaders := map[uint64]int{}
	without := 0
	for _, p := range procs {
		lr, ok := p.(LeaderReporter)
		if !ok {
			continue
		}
		if rank, has := lr.Leader(); has {
			leaders[rank]++
		} else {
			without++
		}
	}
	switch {
	case len(leaders) == 1 && without == 0:
		for rank := range leaders {
			fmt.Printf("OK: wszystkie wierzchołki znają lidera (RANK = %d)\n", rank)
		}
	default:
		fmt.Printf("BŁĄD: %d różnych liderów, %d wierzchołków bez lidera\n", len(leaders), without)
	}
}
