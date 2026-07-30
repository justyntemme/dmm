package extensions

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/fallout4"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/stardewvalley"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func FirstParty() []gameext.Extension {
	return []gameext.Extension{
		gameext.MustCompileExtension(fallout4.Extension()),
		gameext.MustCompileExtension(stardewvalley.Extension()),
	}
}
