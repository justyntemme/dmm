package extensions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/gameext"
)

const pinnedVortexCommit = "c57894eb71af8234b58a6bd15ae5ab543eccac3a"

type vortexParityInventory struct {
	SchemaVersion int                     `json:"schema_version"`
	SourceCommit  string                  `json:"source_commit"`
	Components    []vortexParityComponent `json:"components"`
}

type vortexParityComponent struct {
	ID       string                `json:"id"`
	Surfaces []vortexParitySurface `json:"surfaces"`
}

type vortexParitySurface struct {
	Name  string             `json:"name"`
	Calls []vortexParityCall `json:"calls"`
}

type vortexParityCall struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

func TestFirstPartyParityMatchesPinnedVortexSourceInventory(t *testing.T) {
	inventoryPath := filepath.Join("..", "..", "testing", "vortex-parity-inventory.json")
	data, err := os.ReadFile(inventoryPath)
	if err != nil {
		t.Fatal(err)
	}
	var inventory vortexParityInventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatalf("parse %s: %v", inventoryPath, err)
	}
	if inventory.SchemaVersion != 1 {
		t.Fatalf("parity inventory schema = %d, want 1", inventory.SchemaVersion)
	}
	if inventory.SourceCommit != pinnedVortexCommit {
		t.Fatalf("parity inventory source commit = %q, want %q", inventory.SourceCommit, pinnedVortexCommit)
	}

	summaries := gameext.NewRegistry(FirstParty()).ExtensionSummaries()
	var problems []string
	for _, component := range inventory.Components {
		candidates := sourceBackedExtensions(component.ID, summaries)
		if len(candidates) == 0 {
			problems = append(problems, component.ID+": no DMM extension cites this pinned Vortex component")
			continue
		}
		for _, surface := range component.Surfaces {
			if sourceSurfaceNotApplicable(surface.Name, candidates) {
				continue
			}
			if sourceSurfaceCount(surface.Name, candidates) == 0 {
				problems = append(problems, component.ID+": missing DMM runtime registration for "+surface.Name+" (source "+firstCallLocation(surface)+")")
			}
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		t.Fatalf("pinned Vortex parity inventory has %d unresolved mappings:\n  %s", len(problems), strings.Join(problems, "\n  "))
	}
}

func sourceSurfaceNotApplicable(surface string, summaries []gameext.ExtensionSummary) bool {
	for _, summary := range summaries {
		for _, source := range summary.Sources {
			for _, disposition := range source.Dispositions {
				if disposition.Surface != surface || disposition.Status != sdk.CapabilityStatusNotApplicable {
					continue
				}
				if len(strings.TrimSpace(disposition.Reason)) < 40 {
					continue
				}
				return true
			}
		}
	}
	return false
}

func sourceBackedExtensions(componentID string, summaries []gameext.ExtensionSummary) []gameext.ExtensionSummary {
	needle := "/extensions/" + strings.Trim(componentID, "/")
	var out []gameext.ExtensionSummary
	for _, summary := range summaries {
		for _, source := range summary.Sources {
			url := strings.TrimSpace(source.URL)
			if !strings.Contains(url, needle) {
				continue
			}
			if !strings.Contains(url, "/tree/"+pinnedVortexCommit+"/") && !strings.Contains(url, "/blob/"+pinnedVortexCommit+"/") {
				continue
			}
			out = append(out, summary)
			break
		}
	}
	return out
}

func firstCallLocation(surface vortexParitySurface) string {
	if len(surface.Calls) == 0 {
		return "unknown"
	}
	return surface.Calls[0].File + ":" + strconv.Itoa(surface.Calls[0].Line)
}

func sourceSurfaceCount(surface string, summaries []gameext.ExtensionSummary) int {
	count := 0
	for _, summary := range summaries {
		caps := summary.Capabilities
		switch surface {
		case "registerGame":
			if !summary.VortexStub {
				count++
			}
		case "registerGameStub":
			if summary.VortexGameID != "" {
				count++
			}
		case "registerInstaller":
			count += len(caps.Installers) + len(caps.InstallerChoices)
		case "registerModType":
			count += len(caps.ModTypes)
		case "registerAction":
			count += len(caps.ExtensionActions)
		case "registerTest":
			// DMM uses executable extension tests, mod health checks, and
			// declarative runtime requirements for Vortex diagnostic tests.
			count += len(caps.ExtensionTests) + len(caps.HealthChecks) + len(caps.RuntimeRequirements)
		case "registerReducer":
			count += len(caps.StateReducers)
		case "registerMigration":
			count += len(caps.StateMigrations)
		case "registerDialog":
			count += len(caps.ExtensionDialogs)
		case "registerAPI":
			count += len(caps.ExtensionAPIs)
		case "registerTableAttribute":
			count += len(caps.ExtensionTableAttrs)
		case "registerLoadOrder":
			count += len(caps.LoadOrders)
		case "registerSettings":
			count += len(caps.ExtensionSettings)
		case "registerDashlet":
			count += len(caps.ExtensionDashlets)
		case "registerMainPage":
			count += len(caps.ExtensionMainPages)
		case "registerMerge":
			count += len(caps.Merges)
		case "registerInterpreter":
			count += len(caps.Interpreters)
		case "registerArchiveType":
			count += len(caps.ArchiveTypes)
		case "registerProfileFeature":
			count += len(caps.ProfileFeatures)
		case "registerGameStore":
			count += len(caps.GameStores)
		case "registerCollectionFeature":
			count += len(caps.CollectionFeatures)
		case "registerPersistor":
			count += len(caps.StatePersistors)
		case "registerLoadOrderPage":
			count += len(caps.ExtensionLoadOrderPages)
		case "registerAttributeExtractor":
			count += len(caps.AttributeExtractors)
		case "registerGameInfoProvider":
			count += len(caps.GameInfoProviders)
		case "registerProfileFile":
			count += len(caps.ProfileFiles)
		case "registerActionCheck":
			count += len(caps.ExtensionActionChecks)
		case "registerControlWrapper":
			count += len(caps.ExtensionControlWrappers)
		case "registerHistoryStack":
			count += len(caps.HistoryStacks)
		case "registerHealthCheck":
			count += len(caps.HealthChecks)
		case "registerStartHook":
			count += len(caps.StartHooks)
		case "registerToDo":
			// Vortex TODOs are dashboard shortcuts. DMM maps them to an
			// executable Action Center action or directly editable setting.
			count += len(caps.ExtensionActions) + len(caps.ExtensionSettings)
		}
	}
	return count
}
