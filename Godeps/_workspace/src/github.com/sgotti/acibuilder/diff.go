package acibuilder

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/aci"
	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/schema"
	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/sgotti/fsdiffer"
)

// DiffACIBuilder is an ACIBuilder that creates an ACI containing only the
// different files between a base and a new ACI's rootfs.
// basePath and path are the paths of the rootfs of the two already extracted ACIs.
// If there are deleted files from the base ACI, the imagemanifest will be
// augmented with a pathWhiteList containing all the ACI's files
type DiffACIBuilder struct {
	basePath    string
	path        string
	excludeFunc ExcludeFunc
}

func NewDiffACIBuilder(basePath string, path string) *DiffACIBuilder {
	return &DiffACIBuilder{basePath: basePath, path: path}
}

func (b *DiffACIBuilder) SetExcludeFunc(excludeFunc ExcludeFunc) {
	b.excludeFunc = excludeFunc
}

func (b *DiffACIBuilder) Build(im schema.ImageManifest, out io.Writer) error {

	fsd := fsdiffer.NewSimpleFSDiffer(b.basePath, b.path)

	changes, err := fsd.Diff()

	if err != nil {
		return err
	}

	// Create a file list with all the Added and Modified files
	files := make(map[string]struct{})
	hasDeleted := false
	for _, c := range changes {
		if c.ChangeType == fsdiffer.Added || c.ChangeType == fsdiffer.Modified {
			files[filepath.Join(b.path, c.Path)] = struct{}{}
		}
		if hasDeleted == false && c.ChangeType == fsdiffer.Deleted {
			hasDeleted = true
		}
	}

	// Compose pathWhiteList only if there're some deleted files
	pathWhitelist := []string{}
	if hasDeleted {
		err = filepath.Walk(b.path, func(path string, info os.FileInfo, err error) error {
			relpath, err := filepath.Rel(b.path, path)
			if err != nil {
				return err
			}
			pathWhitelist = append(pathWhitelist, filepath.Join("/", relpath))
			return nil
		})

		if err != nil {
			return fmt.Errorf("error walking rootfs: %v", err)
		}
	}

	gw := gzip.NewWriter(out)
	tr := tar.NewWriter(gw)
	defer func() {
		tr.Close()
		gw.Close()
	}()

	im.PathWhitelist = pathWhitelist

	aw := aci.NewImageWriter(im, tr)

	err = filepath.Walk(b.path, BuildWalker(b.path, files, b.excludeFunc, aw))
	if err != nil {
		return fmt.Errorf("error walking rootfs: %v", err)
	}

	err = aw.Close()
	if err != nil {
		return fmt.Errorf("unable to close image %s: %v", out, err)
	}

	return nil
}
