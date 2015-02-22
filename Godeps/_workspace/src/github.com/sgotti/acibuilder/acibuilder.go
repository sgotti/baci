package acibuilder

import (
	"io"

	"github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/schema"
)

// ACIBuilder creates an aci given a starting ImageManifest and an output io.Writer
// Build will augmentate the provided ImageManifest adding a pathWhiteList if needed.
type ACIBuilder interface {
	Build(im schema.ImageManifest, out io.Writer) error
}
