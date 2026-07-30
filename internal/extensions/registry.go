package extensions

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/stardewvalley"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

func FirstParty() []gameext.Extension {
	return []gameext.Extension{
		stardewvalley.Extension(),
	}
}
