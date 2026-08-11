package commandconquergenerals

import (
	"github.com/justyntemme/decky-mod-manager/internal/extensions/cncgeneralsbig"
	"github.com/justyntemme/decky-mod-manager/internal/installplan"
)

func matchBigArchive(root string) bool {
	return cncgeneralsbig.MatchArchive(root)
}

func buildBigArchive(input installplan.BuildInput) (installplan.Plan, error) {
	return cncgeneralsbig.BuildArchive(input, cncgeneralsbig.BuildConfig{
		DetectionKind:   "cnc-generals-big",
		LayoutError:     "Command & Conquer: Generals archive does not contain a supported .big package layout",
		DetectionReason: "Verified Generals community guidance supports dropping .big packages into the game root",
		EmptyReason:     "Command & Conquer: Generals .big installer matched but produced no deployable files",
	})
}
