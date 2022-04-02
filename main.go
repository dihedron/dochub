package main

import (
	"context"
	"fmt"
	"log"

	"github.com/dihedron/dochub/index"
)

func main() {
	if entry, err := index.Load(context.Background(), "test/manifest.json"); err != nil {
		log.Fatalf("error: %v", err)
	} else {
		fmt.Printf("entry: %v\n", entry)
	}
}
