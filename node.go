package main

import (
	"fmt"
	"log"
	"math"
	"math/rand/v2"
)

type Role int

const (
	Asleep Role = iota
	SilentCandidate
	ActiveCandidate
	Referee
	Retired
	Won
)

func (r Role) String() string {
	return [...]string{"Asleep", "Silent", "Active", "Referee", "Retired", "Winner"}[r]
}

type LeaderNode struct {
	id  int
	n   int
	rng *rand.Rand // prywatna moneta

	role       Role
	rank       uint64
	wakeRound  int
	activeFrom int

	awaitingReplies    map[int]bool
	beatenAsOwnReferee bool

	leader    uint64
	hasLeader bool
}

func NewLeaderNode(id, n int, rng *rand.Rand) *LeaderNode {
	return &LeaderNode{id: id, n: n, rng: rng, role: Asleep}
}

func (s *LeaderNode) Round(round int, wokenByAdversary bool, inbox []Incoming) []Outgoing {
	// odczyt i podzial
	var requests, replies, winners []Incoming
	for _, in := range inbox {
		switch in.Msg.Type {
		case ApprovalRequest:
			requests = append(requests, in)
		case ApprovalReply:
			replies = append(replies, in)
		case Winner:
			winners = append(winners, in)
		}
	}

	// budzimy jesli spimy i zostalismy obudzeni
	if s.role == Asleep {
		switch {
		case wokenByAdversary:
			s.wakeUp(round, SilentCandidate, "obudzony przez adwersarza")
		case len(requests) > 0:
			s.wakeUp(round, Referee, "obudzony przez APPROVAL_REQUEST")
		case len(winners) > 0:
			s.wakeUp(round, Retired, "obudzony przez WINNER")
		default:
			return nil
		}
	}

	// wysylanie tego co powinnismy
	var out []Outgoing
	s.handleWinners(round, winners)
	out = append(out, s.handleApprovalRequests(round, requests)...)
	out = append(out, s.handleApprovalReplies(round, replies)...)
	out = append(out, s.tryBecomeActive(round)...)
	return out
}

func (s *LeaderNode) wakeUp(round int, role Role, why string) {
	s.wakeRound = round
	s.rank = s.randomRank()
	s.become(role, round, fmt.Sprintf("%s, RANK=%d", why, s.rank))
}

func (s *LeaderNode) handleWinners(round int, winners []Incoming) {
	if len(winners) == 0 {
		return
	}
	best := uint64(0)
	if s.hasLeader {
		best = s.leader
	}
	for _, w := range winners {
		best = max(best, w.Msg.Rank)
	}
	s.leader, s.hasLeader = best, true
	if Verbose {
		log.Printf("  wierzchołek %d: lider = RANK %d", s.id, best)
	}
	if s.role == SilentCandidate || s.role == ActiveCandidate {
		s.become(Retired, round, "dostał WINNER")
	}
}

func (s *LeaderNode) handleApprovalRequests(round int, requests []Incoming) []Outgoing {
	if len(requests) == 0 {
		return nil
	}
	// kto zna już lidera, nie odpowiada (spóźniony kandydat z Z'' i tak odpada)
	if s.hasLeader {
		return nil
	}
	if s.role == SilentCandidate {
		s.become(Retired, round, "dostał APPROVAL_REQUEST")
	}

	best := uint64(0)
	for _, r := range requests {
		best = max(best, r.Msg.Rank)
	}
	if s.role == ActiveCandidate && round == s.activeFrom+1 {
		if best > s.rank {
			s.beatenAsOwnReferee = true
		}
		best = max(best, s.rank)
	}

	out := make([]Outgoing, len(requests))
	for i, r := range requests {
		out[i] = Outgoing{Port: r.Port, Msg: Message{Type: ApprovalReply, Rank: best}}
	}
	return out
}

func (s *LeaderNode) handleApprovalReplies(round int, replies []Incoming) []Outgoing {
	if s.role != ActiveCandidate || round != s.activeFrom+2 {
		return nil
	}
	beaten := s.beatenAsOwnReferee
	for _, r := range replies {
		delete(s.awaitingReplies, r.Port)
		if r.Msg.Rank != s.rank {
			beaten = true
		}
	}
	if beaten {
		s.become(Retired, round, "sędzia zwrócił większy RANK")
		return nil
	}
	if len(s.awaitingReplies) > 0 {
		panic(fmt.Sprintf("wierzchołek %d: brak odpowiedzi od %d sędziów", s.id, len(s.awaitingReplies)))
	}

	s.become(Won, round, "wszyscy sędziowie zwrócili jego RANK")
	s.leader = max(s.leader, s.rank)
	s.hasLeader = true
	return s.broadcast(Message{Type: Winner, Rank: s.rank})
}

func (s *LeaderNode) tryBecomeActive(round int) []Outgoing {
	if s.role != SilentCandidate {
		return nil
	}
	var p float64
	switch round - s.wakeRound {
	case 0:
		p = math.Pow(float64(s.n), -2.0/3)
	case 3:
		p = math.Pow(float64(s.n), -1.0/3)
	case 6:
		p = 1
	default:
		return nil
	}
	if !s.coin(p) {
		return nil
	}

	s.activeFrom = round
	s.become(ActiveCandidate, round, fmt.Sprintf("orzeł w próbie %d", (round-s.wakeRound)/3+1))
	ports := s.sampleReferees(numReferees(s.n))
	s.awaitingReplies = make(map[int]bool, len(ports))
	out := make([]Outgoing, len(ports))
	for i, p := range ports {
		s.awaitingReplies[p] = true
		out[i] = Outgoing{Port: p, Msg: Message{Type: ApprovalRequest, Rank: s.rank}}
	}
	return out
}

func (s *LeaderNode) Done() bool {
	return s.role != SilentCandidate && s.role != ActiveCandidate
}

func (s *LeaderNode) Awake() bool {
	return s.role != Asleep
}

func (s *LeaderNode) String() string {
	if s.role == Asleep {
		return "Asleep"
	}
	str := fmt.Sprintf("%v, RANK=%d, obudzony w rundzie %d", s.role, s.rank, s.wakeRound)
	if s.hasLeader {
		str += fmt.Sprintf(", lider=%d", s.leader)
	}
	return str
}

func (s *LeaderNode) Leader() (uint64, bool) {
	return s.leader, s.hasLeader
}

func (s *LeaderNode) become(role Role, round int, why string) {
	if Verbose {
		log.Printf("  wierzchołek %d: %v -> %v (%s)", s.id, s.role, role, why)
	}
	s.role = role
}

func (s *LeaderNode) randomRank() uint64 {
	n := uint64(s.n)
	return 1 + s.rng.Uint64N(n*n*n*n)
}

func (s *LeaderNode) coin(p float64) bool {
	return s.rng.Float64() < p
}

func numReferees(n int) int {
	k := int(math.Ceil(2 * math.Sqrt(float64(n)) * math.Log(float64(n))))
	return min(k, n-1)
}

func (s *LeaderNode) sampleReferees(k int) []int {
	if k == s.n-1 {
		ports := make([]int, k)
		for i := range ports {
			ports[i] = i
		}
		return ports
	}
	chosen := make(map[int]bool, k)
	ports := make([]int, 0, k)
	for len(ports) < k {
		p := s.rng.IntN(s.n - 1)
		if !chosen[p] {
			chosen[p] = true
			ports = append(ports, p)
		}
	}
	return ports
}

func (s *LeaderNode) broadcast(m Message) []Outgoing {
	out := make([]Outgoing, s.n-1)
	for p := range out {
		out[p] = Outgoing{Port: p, Msg: m}
	}
	return out
}
