package main

import "fmt"

func main() {
	var cache LRU[string, string]
	cache = NewLRUCache[string, string](2)
	cache.Set("a", "1")

	if val, ok := cache.Get("a"); ok {
		fmt.Println("Got:", val) // Should print "Got: 1"
	} else {
		fmt.Println("Key 'a' not found")
	}

}
