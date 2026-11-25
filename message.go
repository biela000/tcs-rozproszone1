package main

import "fmt"

// MsgType to typ wiadomości. Algorytm z Sekcji 3 paperu potrzebuje trzech.
type MsgType int

const (
	ApprovalRequest MsgType = iota // aktywny kandydat -> sędzia, zawiera RANK kandydata
	ApprovalReply                  // sędzia -> kandydat, zawiera największy RANK, jaki sędzia zobaczył w tej rundzie
	Winner                         // zwycięzca -> wszyscy, WINNER(RANK)

	numMsgTypes // liczba typów (musi być ostatnia)
)

func (t MsgType) String() string {
	switch t {
	case ApprovalRequest:
		return "APPROVAL_REQUEST"
	case ApprovalReply:
		return "APPROVAL_REPLY"
	case Winner:
		return "WINNER"
	}
	return fmt.Sprintf("MsgType(%d)", int(t))
}

// Message to treść wiadomości. Model CONGEST: jedna wiadomość ma O(log n) bitów,
// więc trzymamy tu tylko typ i jedną liczbę (RANK <= n^4 mieści się w uint64).
type Message struct {
	Type MsgType
	Rank uint64
}

func (m Message) String() string {
	return fmt.Sprintf("%v(%d)", m.Type, m.Rank)
}

// Incoming to wiadomość odebrana przez wierzchołek: wie on tylko, z którego
// swojego portu ją dostał, a nie kto jest nadawcą (nie zna identyfikatorów).
type Incoming struct {
	Port int
	Msg  Message
}

// Outgoing to wiadomość, którą wierzchołek chce wysłać przez dany port.
type Outgoing struct {
	Port int
	Msg  Message
}
