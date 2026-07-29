package games

import (
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
	"github.com/justyntemme/decky-mod-manager/internal/games/stardewvalley"
)

type Registry = gameext.Registry
type Extension = gameext.Extension
type LaunchToolSpec = gameext.LaunchToolSpec

var DefaultRegistry = gameext.NewRegistry([]gameext.Extension{
	stardewvalley.Extension(),
})
