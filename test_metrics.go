package main

import (
	"fmt"
	"expvar"
)

func main() {
	// Test that expvar works
	m := expvar.NewInt("test_counter")
	m.Add(1)
	
	// Print all expvars
	expvar.Do(func(kv expvar.KeyValue) {
		fmt.Printf("%s: %s\n", kv.Key, kv.Value)
	})
}