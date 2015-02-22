package acibuilder

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/aci"
	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/schema"
)

// SimpleACIBuilder is an ACIBuilder that creates an ACI containing the
// file in the provided path (which should be the rootfs)
// plus the provided imagemanifest without any change.
type SimpleACIBuilder struct {
	path        string
	excludeFunc ExcludeFunc
}

func NewSimpleACIBuilder(path string) *SimpleACIBuilder {
	return &SimpleACIBuilder{path: path}
}

func (b *SimpleACIBuilder) SetExcludeFunc(excludeFunc ExcludeFunc) {
	b.excludeFunc = excludeFunc
}

func (b *SimpleACIBuilder) Build(im schema.ImageManifest, out io.Writer) error {
	gw := gzip.NewWriter(out)
	tr := tar.NewWriter(gw)
	defer func() {
		tr.Close()
		gw.Close()
	}()

	aw := aci.NewImageWriter(im, tr)

	err := filepath.Walk(b.path, BuildWalker(b.path, nil, b.excludeFunc, aw))
	if err != nil {
		return fmt.Errorf("error walking rootfs: %v", err)
	}

	err = aw.Close()
	if err != nil {
		return fmt.Errorf("unable to close image %s: %v", out, err)
	}

	return nil
}
