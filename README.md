# Singularly Optimal Randomized Leader Election: wersja synchroniczna

Implementacja algorytmu z Sekcji 3 pracy Kutten, Moses Jr., Pandurangan,
Peleg, *Singularly Optimal Randomized Leader Election* (DISC 2020).

## Struktura

| Plik           | Zawartość                                                                                                  |
| -------------- | ---------------------------------------------------------------------------------------------------------- |
| `main.go`      | flagi, generowanie sieci i planu budzenia, uruchomienie, podsumowanie                                      |
| `network.go`   | synchroniczna sieć: jedna gorutyna = jeden wierzchołek, kanały z buforem 1, bariera rund, logi, statystyki |
| `ports.go`     | przyporządkowanie portów do sąsiadów (ustalane przez przeciwnika *oblivious*)                              |
| `wakeup.go`    | plan budzenia przez przeciwnika (`-wake all\|one\|random`)                                                 |
| `message.go`   | typy wiadomości: `APPROVAL_REQUEST`, `APPROVAL_REPLY`, `WINNER`                                            |
| `node.go`      | **stan wierzchołka (`LeaderNode`) i kod jednej rundy (`Round`)**: tu jest algorytm                         |
| `adversary.go` | adwersarz interaktywny (`-interactive`)                                                                    |

Cały algorytm jest w `node.go`; reszta to sieć, adwersarz i uruchamianie.

## Model w kodzie

- Wierzchołek zna tylko swoje porty `0..n-2`, a nie identyfikatory sąsiadów.
  Pole `id` służy wyłącznie do logów.
- Wiadomość wysłana w rundzie `r` dociera na końcu rundy `r`, więc wierzchołek widzi ją w `inbox` w rundzie `r+1`.
- Każdy wierzchołek ma własny generator losowy (prywatna moneta).
- Struktura portów i plan budzenia są ustalane przed losowaniami wierzchołków.

## Uruchomienie

Wymagany Go zgodny z wersją w `go.mod`.

```sh
go run .                                  # n=16, przeciwnik budzi 4 losowe wierzchołki w rundach 0..4
go run . -n 100 -wake all                 # wszystkie wierzchołki budzone w rundzie 0
go run . -n 100 -wake one                 # tylko jeden wierzchołek
go run . -n 1000 -wake random -k 200 -T 10 -v=false
go run . -n 8 -wake all -delay 1s         # śledzenie na żywo, 1 s między rundami
go test ./...                             # testy
```

### Przeciwnik interaktywny (adaptive)

```sh
go run . -n 8 -interactive
```

Przed każdą rundą program pyta, które wierzchołki obudzić:

- `0 3 5` lub `0,3,5`: obudź te wierzchołki,
- Enter: nie budź nikogo w tej rundzie,
- `s`: pokaż stan wszystkich wierzchołków (rola, RANK, lider), czyli to, co wie przeciwnik *adaptive*,
- `q`: zakończ budzenie (program dalej działa bez pytań).

Jeśli wskazany wierzchołek już nie śpi (obudził go przeciwnik albo wiadomość), program o tym informuje i go pomija. Pytania znikają, gdy nie ma już śpiących wierzchołków.

### Flagi

| Flaga | Domyślnie | Znaczenie |
|---|---|---|
| `-n` | 16 | liczba wierzchołków (2..65535, żeby `n^4` mieściło się w `uint64`) |
| `-wake` | `random` | `all`, `one` lub `random` |
| `-k`, `-T` | 4, 5 | tryb `random`: `k` wierzchołków budzonych w losowych rundach z `[0, T)` |
| `-interactive` | false | przeciwnik interaktywny: przed każdą rundą pyta, kogo obudzić (ignoruje `-wake`) |
| `-seed` | 0 | seed generatora losowego; `0` oznacza losowy seed |
| `-delay` | 0 | opóźnienie między rundami, np. `500ms` |
| `-v` | true | logi `runda N rozpoczęta` i `X wysłał M do Y` |
| `-maxrounds` | 100 | limit rund symulacji |

### Wyjście

Przy `-v` program loguje każdą rundę i każdą wiadomość, a na końcu wypisuje:

- liczbę rund symulacji,
- **czas algorytmu**: rundy od pierwszego obudzenia do ostatniej wiadomości (miara z paperu; teoria mówi, że jest ich najwyżej 9),
- **liczbę wiadomości**: łącznie, na wierzchołek i według typu,
- sprawdzenie specyfikacji, czyli czy wszystkie wierzchołki znają tego samego lidera.
