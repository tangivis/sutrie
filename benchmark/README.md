# Benchmarks

A reproducible comparison of **sutrie** against other well-known Go
string-set / trie implementations, covering both **space usage** and
**time** (build + lookup).

This lives in a **separate Go module** so the comparison dependencies never
leak into the sutrie library's own dependency graph.

## Implementations compared

| Name               | Package                          | Kind                         | Exact membership? |
|--------------------|----------------------------------|------------------------------|-------------------|
| `map`              | built-in `map[string]struct{}`   | hash set (baseline)          | yes               |
| `sutrie`           | this repo                        | succinct (LOUDS) trie set    | yes               |
| `go-radix`         | `github.com/armon/go-radix`      | radix tree                   | yes               |
| `derekparker-trie` | `github.com/derekparker/trie`    | pointer/map trie             | yes               |
| `slimtrie`         | `github.com/openacid/slim`       | succinct key→value index     | **no** (lossy)¹   |

¹ SlimTrie is famous for its tiny footprint, but it does **not** retain the
original keys: it is a lossy index that can report an *absent* key as present
(a false positive). It is therefore excluded from the "miss" lookup benchmark,
and it is always built with **distinct** values — with identical values it
collapses to a trivial constant trie that answers `true` for everything. It
also stores a value per key (here a 4-byte `int32`), which sutrie, a pure set,
does not.

## Running

```bash
cd benchmark

# Space / footprint table (in-memory + serialized):
go test -run TestSpaceComparison -v -keys 200000 -dataset words
go test -run TestSpaceComparison -v -keys 200000 -dataset domains

# Time (build + lookup hit/miss):
go test -run '^$' -bench . -benchmem -keys 200000 -dataset words
```

Flags: `-keys N` (dataset size), `-dataset words|domains`. Datasets are
generated deterministically by a tiny built-in LCG, so runs are reproducible
and need no external files.

### How space is measured

`TestSpaceComparison` reports the heap bytes each structure **retains after the
input key slice is released** (`runtime.MemStats.HeapAlloc`, delta around a
forced GC). This models real usage: build the index, discard the input, keep
the structure to serve queries. Implementations that hold references to the
original key strings (`map`, `go-radix`, `derekparker-trie`) keep those bytes
alive; `sutrie`/`slimtrie` copy keys into a compact encoding and let the
originals be collected.

## Results

Machine: linux/amd64, 4 vCPU, Go 1.24. Numbers vary by machine — reproduce
locally with the commands above. Dataset: 200,000 unique keys.

### Space — in-memory footprint (words, raw key bytes ≈ 2.13 MB)

| impl               | in-mem bytes | bytes/key | vs sutrie |
|--------------------|-------------:|----------:|----------:|
| slimtrie¹          |    1,277,896 |      6.39 |    0.52×  |
| **sutrie**         |    2,474,176 |     12.37 |    1.00×  |
| map                |   10,198,992 |     50.99 |    4.12×  |
| go-radix           |   25,772,848 |    128.86 |   10.42×  |
| derekparker-trie   |  469,510,000 |   2347.55 |  189.77×  |

`sutrie` uses **~4× less memory than the built-in map**, ~10× less than
go-radix, and ~190× less than a pointer/map trie. SlimTrie is smaller still,
but is a lossy key→value index (see note ¹) rather than an exact set.

### Space — serialized / on-disk (the two impls with native marshalling)

| impl       | bytes      | bytes/key |
|------------|-----------:|----------:|
| slimtrie¹  |  1,064,563 |      5.32 |
| **sutrie** |  2,128,947 |     10.64 |

`sutrie` serializes to ~10.6 bytes/key — close to the raw key bytes — via
`(*SuccinctTrie).Marshal`. (`map`, `go-radix`, `derekparker-trie` have no
built-in serialization.)

### Time — build (words, 200k keys)

| impl               | ns/op       | B/op        | allocs/op |
|--------------------|------------:|------------:|----------:|
| map                |  13,367,855 |   6,990,252 |       516 |
| **sutrie**         |  36,950,001 |  15,443,481 |    **90** |
| go-radix           |  56,834,088 |  32,289,220 |   815,906 |
| slimtrie           | 144,946,305 | 123,249,516 | 1,244,103 |
| derekparker-trie   | 356,433,283 | 468,359,344 | 4,969,067 |

`sutrie` builds with **far fewer allocations than any other trie** (90 vs
hundreds of thousands to millions), and faster than go-radix / slimtrie /
derekparker-trie. It is slower than a raw map insert, as expected.

### Time — lookup (words, 200k keys)

| impl               | hit ns/op | miss ns/op |
|--------------------|----------:|-----------:|
| map                |     44.7  |      54.3  |
| go-radix           |     90.7  |     114.7  |
| slimtrie¹          |    160.3  |      n/a   |
| derekparker-trie   |    379.3  |     240.7  |
| **sutrie**         |    502.6  |     181.7  |

Lookups are sutrie's **deliberate trade-off**: the succinct `select`/`rank`
machinery that makes it so compact costs cache misses on a *hit*, where it
trails the pointer-chasing structures. On a *miss* it terminates early and is
competitive. If you need the absolute fastest lookups and have memory to
spare, a `map` wins; if you need a compact, serializable, exact set, sutrie is
the sweet spot.

> On the `domains` dataset the rankings are the same; sutrie's miss lookups are
> actually the fastest non-map option (≈25 ns) because mismatches are rejected
> at the first differing byte. Run the commands above to see the full table.

## Takeaways

- **Memory is sutrie's headline:** ~4× smaller than a Go map and 10–190×
  smaller than common Go tries, while remaining an *exact* set.
- **Cheap to build:** essentially constant allocation count regardless of key
  count.
- **Serializable:** native `Marshal`/`Unmarshal` at ~key-size on disk.
- **Lookup is the trade-off:** slower per-hit than pointer tries / maps, very
  competitive on misses. Pick sutrie when footprint and serialization matter
  more than raw lookup throughput.
