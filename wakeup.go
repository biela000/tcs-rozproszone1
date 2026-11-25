package main

import (
	"fmt"
	"math/rand/v2"
)

type WakeSchedule map[int][]int

func GenerateSchedule(mode string, n, k, T int, adversary *rand.Rand) (WakeSchedule, error) {
	s := WakeSchedule{}
	switch mode {
	case "all":
		for v := range n {
			s[0] = append(s[0], v)
		}
	case "one":
		s[0] = []int{adversary.IntN(n)}
	case "random":
		if k < 1 || k > n {
			return nil, fmt.Errorf("k musi być z zakresu 1..%d, jest %d", n, k)
		}
		if T < 1 {
			return nil, fmt.Errorf("T musi być >= 1, jest %d", T)
		}
		for _, v := range adversary.Perm(n)[:k] {
			r := adversary.IntN(T)
			s[r] = append(s[r], v)
		}
	default:
		return nil, fmt.Errorf("nieznany tryb budzenia %q (dostępne: all, one, random)", mode)
	}
	return s, nil
}

func (s WakeSchedule) FirstRound() int {
	first := -1
	for r := range s {
		if first < 0 || r < first {
			first = r
		}
	}
	return first
}

func (s WakeSchedule) WakeUps(round int) []int { return s[round] }

func (s WakeSchedule) Finished(round int) bool { return round >= s.LastRound() }

func (s WakeSchedule) LastRound() int {
	last := -1
	for r := range s {
		if r > last {
			last = r
		}
	}
	return last
}
