# Succinct Trie

"Sutrie" is an extremely memory-efficient prefix tree or trie data structure, which also boasts impressive time
efficiency.

## Install

```bash
go get -u github.com/nobekanai/sutrie
```

## Benchmark

`sutrie` is built for **memory efficiency**. On a 200,000-key dataset it uses
roughly **4× less memory than Go's built-in `map`**, ~10× less than
`armon/go-radix`, and ~190× less than `derekparker/trie`, while staying an
*exact* membership set and supporting serialization:

| impl (in-memory)   | bytes/key | vs sutrie |
|--------------------|----------:|----------:|
| `slimtrie`¹        |      6.39 |    0.52×  |
| **`sutrie`**       |     12.37 |    1.00×  |
| built-in `map`     |     50.99 |    4.12×  |
| `go-radix`         |    128.86 |   10.42×  |
| `derekparker/trie` |   2347.55 |  189.77×  |

¹ `slimtrie` (`openacid/slim`) is smaller still but is a *lossy* key→value
index that can false-positive on absent keys, whereas `sutrie` is an exact set.

The trade-off is lookup throughput: sutrie's succinct `rank`/`select` machinery
is slower per hit than pointer-based tries or a `map`. See
[`benchmark/`](benchmark/) for the full methodology, the time benchmarks, and
instructions to reproduce the numbers on your own machine.

## Documentation

A simple and common use case: querying whether the key appears in the dictionary or whether the prefix of the key is in
the dictionary

```go
keys := []string{"hat", "is", "it", "a"}
trie := sutrie.BuildSuccinctTrie(keys).Root()

lastUnmatch := trie.SearchPrefix("hatt")
println(lastUnmatch) // will print 3, same if you search "hat"

lastUnmatch = trie.SearchPrefix("ha")
println(lastUnmatch) // will print 0, because neither "ha" nor its prefix (that is "h") is in trie
```

### Advanced Usage

You can customize trie's traversal process. For example, define a domain name lookup rule: `*.example.com` matches
`www.example.com` and `xxx.example.com` but not `xxx.www.example.com` and `example.com`

```go
func reverse(s string) string {
	bytes := []byte(s)
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}
	return string(bytes)
}

func reverseSplit(domain string) []string {
	return strings.Split(reverse(domain), ".")
}

func main() {
	// First build a trie with reversed domains because we match domains backwards
	domains := []string{reverse("*.example.com"), reverse("google.*")}
	trie := sutrie.BuildSuccinctTrie(domains)

	// Then let's define matching rules
	var search func(node sutrie.Node, ds []string, idx int) bool
	search = func(node sutrie.Node, ds []string, idx int) bool {
		if !node.Exists() {
			return false
		}

		wildcard := node.Next('*')

		if idx == len(ds)-1 {
			return wildcard.Leaf() || node.Search(ds[idx]).Leaf()
		}

		if wildcard.Exists() && search(wildcard.Next('.'), ds, idx+1) {
			return true
		}

		return search(node.Search(ds[idx]).Next('.'), ds, idx+1)
	}

	Search := func(domain string) bool {
		return search(trie.Root(), reverseSplit(domain), 0)
	}

	// Finally we apply the rules to search for domain names
	println(Search("www.example.com"))     // true
	println(Search("xxx.example.com"))     // true
	println(Search("xxx.www.example.com")) // false
	println(Search("example.com"))         // false
	println(Search("google.io"))           // true
}
```
