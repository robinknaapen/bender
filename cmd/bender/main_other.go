//go:build !windows && !linux

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "bender: this platform is not supported yet (build with GOOS=windows or GOOS=linux)")
	os.Exit(1)
}
