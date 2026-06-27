package benchmark

import (
	"flag"
	"testing"
)

// keyCount controls dataset size; override with -keys.
var keyCount = flag.Int("keys", 200000, "number of keys in the benchmark dataset")

// dataset selects the key generator; override with -dataset words|domains.
var dataset = flag.String("dataset", "words", "dataset to use: words or domains")

func genKeys() []string {
	if *dataset == "domains" {
		return genDomains(*keyCount)
	}
	return genWords(*keyCount)
}

// BenchmarkBuild measures construction time and allocations for each impl.
func BenchmarkBuild(b *testing.B) {
	keys := genKeys()
	for _, im := range implementations() {
		im := im
		b.Run(im.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				obj, _ := im.build(keys)
				_ = obj
			}
		})
	}
}

// BenchmarkLookupHit measures membership lookups for keys that are present.
func BenchmarkLookupHit(b *testing.B) {
	keys := genKeys()
	for _, im := range implementations() {
		im := im
		_, lookup := im.build(keys)
		b.Run(im.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if !lookup(keys[i%len(keys)]) {
					b.Fatalf("%s: expected hit for present key", im.name)
				}
			}
		})
	}
}

// BenchmarkLookupMiss measures membership lookups for absent keys. SlimTrie is
// skipped because it is a lossy index that can report false positives.
func BenchmarkLookupMiss(b *testing.B) {
	keys := genKeys()
	absent := absentKeys(keys, 50000)
	for _, im := range implementations() {
		im := im
		if !im.exact {
			continue
		}
		_, lookup := im.build(keys)
		b.Run(im.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if lookup(absent[i%len(absent)]) {
					b.Fatalf("%s: expected miss for absent key", im.name)
				}
			}
		})
	}
}
