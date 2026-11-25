package main

import "math/rand/v2"

// Mapujemy porty do konkretnych wierzcholkow
// sigma to permutacja ktorej uzywamy, zupelnie losowa
// pod portem p wierzcholka v stoi (v + 1 + sigma[p]) mod n - jakis wierzcholek poza v
// inv po to zeby wiedziec ktorym portem wyslac wiadomosc
type PortMap struct {
	n     int
	sigma []int // port -> przesunięcie
	inv   []int // przesunięcie -> port
}

func NewPortMap(n int, adversary *rand.Rand) *PortMap {
	sigma := adversary.Perm(n - 1)
	inv := make([]int, n-1)
	for p, d := range sigma {
		inv[d] = p
	}
	return &PortMap{n: n, sigma: sigma, inv: inv}
}

func (pm *PortMap) Neighbor(v, port int) int {
	return (v + 1 + pm.sigma[port]) % pm.n
}

func (pm *PortMap) Port(v, u int) int {
	d := ((u-v-1)%pm.n + pm.n) % pm.n
	return pm.inv[d]
}
