package common

import "github.com/appc/spec/schema/types"

// Configuration data provided by baci to the baci builder
type ConfigData struct {
	OutFile string
	AppName types.ACName
	Labels  types.Labels
}
