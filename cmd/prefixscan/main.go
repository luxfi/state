package main

import (
    "encoding/hex"
    "fmt"
    "os"
    "sort"

    "github.com/cockroachdb/pebble"
)

func main() {
    if len(os.Args) != 2 {
        fmt.Println("usage: prefixscan </full/path/to/db>")
        return
    }
    db, _ := pebble.Open(os.Args[1], &pebble.Options{ReadOnly: true})
    defer db.Close()

    type kv struct{ pref string; n int }
    count := map[string]int{}
    it, _ := db.NewIter(nil)
    for it.First(); it.Valid(); it.Next() {
        if len(it.Key()) < 33 { continue }
        p := hex.EncodeToString(it.Key()[:33]) // 32B ID + 00 delimiter
        count[p]++
    }
    var list []kv
    for p, n := range count { list = append(list, kv{p, n}) }
    sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })

    fmt.Printf("%-70s  keys\n", "prefix (hex)")
    for i := 0; i < 10 && i < len(list); i++ {
        fmt.Printf("%s  %d\n", list[i].pref, list[i].n)
    }
}