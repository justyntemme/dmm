package server

import (
	"os"
	"runtime"
	"strings"
	"syscall"

	"github.com/justyntemme/decky-mod-manager/internal/deploy"
	"github.com/justyntemme/decky-mod-manager/internal/storage"
)

func (s *Server) deploymentStrategyCapabilities(game storage.Game, effective string) ([]deploymentStrategyCapability, string, []string) {
	gamePath := strings.TrimSpace(game.GamePath)
	gameExists := directoryExists(gamePath)
	s.cfgMu.RLock()
	dataDir := s.cfg.DataDir
	s.cfgMu.RUnlock()
	sameDevice, deviceKnown, deviceReason := sameFilesystem(dataDir, gamePath)

	capabilities := []deploymentStrategyCapability{
		{
			Strategy:  string(deploy.StrategySymlink),
			Supported: gameExists && runtime.GOOS != "windows",
			Reason:    symlinkCapabilityReason(gameExists),
		},
		{
			Strategy:  string(deploy.StrategyHardlink),
			Supported: gameExists && deviceKnown && sameDevice,
			Reason:    hardlinkCapabilityReason(gameExists, deviceKnown, sameDevice, deviceReason),
		},
		{
			Strategy:  string(deploy.StrategyCopy),
			Supported: gameExists,
			Reason:    copyCapabilityReason(gameExists),
		},
	}
	recommended := recommendedDeploymentStrategy(effective, capabilities)
	for i := range capabilities {
		capabilities[i].Recommended = capabilities[i].Strategy == recommended
	}
	var warnings []string
	for _, capability := range capabilities {
		if capability.Strategy == effective && !capability.Supported {
			warnings = append(warnings, "Effective strategy "+effective+" is not recommended: "+capability.Reason)
		}
	}
	return capabilities, recommended, warnings
}

func recommendedDeploymentStrategy(effective string, capabilities []deploymentStrategyCapability) string {
	for _, capability := range capabilities {
		if capability.Strategy == effective && capability.Supported {
			return effective
		}
	}
	for _, preferred := range []string{string(deploy.StrategySymlink), string(deploy.StrategyHardlink), string(deploy.StrategyCopy)} {
		for _, capability := range capabilities {
			if capability.Strategy == preferred && capability.Supported {
				return preferred
			}
		}
	}
	if strings.TrimSpace(effective) != "" {
		return effective
	}
	return string(deploy.StrategyCopy)
}

func directoryExists(path string) bool {
	info, err := os.Stat(strings.TrimSpace(path))
	return err == nil && info.IsDir()
}

func sameFilesystem(left, right string) (same bool, known bool, reason string) {
	leftInfo, err := os.Stat(strings.TrimSpace(left))
	if err != nil {
		return false, false, "DMM data directory is unavailable."
	}
	rightInfo, err := os.Stat(strings.TrimSpace(right))
	if err != nil {
		return false, false, "Game directory is unavailable."
	}
	leftStat, ok := leftInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, false, "DMM cannot identify the data directory filesystem."
	}
	rightStat, ok := rightInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, false, "DMM cannot identify the game directory filesystem."
	}
	return leftStat.Dev == rightStat.Dev, true, ""
}

func symlinkCapabilityReason(gameExists bool) string {
	if !gameExists {
		return "Game directory is unavailable."
	}
	if runtime.GOOS == "windows" {
		return "Windows symlink privileges are not assumed."
	}
	return "Symlinks can cross filesystems on SteamOS/Linux; DMM still validates each target during deployment."
}

func hardlinkCapabilityReason(gameExists, deviceKnown, sameDevice bool, deviceReason string) string {
	if !gameExists {
		return "Game directory is unavailable."
	}
	if !deviceKnown {
		return deviceReason
	}
	if !sameDevice {
		return "Hardlinks cannot cross filesystems; use symlink or copy for games on a different drive."
	}
	return "DMM data and game folders are on the same filesystem."
}

func copyCapabilityReason(gameExists bool) string {
	if !gameExists {
		return "Game directory is unavailable."
	}
	return "Copy deployment works across filesystems but uses more disk space."
}
