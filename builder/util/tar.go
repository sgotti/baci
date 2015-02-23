package util

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/aci"
	"github.com/sgotti/baci/common"

	ptar "github.com/sgotti/baci/Godeps/_workspace/src/github.com/sgotti/acido/pkg/tar"
)

var (
	ldpath string
	xzPath = filepath.Join(common.BaciRootDir, "/usr/bin/xz")
)

var (
	// lib/ contains the xz needed libs, it's named just lib as the host's
	// libs can be placed in different places (/lib64 on fedora,
	// /lib/x86_64-linux-gnu/ on debian/ubuntu etc...)
	baciRootEnv = []string{"PATH=" + filepath.Join(common.BaciRootDir, "usr/bin"), "LD_LIBRARY_PATH=" + filepath.Join(common.BaciRootDir, "lib")}
)

func init() {

}
func decompress(rs io.Reader, typ aci.FileType) (io.Reader, error) {
	var (
		dr  io.Reader
		err error
	)
	switch typ {
	case aci.TypeGzip:
		dr, err = gzip.NewReader(rs)
		if err != nil {
			return nil, err
		}
	case aci.TypeBzip2:
		dr = bzip2.NewReader(rs)
	case aci.TypeXz:
		if ldpath == "" {
			return nil, fmt.Errorf("ld.so path not defined. Cannot extract xz files.")
		}
		dr = xzReader(rs)
	case aci.TypeTar:
		dr = rs
	case aci.TypeUnknown:
		return nil, errors.New("error: unknown image filetype")
	default:
		return nil, errors.New("no type returned from DetectFileType?")
	}
	return dr, nil
}

// xzReader shells out to a command line xz executable (if
// available) to decompress the given io.Reader using the xz
// compression format
func xzReader(r io.Reader) io.ReadCloser {
	rpipe, wpipe := io.Pipe()
	cmd := exec.Command(filepath.Join(common.BaciRootDir, ldpath), xzPath, "--decompress", "--stdout")
	cmd.Stdin = r
	cmd.Stdout = wpipe
	cmd.Env = baciRootEnv

	go func() {
		err := cmd.Run()
		wpipe.CloseWithError(err)
	}()

	return rpipe
}

func ExtractACI(source string, dir string) error {
	r, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("cannot open source file: %v", err)
	}
	// Peek at the first 512 bytes of the reader to detect filetype
	br := bufio.NewReaderSize(r, 512)
	hd, err := br.Peek(512)
	switch err {
	case nil:
	case io.EOF: // We may have still peeked enough to guess some types, so fall through
	default:
		return fmt.Errorf("error reading image header: %v", err)
	}

	typ, err := aci.DetectFileType(bytes.NewBuffer(hd))
	if err != nil {
		return fmt.Errorf("error detecting image type: %v", err)
	}
	dr, err := decompress(br, typ)
	if err != nil {
		return fmt.Errorf("error decompressing image: %v", err)
	}

	tr := tar.NewReader(dr)
	// TODO(sgotti). As some aci can contains files under mounted fs (like
	// /dev, /sys etc...) they should be filtered out.
	err = ptar.ExtractTar(tr, dir, true, nil)
	if err != nil {
		return fmt.Errorf("error extracting tar: %v", err)
	}

	return nil

}

// ExtractTarRootfs extract only the rootfs dir from a tarball (from a
// tar.Reader) into the given directory stripping out the "rootfs/" initial
// path.
// If overwrite is true, existing files will be overwritten.
func ExtractTarRootFS(tr *tar.Reader, dir string, overwrite bool) error {
	um := syscall.Umask(0)
	defer syscall.Umask(um)
	for {
		hdr, err := tr.Next()
		switch err {
		case io.EOF:
			return nil
		case nil:
			if strings.HasPrefix(hdr.Name, "rootfs") {
				hdr.Name, err = filepath.Rel("rootfs", hdr.Name)
				if err != nil {
					return fmt.Errorf("error extracting tarball: %v", err)
				}
				// Rename hard links
				if hdr.Typeflag == tar.TypeLink {
					hdr.Linkname, err = filepath.Rel("rootfs", hdr.Linkname)
					if err != nil {
						return fmt.Errorf("error extracting tarball: %v", err)
					}
				}
			} else {
				continue
			}
			err = ptar.ExtractFile(tr, hdr, dir, overwrite)
			if err != nil {
				return fmt.Errorf("error extracting tarball: %v", err)
			}
		default:
			return fmt.Errorf("error extracting tarball: %v", err)
		}
	}
}
