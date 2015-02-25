package acibuilder

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/aci"
	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/pkg/tarheader"
)

// function called to check if the given path should be excluded. path is
// relative to the root dir given to BuildWalker (ex. usr/bin/ls)
type ExcludeFunc func(path string, info os.FileInfo) (bool, error)

// BuildWalker creates a filepath.WalkFunc that walks over the given root
// (which is the rootfs of the ACI on disk, NOT the ACI layout on disk) and
// adds the files in the directory to the given ArchiveWriter
// If excludeFunc is not nil then it's called for every path
func BuildWalker(root string, files map[string]struct{}, excludeFunc ExcludeFunc, aw aci.ArchiveWriter) filepath.WalkFunc {
	// cache of inode -> filepath, used to leverage hard links in the archive
	inos := map[uint64]string{}
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Ignore Walk errors
			return nil
		}

		if files != nil {
			if _, ok := files[path]; !ok {
				return nil
			}
		}

		relpath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relpath == "." {
			return nil
		}

		if excludeFunc != nil {
			exclude, err := excludeFunc(relpath, info)
			if err != nil {
				return err
			}
			if exclude {
				return nil
			}
		}

		link := ""
		var r io.Reader
		switch info.Mode() & os.ModeType {
		case os.ModeCharDevice:
		case os.ModeDevice:
		case os.ModeDir:
		case os.ModeSymlink:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			link = target
		default:
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()
			r = file
		}

		hdr, err := tar.FileInfoHeader(info, link)
		if err != nil {
			panic(err)
		}
		// Because os.FileInfo's Name method returns only the base
		// name of the file it describes, it may be necessary to
		// modify the Name field of the returned header to provide the
		// full path name of the file.
		hdr.Name = filepath.Join("rootfs", relpath)
		tarheader.Populate(hdr, info, inos)
		// If the file is a hard link to a file we've already seen, we
		// don't need the contents
		if hdr.Typeflag == tar.TypeLink {
			hdr.Size = 0
			r = nil
		}
		if err := aw.AddFile(hdr, r); err != nil {
			return err
		}
		return nil
	}
}
