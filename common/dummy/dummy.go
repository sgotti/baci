package dummy

// This is a dummy package to ensure that Godep vendors
// actool, which is used in building the baci ACI.
import (
	_ "github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/aci"
	_ "github.com/sgotti/baci/Godeps/_workspace/src/github.com/appc/spec/actool"
)
