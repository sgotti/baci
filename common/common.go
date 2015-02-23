package common

import "github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/schema/types"

// Configuration data provided by baci to the baci builder
type ConfigData struct {
	OutFile string
	AppName types.ACName
	Labels  types.Labels
}

const (
	BaciSourceDir = "/baci/source"
	BaciDestDir   = "/baci/dest"
	BaciDataDir   = "/baci/data"
	BaciRootDir   = "/baci/root"
)
