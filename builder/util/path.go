package util

import (
	"os"
	"os/exec"
	"path/filepath"
)

func GetFullPath(filename string, pathEnv string) (string, error) {
	savedPathEnv := os.Getenv("PATH")
	os.Setenv("PATH", pathEnv)
	fp, err := exec.LookPath(filename)
	os.Setenv("PATH", savedPathEnv)
	if err != nil {
		return "", err
	}
	// If the path is relative to the current workdir make it absolute.
	if !filepath.IsAbs(fp) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		fp = filepath.Join(wd, fp)
	}
	return fp, nil
}
