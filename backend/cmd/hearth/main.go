package main

import (
	"fmt"
	"os"
)

var (
	Version = "dev"
	Commit  = "unknown"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Printf("Hearth %s (%s)\n", Version, Commit)
		return
	}

	fmt.Println("🔥 Hearth - Self-hosted Discord alternative")
	fmt.Printf("Version: %s (%s)\n", Version, Commit)
	fmt.Println()
	fmt.Println("Coming soon! See docs/ for design details.")
}
