package galacticcivilizations3

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
)

func willDeploySnapshot(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	if !containsGalCivDeployment(input.Mappings, input.ManagedFiles) {
		return sdk.EventHandlerResult{}, nil
	}
	return sdk.EventHandlerResult{Messages: []string{"Galactic Civilizations III deployment snapshot captured managed mod files before deploy."}}, nil
}

func didDeployReminder(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	for _, mod := range input.Mods {
		switch strings.TrimSpace(mod.ModType) {
		case modType, crusadeModType:
			return sdk.EventHandlerResult{Notices: []sdk.EventNotice{{
				Message:     "Galactic Civilizations III mods were deployed. Enable mods inside the game's options menu before starting a modded save.",
				ToolID:      "galciv3-enable-mods",
				ToolName:    "GalCiv3 mod activation",
				ActionLabel: "Review in-game",
			}}}, nil
		}
	}
	return sdk.EventHandlerResult{}, nil
}

func containsGalCivDeployment(mappings []deploy.FileMapping, files []deploy.AppliedFile) bool {
	for _, mapping := range mappings {
		if isGalCivManagedTarget(mapping.TargetRelative) {
			return true
		}
	}
	for _, file := range files {
		if isGalCivManagedTarget(file.TargetPath) {
			return true
		}
	}
	return false
}

func isGalCivManagedTarget(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	for _, segment := range []string{"/Mods/", "Mods/", "/Factions/", "Factions/"} {
		if strings.Contains(path, segment) || strings.HasPrefix(path, segment) {
			return true
		}
	}
	return false
}
