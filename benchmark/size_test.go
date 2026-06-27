package benchmark

import (
	"bytes"
	"fmt"
	"runtime"
	"testing"

	"github.com/nobekanai/sutrie"
	"github.com/openacid/slim/trie"
)

// retainedHeap builds a structure and returns the number of heap bytes it
// retains once the original key slice has been released. This reflects the
// real-world footprint: you build the index, discard the input, and keep the
// structure around to serve queries. Implementations that keep references to
// the original key strings (map, go-radix, derekparker/trie) therefore keep
// those bytes alive, while sutrie/slimtrie copy keys into a compact encoding
// and let the originals be collected.
func retainedHeap(t *testing.T, build func(keys []string) any) uint64 {
	t.Helper()
	runtime.GC()
	runtime.GC()
	var m0 runtime.MemStats
	runtime.ReadMemStats(&m0)

	keys := genKeys()
	obj := build(keys)
	keys = nil // allow the input to be collected; only the structure should remain

	runtime.GC()
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)
	runtime.KeepAlive(obj)

	if m1.HeapAlloc < m0.HeapAlloc {
		return 0
	}
	return m1.HeapAlloc - m0.HeapAlloc
}

// TestSpaceComparison prints an in-memory and on-disk footprint table. It is
// not a pass/fail test; run it with -v to see the numbers:
//
//	go test -run TestSpaceComparison -v
//	go test -run TestSpaceComparison -v -keys 1000000 -dataset domains
func TestSpaceComparison(t *testing.T) {
	n := *keyCount

	// Raw size of the key data, for reference.
	var raw uint64
	for _, k := range genKeys() {
		raw += uint64(len(k))
	}

	fmt.Printf("\n=== Space comparison: dataset=%s keys=%d rawKeyBytes=%d ===\n", *dataset, n, raw)
	fmt.Printf("%-18s %14s %12s\n", "impl", "in-mem bytes", "bytes/key")

	for _, im := range implementations() {
		im := im
		bytesUsed := retainedHeap(t, func(keys []string) any {
			obj, _ := im.build(keys)
			return obj
		})
		fmt.Printf("%-18s %14d %12.2f\n", im.name, bytesUsed, float64(bytesUsed)/float64(n))
	}

	// Serialized (on-disk) footprint, for the two impls that support it.
	fmt.Printf("\n%-18s %14s %12s\n", "impl (serialized)", "bytes", "bytes/key")
	keys := genKeys()

	var buf bytes.Buffer
	if err := sutrie.BuildSuccinctTrie(append([]string(nil), keys...)).Marshal(&buf); err != nil {
		t.Fatalf("sutrie marshal: %v", err)
	}
	fmt.Printf("%-18s %14d %12.2f\n", "sutrie", buf.Len(), float64(buf.Len())/float64(n))

	// SlimTrie must be given distinct values, otherwise it collapses to a
	// trivial constant trie (and reports every key as present).
	vals := make([]int32, len(keys))
	for i := range vals {
		vals[i] = int32(i)
	}
	st, err := trie.NewSlimTrie(slimEncoder(), keys, vals)
	if err != nil {
		t.Fatalf("slim build: %v", err)
	}
	if b, err := st.Marshal(); err != nil {
		t.Fatalf("slim marshal: %v", err)
	} else {
		fmt.Printf("%-18s %14d %12.2f\n", "slimtrie", len(b), float64(len(b))/float64(n))
	}
	fmt.Println()
}
