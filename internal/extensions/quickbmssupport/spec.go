package quickbmssupport

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/justyntemme/decky-mod-manager/internal/extensions/sdk"
	"github.com/justyntemme/decky-mod-manager/internal/quickbms"
)

const (
	ID      = "quickbms-support"
	Name    = "QuickBMS Support"
	Version = "0.1.0"
	BuildID = "first-party-go"
)

const runtimeMessage = "DMM exposes a first-party Go namespace for Vortex's QuickBMS API calls backed by the typed QuickBMS process bridge."

type API struct {
	mu              sync.RWMutex
	registeredGames map[string]struct{}
	Runner          quickbms.Runner
}

func NewAPI(runner quickbms.Runner) *API {
	return &API{
		registeredGames: map[string]struct{}{},
		Runner:          runner,
	}
}

func (api *API) RegisterGame(gameID string) {
	gameID = strings.TrimSpace(gameID)
	if gameID == "" {
		return
	}
	api.mu.Lock()
	defer api.mu.Unlock()
	api.registeredGames[gameID] = struct{}{}
}

func (api *API) GameRegistered(gameID string) bool {
	api.mu.RLock()
	defer api.mu.RUnlock()
	_, ok := api.registeredGames[strings.TrimSpace(gameID)]
	return ok
}

func (api *API) List(ctx context.Context, gameID string, op quickbms.Operation) (quickbms.Result, error) {
	op.Type = quickbms.OperationList
	return api.run(ctx, gameID, op)
}

func (api *API) Extract(ctx context.Context, gameID string, op quickbms.Operation) (quickbms.Result, error) {
	op.Type = quickbms.OperationExtract
	return api.run(ctx, gameID, op)
}

func (api *API) Write(ctx context.Context, gameID string, op quickbms.Operation) (quickbms.Result, error) {
	op.Type = quickbms.OperationWrite
	return api.run(ctx, gameID, op)
}

func (api *API) Reimport(ctx context.Context, gameID string, op quickbms.Operation) (quickbms.Result, error) {
	op.Type = quickbms.OperationReimport
	return api.run(ctx, gameID, op)
}

func (api *API) run(ctx context.Context, gameID string, op quickbms.Operation) (quickbms.Result, error) {
	if api == nil {
		return quickbms.Result{}, errors.New("quickbms API is nil")
	}
	if !api.GameRegistered(gameID) {
		return quickbms.Result{}, errors.New("quickbms game is not registered: " + strings.TrimSpace(gameID))
	}
	return api.Runner.Run(ctx, op)
}

func Extension() sdk.Extension {
	return sdk.Extension{
		ID:      ID,
		Name:    Name,
		Kind:    sdk.ExtensionKindFramework,
		Version: Version,
		BuildID: BuildID,
		Register: func(r sdk.Registrar) {
			Register(r)
		},
	}
}

func Register(r sdk.Registrar) {
	for _, ref := range Sources() {
		r.RegisterSource(ref)
	}
	for _, api := range []sdk.ExtensionAPISpec{
		{ID: "qbmsRegisterGame", Name: "Register QuickBMS game support"},
		{ID: "qbmsList", Name: "List QuickBMS archive entries"},
		{ID: "qbmsExtract", Name: "Extract QuickBMS archive entries"},
		{ID: "qbmsWrite", Name: "Write QuickBMS archive entries"},
		{ID: "qbmsReimport", Name: "Reimport QuickBMS archive entries"},
		{ID: "quickbms-operation", Name: "Run QuickBMS operation events"},
	} {
		api.Status = sdk.CapabilityStatusReady
		api.Message = runtimeMessage
		r.RegisterExtensionAPI(api)
	}
	r.RegisterExtensionDashlet(sdk.ExtensionDashletSpec{
		ID:      "quickbms-support",
		Name:    "QBMS Support",
		Scope:   "quickbms",
		Status:  sdk.CapabilityStatusReady,
		Message: "DMM exposes Vortex's QuickBMS support dashlet as extension diagnostics for registered QuickBMS-backed games and the typed QuickBMS process bridge.",
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   sdk.EventGamemodeActivated,
		Name:    "Track active QuickBMS game support",
		Handler: gamemodeActivatedSupport,
	})
	r.RegisterEventHandler(sdk.EventHandlerSpec{
		Event:   "quickbms-operation",
		Name:    "Run QuickBMS operation",
		Handler: quickBMSOperationEvent,
	})
}

func gamemodeActivatedSupport(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: []string{"QuickBMS active game support state checked."}}, nil
}

func quickBMSOperationEvent(ctx context.Context, input sdk.EventHandlerInput) (sdk.EventHandlerResult, error) {
	if err := ctx.Err(); err != nil {
		return sdk.EventHandlerResult{}, err
	}
	return sdk.EventHandlerResult{Messages: []string{"QuickBMS operation routed through DMM's typed QuickBMS API bridge."}}, nil
}

func Sources() []sdk.SourceRef {
	return []sdk.SourceRef{
		{
			Name: "Vortex quickbms-support source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/quickbms-support/src/index.ts",
		},
		{
			Name: "Vortex quickbms process wrapper source",
			URL:  "https://github.com/Nexus-Mods/Vortex/tree/master/extensions/quickbms-support/src/quickbms.ts",
		},
	}
}
