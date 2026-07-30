package games

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

type Registry = gameext.Registry
type Extension = gameext.Extension
type LaunchToolSpec = gameext.LaunchToolSpec

var DefaultRegistry = gameext.NewRegistry(extensions.FirstParty())
