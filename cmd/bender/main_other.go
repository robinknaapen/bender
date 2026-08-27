//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "bender: only Windows is supported so far (build with GOOS=windows)")
	os.Exit(1)
}
