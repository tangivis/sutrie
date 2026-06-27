package benchmark

import (
	radix "github.com/armon/go-radix"
	dptrie "github.com/derekparker/trie"
	"github.com/nobekanai/sutrie"
	"github.com/openacid/slim/encode"
	slim "github.com/openacid/slim/trie"
)

// impl describes one string-set implementation under test. build constructs
// the structure from the given sorted keys and returns it together with a
// membership-test closure.
type impl struct {
	name string
	// exact reports whether the membership test never returns false positives
	// for absent keys. SlimTrie is a lossy succinct index (it does not retain
	// full keys) and may report an absent key as present, so it is excluded
	// from the "absent" correctness benchmark.
	exact bool
	build func(keys []string) (obj any, lookup func(string) bool)
}

// slimEncoder returns the value encoder used to build SlimTrie instances.
func slimEncoder() encode.Encoder { return encode.I32{} }

func implementations() []impl {
	return []impl{
		{
			name:  "map",
			exact: true,
			build: func(keys []string) (any, func(string) bool) {
				m := make(map[string]struct{}, len(keys))
				for _, k := range keys {
					m[k] = struct{}{}
				}
				return m, func(s string) bool { _, ok := m[s]; return ok }
			},
		},
		{
			name:  "sutrie",
			exact: true,
			build: func(keys []string) (any, func(string) bool) {
				t := sutrie.BuildSuccinctTrie(keys)
				root := t.Root()
				return t, func(s string) bool { return root.Search(s).Leaf() }
			},
		},
		{
			name:  "go-radix",
			exact: true,
			build: func(keys []string) (any, func(string) bool) {
				t := radix.New()
				for _, k := range keys {
					t.Insert(k, struct{}{})
				}
				return t, func(s string) bool { _, ok := t.Get(s); return ok }
			},
		},
		{
			name:  "derekparker-trie",
			exact: true,
			build: func(keys []string) (any, func(string) bool) {
				t := dptrie.New()
				for _, k := range keys {
					t.Add(k, nil)
				}
				return t, func(s string) bool { _, ok := t.Find(s); return ok }
			},
		},
		{
			name:  "slimtrie",
			exact: false,
			build: func(keys []string) (any, func(string) bool) {
				vals := make([]int32, len(keys))
				for i := range vals {
					vals[i] = int32(i)
				}
				t, err := slim.NewSlimTrie(encode.I32{}, keys, vals)
				if err != nil {
					panic(err)
				}
				return t, func(s string) bool { _, ok := t.Get(s); return ok }
			},
		},
	}
}
