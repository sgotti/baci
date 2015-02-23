package main

import (
	"bufio"
	"bytes"
	"debug/elf"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

var (
	libRegexp = regexp.MustCompile(`^\s*(lib.*)\s+=>\s+(.*)\s+.*$`)
)

func die(s string, i ...interface{}) {
	s = fmt.Sprintf(s, i...)
	fmt.Fprintln(os.Stderr, strings.TrimSuffix(s, "\n"))
	os.Exit(1)
}

func getLibsPaths(elffile string) ([]string, error) {
	cmd := exec.Command("ldd", elffile)
	out, err := cmd.Output()
	if err != nil {
		die("error: %v", err)
	}

	paths := []string{}
	r := bytes.NewReader(out)
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		m := libRegexp.FindStringSubmatch(scanner.Text())
		if m != nil && len(m) >= 3 {
			path := m[2]
			if path != "" {
				paths = append(paths, path)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading command output:", err)
	}

	return paths, nil
}

func getLdPath(elffile string) (string, error) {
	f, err := elf.Open(elffile)
	if err != nil {
		die("error: %v\n", err)
	}
	defer f.Close()
	for _, p := range f.Progs {
		if p.ProgHeader.Type == elf.PT_INTERP {
			v, err := ioutil.ReadAll(p.Open())
			if err != nil {
				return "", fmt.Errorf("%v", err)
			}
			return string(v), nil
		}
	}
	return "", fmt.Errorf("cannot find interpreter")
}

func main() {
	if len(os.Args) <= 2 {
		die(`usage: command elffile`)
	}

	switch os.Args[1] {
	case "ldpath":
		ldpath, err := getLdPath(os.Args[2])
		if err != nil {
			die("error: %v", err)
		}
		fmt.Println(ldpath)
	case "libs":
		paths, err := getLibsPaths(os.Args[2])
		if err != nil {
			die("error: %v", err)
		}
		for _, path := range paths {
			fmt.Println(path)
		}
	default:
		die("wrong command: %q", os.Args[1])
	}
}
