package main

import (
	"debug/elf"
	"fmt"
	"os"
	"strings"
)

func die(s string, i ...interface{}) {
	s = fmt.Sprintf(s, i...)
	fmt.Fprintln(os.Stderr, strings.TrimSuffix(s, "\n"))
	os.Exit(1)
}

func main() {
	if len(os.Args) <= 1 {
		die("elf file needed")
	}
	f, err := elf.Open(os.Args[1])
	if err != nil {
		die("error: %v\n", err)
	}
	libs, err := f.ImportedLibraries()
	if err != nil {
		die("error: %v", err)
	}
	for _, lib := range libs {
		fmt.Println(lib)
	}
}
