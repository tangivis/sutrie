// Package benchmark contains a self-contained, reproducible comparison of
// sutrie against other well-known Go string-set / trie implementations.
//
// It lives in its own Go module so that the heavyweight comparison
// dependencies (slim, go-radix, derekparker/trie) never leak into the
// dependency graph of the sutrie library itself.
package benchmark

import "sort"

// Datasets are generated deterministically with a tiny self-contained LCG so
// that results are reproducible across machines and Go versions without
// relying on math/rand's (version-dependent) sequence or any external file.
type lcg struct{ state uint64 }

func newLCG(seed uint64) *lcg { return &lcg{state: seed*2862933555777941757 + 3037000493} }

func (r *lcg) next() uint64 {
	// numerical recipes constants
	r.state = r.state*6364136223846793005 + 1442695040888963407
	return r.state
}

func (r *lcg) intn(n int) int { return int(r.next()>>33) % n }

// genWords returns n unique, sorted, lowercase-ascii "word" keys of length
// 3..18. This models a general dictionary / token set.
func genWords(n int) []string {
	const alpha = "abcdefghijklmnopqrstuvwxyz"
	r := newLCG(0xC0FFEE)
	seen := make(map[string]struct{}, n*2)
	out := make([]string, 0, n)
	for len(out) < n {
		l := 3 + r.intn(16)
		b := make([]byte, l)
		for i := range b {
			b[i] = alpha[r.intn(len(alpha))]
		}
		s := string(b)
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// genDomains returns n unique, sorted, reversed domain-name keys
// (e.g. "com.example.www"), which is the use case sutrie was designed for.
func genDomains(n int) []string {
	const alpha = "abcdefghijklmnopqrstuvwxyz0123456789"
	tlds := []string{"com", "net", "org", "io", "co", "top", "dev", "app"}
	r := newLCG(0xBEEF)
	seen := make(map[string]struct{}, n*2)
	out := make([]string, 0, n)
	label := func() string {
		l := 2 + r.intn(10)
		b := make([]byte, l)
		for i := range b {
			b[i] = alpha[r.intn(len(alpha))]
		}
		return string(b)
	}
	for len(out) < n {
		// reversed: tld first, then 1..3 labels
		s := tlds[r.intn(len(tlds))]
		for parts := 1 + r.intn(3); parts > 0; parts-- {
			s += "." + label()
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// absentKeys returns n sorted keys guaranteed not to be in present.
func absentKeys(present []string, n int) []string {
	const alpha = "abcdefghijklmnopqrstuvwxyz"
	set := make(map[string]struct{}, len(present))
	for _, k := range present {
		set[k] = struct{}{}
	}
	r := newLCG(0x1234567)
	out := make([]string, 0, n)
	for len(out) < n {
		l := 3 + r.intn(16)
		b := make([]byte, l)
		for i := range b {
			b[i] = alpha[r.intn(len(alpha))]
		}
		s := string(b)
		if _, ok := set[s]; ok {
			continue
		}
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
