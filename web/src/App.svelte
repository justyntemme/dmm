<script lang="ts">
  import { onMount } from "svelte";

  type UISettings = {
    favorite_game_ids?: string[];
    recent_games?: Record<string, number>;
    game_sort?: GameSort;
  };

  type Status = {
    game_count: number;
    install: { auto_install_captured_downloads: boolean; auto_enable_installed_mods: boolean };
    download?: {
      max_concurrent_captured_downloads: number;
      max_concurrent_captured_downloads_per_game: number;
      active_captured_downloads: number;
      active_captured_downloads_by_game?: Record<string, number>;
    };
    nexus: { api_key_configured: boolean };
    ui?: UISettings;
  };

  type CatalogStatus = {
    id: string;
    name: string;
    kind: string;
    status: string;
    configured: boolean;
    credentials_required: boolean;
    capabilities: string[];
    url_import: boolean;
    search: boolean;
    browse: boolean;
    download: boolean;
    archive_upload: boolean;
    installed_management: boolean;
    source_tag: string;
    notes?: string[];
  };

  type Game = {
    app_id: string;
    name: string;
    path: string;
    library_path: string;
    state: string;
    markers?: string[];
    steam_workshop?: SteamWorkshop;
    nexus_domains?: string[];
  };

  type SteamWorkshop = {
    detected: boolean;
    content_path?: string;
    manifest_path?: string;
    item_count: number;
    sample_item_ids?: string[];
    coexistence_allowed: boolean;
    management_supported: boolean;
    message?: string;
  };

  type WorkshopItem = {
    steam_app_id?: string;
    catalog?: string;
    source_tag?: string;
    published_file_id: string;
    title?: string;
    subscribed: boolean;
    downloaded: boolean;
    disabled_locally: boolean;
    disabled_known: boolean;
    position: number;
  };

  type WorkshopState = {
    app_id: string;
    supported: boolean;
    info?: SteamWorkshop;
    items: WorkshopItem[];
  };

  type Profile = {
    id: number;
    game_id: number;
    name: string;
    is_default: boolean;
    deployment_strategy?: string;
  };

  type Job = {
    id: string;
    type: string;
    title: string;
    status: string;
    message: string;
    payload?: Record<string, string>;
    app_id?: string;
    catalog?: string;
    source_tag?: string;
    created_at: string;
    updated_at: string;
  };

  type DomainEvent = {
    id: number;
    type: string;
    app_id?: string;
    job_id?: string;
    payload?: unknown;
    created_at: string;
  };

  type ClientEventDetail = Record<string, string | number | boolean | null | undefined>;

  type DownloadLink = {
    name: string;
    short_name: string;
    URI: string;
  };

  type NexusFile = {
    file_id: number;
    name: string;
    version: string;
    file_name: string;
    size: number;
  };

  type NexusSearchSort = "downloads" | "unique_downloads" | "popular" | "updated" | "name" | "relevance";
  type NexusTimeWindow = "all" | "one_week" | "three_weeks" | "one_month" | "three_months" | "one_year";

  type NexusModResult = {
    mod_id: number;
    name: string;
    summary: string;
    version: string;
    thumbnail_url: string;
    downloads: number;
    endorsements: number;
    updated_at: string;
    supports_vortex: boolean;
    url: string;
  };

  type InstalledMod = {
    id: number;
    name: string;
    profile_id: number;
    catalog: string;
    source_tag?: string;
    source_game_domain: string;
    source_mod_id: string;
    source_file_id: string;
    version: string;
    enabled: boolean;
    priority: number;
    status: string;
    mod_type?: string;
    planner_id?: string;
    metadata?: ModMetadata[];
    update?: ModUpdate;
  };

  type ModUpdate = {
    catalog?: string;
    source_tag?: string;
    status: string;
    latest_file_id?: string;
    latest_file_name?: string;
    latest_version?: string;
    latest_uploaded_at?: number;
    message?: string;
    checked_at?: string;
  };

  type ModDependency = {
    unique_id?: string;
    minimum_version?: string;
    required: boolean;
  };

  type ModMetadata = {
    kind?: string;
    name?: string;
    unique_id?: string;
    version?: string;
    entry_dll?: string;
    minimum_api_version?: string;
    additional_logical_file_names?: string[];
    manifest_version?: string;
    content_pack_for?: ModDependency;
    dependencies?: ModDependency[];
  };

  type InstallCandidate = {
    id: number;
    steam_app_id: string;
    name: string;
    catalog: string;
    source_tag?: string;
    source_url?: string;
    source_game_domain: string;
    source_mod_id: string;
    source_file_id: string;
    archive_path: string;
    status: string;
    reason: string;
    installer_json?: string;
    choices_json?: string;
  };

  type InstallerChoicePreset = {
    id: number;
    steam_app_id: string;
    catalog: string;
    source_game_domain: string;
    source_mod_id: string;
    source_file_id: string;
    installer_kind: string;
    reuse_scope: string;
    choices_json: string;
    updated_at: string;
  };

  type FomodInstaller = {
    name: string;
    steps?: FomodStep[];
  };

  type FomodStep = {
    id: string;
    name: string;
    visible?: boolean;
    groups?: FomodGroup[];
  };

  type FomodGroup = {
    id: string;
    name: string;
    type: string;
    plugins?: FomodPlugin[];
  };

  type FomodPlugin = {
    id: string;
    name: string;
    description?: string;
    type?: string;
    effective_type?: string;
  };

  type DeployAction = {
    source_path: string;
    target_path: string;
    target_relative: string;
    strategy: string;
    operation: string;
    installed_mod_id?: number;
    mod_id?: string;
    priority?: number;
    winner_installed_mod_id?: number;
    winner_mod_id?: string;
    winner_priority?: number;
    conflict: boolean;
    conflict_reason?: string;
  };

  type DeployPlan = {
    staging_root: string;
    target_root: string;
    strategy: string;
    actions: DeployAction[];
    conflicts: DeployAction[];
  };

  type ConflictChoiceTarget = {
    target_path: string;
    target_relative: string;
    current_winner_id: number;
    current_winner_name: string;
    reason: string;
    candidates: Array<{
      id: number;
      name: string;
      catalog?: string;
      priority?: number;
      current: boolean;
    }>;
  };

  type DeploymentStatus = {
    deployed: boolean;
    file_count: number;
    strategy?: string;
    sample_files?: string[];
    apply_rollback_on_failure: boolean;
    repair_available: boolean;
    restore_available: boolean;
    purge_available: boolean;
    recovery_summary?: string;
    restore_summary?: string;
  };

  type DeploymentHistoryItem = {
    id: number;
    profile_id: number;
    profile_name: string;
    status: string;
    strategy: string;
    file_count: number;
    created_at: string;
    updated_at: string;
  };

  type DeploymentSettings = {
    app_id: string;
    profile_id?: number;
    profile_name?: string;
    strategy: string;
    profile_strategy: string;
    game_strategy: string;
    effective_strategy: string;
    source: string;
    extension_default: string;
    allowed_strategies: string[];
    recommended_strategy: string;
    strategy_warnings?: string[];
    capabilities?: DeploymentStrategyCapability[];
  };

  type DeploymentStrategyCapability = {
    strategy: string;
    supported: boolean;
    recommended: boolean;
    reason: string;
  };

  type PluginLoadOrder = {
    app_id: string;
    supported: boolean;
    activation_id?: string;
    name?: string;
    target_root?: string;
    plugins_file?: string;
    load_order_file?: string;
    plugins: PluginLoadOrderEntry[];
    warnings?: string[];
  };

  type PluginLoadOrderEntry = {
    name: string;
    source: string;
    catalog?: string;
    installed_mod_id?: number;
    mod_id?: string;
    priority: number;
    active: boolean;
  };

  type RuntimeRequirement = {
    id: string;
    name: string;
    kind: string;
    required: boolean;
    status: "ok" | "missing" | string;
    message: string;
    details?: string[];
    help_url?: string;
    install_hint?: string;
  };

  type GameLaunchStatus = {
    required: boolean;
    configured: boolean;
    can_configure: boolean;
    desired_options?: string;
    current_options?: string;
    missing_files?: string[];
    tool?: {
      id: string;
      name: string;
      executable_path: string;
      source_extension: string;
    };
    action?: {
      type: string;
      app_id: string;
      tool_id: string;
      desired_options: string;
      current_options?: string;
      risk: string;
      source_extension: string;
    };
    error?: string;
  };

  type GameDiagnostics = {
    steam_workshop?: SteamWorkshop;
    runtime_requirements?: RuntimeRequirement[];
    validation_warnings?: string[];
  };

  type ProfileApplyResult = {
    status: "applied" | "blocked" | "failed" | string;
    message: string;
    job?: Job;
    plan?: DeployPlan;
    applied?: unknown[];
    launch?: GameLaunchStatus;
  };

  type ProfileModUpdateResult = {
    mod: InstalledMod;
    apply: ProfileApplyResult;
  };

  type ProfileModOrderUpdateResult = {
    mods: InstalledMod[];
    apply: ProfileApplyResult;
  };

  type SetDefaultProfileResult = {
    profile: Profile;
    apply: ProfileApplyResult;
  };

  type Confirmation = {
    title: string;
    message: string;
    detail?: string;
    confirmLabel: string;
    danger?: boolean;
    run: () => Promise<void>;
  };

  type Drawer = "games" | "settings" | null;
  type Surface = "actions" | "game" | "settings";
  type GameModule = "plugins" | "actions" | "profiles" | "review" | "paths";
  type SettingsPage = "overview" | "jobs" | "install" | "sources" | "nexus";
  type GameSort = "recent" | "az" | "za";
  type ModListSort = "profile" | "source" | "az" | "enabled";

  let status: Status | null = null;
  let catalogs: CatalogStatus[] = [];
  let games: Game[] = [];
  let jobs: Job[] = [];
  let selectedGame: Game | null = null;
  let profiles: Profile[] = [];
  let installedMods: InstalledMod[] = [];
  let installCandidates: InstallCandidate[] = [];
  let installerChoicePresets: InstallerChoicePreset[] = [];
  let profileName = "";
  let captureURL = "";
  let lastCaptureURL = "";
  let resolvedCapture = "";
  let nexusFiles: NexusFile[] = [];
  let nexusSearchQuery = "";
  let nexusSearchSort: NexusSearchSort = "downloads";
  let nexusSearchTimeWindow: NexusTimeWindow = "all";
  let nexusSearchVortexOnly = true;
  let nexusSearchResults: NexusModResult[] = [];
  let nexusSearchTotal = 0;
  let nexusSearchBusy = false;
  let nexusSearchError = "";
  let selectedNexusModID: number | null = null;
  let nexusFilesByMod: Record<number, NexusFile[]> = {};
  let busyNexusFileKey = "";
  let localArchiveFile: File | null = null;
  let localArchiveInput: HTMLInputElement | null = null;
  let localArchiveBusy = false;
  let localArchiveMessage = "";
  let downloadLinks: DownloadLink[] = [];
  let deployPlan: DeployPlan | null = null;
  let deploymentStatus: DeploymentStatus | null = null;
  let deploymentSettings: DeploymentSettings | null = null;
  let deploymentHistory: DeploymentHistoryItem[] = [];
  let pluginLoadOrder: PluginLoadOrder | null = null;
  let gameDiagnostics: GameDiagnostics | null = null;
  let gameLaunchStatus: GameLaunchStatus | null = null;
  let workshopState: WorkshopState | null = null;
  let workshopItems: WorkshopItem[] = [];
  let globalInstallCandidates: InstallCandidate[] = [];
  let loading = true;
  let error = "";
  let drawer: Drawer = null;
  let confirmation: Confirmation | null = null;
  let surface: Surface = "actions";
  let activeGameModule: GameModule = "plugins";
  let activeSettingsPage: SettingsPage = "overview";
  let gameQuery = "";
  let gameSort: GameSort = "recent";
  let actionSourceFilter = "all";
  let jobSourceFilter = "all";
  let modSourceFilter = "all";
  let modListSort: ModListSort = "profile";
  let favoriteGameIDs = new Set<string>();
  let gameRecent: Record<string, number> = {};
  let busyJobs: Record<string, boolean> = {};
  let busyInstallCandidates: Record<number, boolean> = {};
  let busyWorkshopActions: Record<string, boolean> = {};
  let workshopOrderBusy = false;
  let busyMods: Record<number, "toggle" | "remove" | "reinstall" | "update"> = {};
  let modUpdateBusy = false;
  let modUpdateMessage = "";
  let modUpdateBrowserURL = "";
  let modIOAPIKey = "";
  let curseForgeAPIKey = "";
  let catalogSettingsBusy = "";
  let catalogSettingsMessage = "";
  let initialRefreshComplete = false;
  let selectedGameRefreshTimer: number | null = null;
  let selectedGameRefreshNeedsPreview = false;
  let selectedGameRefreshNeedsJobs = false;
  let actionStateRefreshTimer: number | null = null;
  let actionStateRefreshInFlight = false;
  let actionStateRefreshQueued = false;
  let actionStateRefreshNeedsSelectedGame = false;
  let actionStateRefreshNeedsPreview = false;
  let candidateSelections: Record<number, Record<string, string[]>> = {};
  let candidateStepIndices: Record<number, number> = {};
  let refreshJobsInFlight = false;
  let refreshJobsQueued = false;
  let eventSocket: WebSocket | null = null;
  let eventReconnectTimer: number | null = null;
  let eventReconnectDelay = 1000;
  let lastEventID = 0;
  let actionStateRefreshReasons = new Set<string>();
  let selectedGameRefreshReasons = new Set<string>();
  let actionStateRefreshSequence = 0;
  let selectedGameRefreshSequence = 0;
  let fullRefreshSequence = 0;

  function setBusyMod(modID: number, action: "toggle" | "remove" | "reinstall" | "update") {
    busyMods = { ...busyMods, [modID]: action };
  }

  function clearBusyMod(modID: number) {
    const { [modID]: _removed, ...rest } = busyMods;
    busyMods = rest;
  }

  $: cleanCount = games.filter((game) => game.state === "clean_candidate").length;
  $: reviewCount = games.length - cleanCount;
  $: readyCatalogCount = catalogs.filter((catalog) => catalog.status === "ready").length;
  $: sourceCatalogCount = catalogs.filter((catalog) => catalog.kind !== "platform").length;
  $: readySourceCatalogCount = catalogs.filter((catalog) => catalog.kind !== "platform" && catalog.status === "ready").length;
  $: selectedProfile = profiles.find((profile) => profile.is_default) ?? profiles[0] ?? null;
  $: capturedInstallActions = jobs.filter((job) => job.type === "captured-install" && !["completed", "canceled"].includes(job.status));
  $: actionItems = jobs.filter((job) => ["captured-install", "installer-choice", "steam-workshop-action"].includes(job.type) && !["completed", "canceled"].includes(job.status));
  $: actionCenterCandidates = globalInstallCandidates.filter((candidate) => !hasOpenInstallerChoiceJob(candidate));
  $: actionSourceOptions = sourceOptionsForActions(actionItems, actionCenterCandidates);
  $: visibleActionItems = filterJobsBySource(actionItems, actionSourceFilter);
  $: visibleActionCenterCandidates = filterCandidatesBySource(actionCenterCandidates, actionSourceFilter);
  $: jobSourceOptions = sourceOptionsForJobs(jobs);
  $: visibleJobs = filterJobsBySource(jobs, jobSourceFilter);
  $: selectedGameCapturedInstallActions = selectedGame ? capturedInstallActions.filter((job) => actionMatchesGame(job, selectedGame)) : capturedInstallActions;
  $: selectedGameActionItems = selectedGame ? actionItems.filter((job) => actionMatchesGame(job, selectedGame)) : actionItems;
  $: selectedGameActionCount = selectedGameActionItems.length + installCandidates.length;
  $: globalActionCount = actionItems.length + actionCenterCandidates.length;
  $: selectedWorkshop = gameDiagnostics?.steam_workshop?.detected
    ? gameDiagnostics.steam_workshop
    : selectedGame?.steam_workshop?.detected
      ? selectedGame.steam_workshop
      : null;
  $: selectedGameActivity = selectedGame
    ? jobs.filter((job) => {
        if (job.type === "captured-install") return actionMatchesGame(job, selectedGame) && !["completed", "canceled"].includes(job.status);
        return ["installer-choice", "steam-workshop-action", "deploy", "purge", "repair", "recover-downloads", "rollback"].includes(job.type) && jobMatchesGame(job, selectedGame) && !["completed", "canceled"].includes(job.status);
      })
    : [];
  $: filteredGames = sortDrawerGames(games.filter((game) => {
    const query = gameQuery.trim().toLowerCase();
    if (!query) return true;
    return game.name.toLowerCase().includes(query) || game.app_id.includes(query);
  }));
  $: title = surface === "settings" ? settingsTitle(activeSettingsPage) : surface === "actions" ? "Action Center" : selectedGame?.name ?? "Select a Game";
  $: deployableActions = getDeployableActions(deployPlan);
  $: conflictChoiceTargets = getConflictChoiceTargets(deployPlan);
  $: enabledMods = installedMods.filter((mod) => mod.enabled);
  $: disabledMods = installedMods.filter((mod) => !mod.enabled);
  $: modSourceOptions = sourceOptionsForMods(installedMods, workshopItems, selectedWorkshop);
  $: visibleInstalledMods = sortModsForList(installedMods.filter((mod) => modSourceFilter === "all" || sourceKey(sourceForMod(mod)) === modSourceFilter));
  $: showWorkshopModRows = Boolean(selectedWorkshop && (modSourceFilter === "all" || modSourceFilter === "steam-workshop") && (workshopItems.length > 0 || selectedWorkshop.management_supported));
  $: hasVisibleModRows = visibleInstalledMods.length > 0 || showWorkshopModRows;
  $: deployAdds = deployableActions.filter((action) => action.operation === "add").length;
  $: deployReplaces = deployableActions.filter((action) => action.operation === "replace").length;
  $: deployRemoves = deployableActions.filter((action) => action.operation === "remove").length;
  $: hasDeployConflicts = (deployPlan?.conflicts.length ?? 0) > 0;
  $: hasPendingProfileChanges = Boolean(deployPlan && deployableActions.length > 0 && !hasDeployConflicts);
  $: visibleValidationWarnings = displayValidationWarnings(gameDiagnostics);
  $: launchSetupAvailable = Boolean(gameLaunchStatus?.required && !gameLaunchStatus.configured && gameLaunchStatus.can_configure && gameLaunchStatus.action);

  async function getJSON<T>(url: string): Promise<T> {
    const response = await fetch(url);
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  function logClientEvent(message: string, detail: ClientEventDetail = {}) {
    const body = JSON.stringify({ message, detail });
    fetch("/api/client-events", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body
    }).catch(() => {
      // Client diagnostics must not interfere with the mod-management flow.
    });
  }

  function compactLogValue(value: unknown, maxLength = 240) {
    const text = String(value ?? "");
    if (text.length <= maxLength) return text;
    return `${text.slice(0, maxLength)}...`;
  }

  function eventDiagnosticDetail(event: DomainEvent): ClientEventDetail {
    const detail: ClientEventDetail = {
      event_id: event.id,
      type: event.type,
      app_id: event.app_id,
      job_id: event.job_id,
      selected_app_id: selectedGame?.app_id ?? "",
      surface,
      module: activeGameModule
    };
    if (isJob(event.payload)) {
      detail.job_type = event.payload.type;
      detail.job_status = event.payload.status;
      detail.job_app_id = event.payload.app_id ?? event.payload.payload?.app_id;
      detail.job_game_domain = event.payload.payload?.game_domain;
    }
    return detail;
  }

  function joinedRefreshReasons(reasons: Set<string>) {
    return Array.from(reasons).filter(Boolean).slice(0, 8).join(",");
  }

  async function refresh(reason = "manual") {
    const sequence = ++fullRefreshSequence;
    logClientEvent("full refresh started", { sequence, reason, selected_app_id: selectedGame?.app_id ?? "", surface });
    error = "";
    try {
      const [nextStatus, nextGames, nextCatalogs] = await Promise.all([
        getJSON<Status>("/api/status"),
        getJSON<Game[]>("/api/games"),
        getJSON<CatalogStatus[]>("/api/catalogs")
      ]);
      status = nextStatus;
      applyUIPreferences(nextStatus);
      games = nextGames;
      catalogs = nextCatalogs;
      const previousSelection = selectedGame?.app_id;
      selectedGame = nextGames.find((game) => game.app_id === previousSelection) ?? null;
      if (selectedGame) await loadGameState(selectedGame);
      logClientEvent("full refresh completed", {
        sequence,
        reason,
        games: nextGames.length,
        catalogs: nextCatalogs.length,
        selected_app_id: selectedGame?.app_id ?? ""
      });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      logClientEvent("full refresh failed", { sequence, reason, error: compactLogValue(error) });
    } finally {
      loading = false;
      initialRefreshComplete = true;
    }
    try {
      await refreshActionState(`${reason}:action-state`);
      reconcileBusyState();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      logClientEvent("full refresh action state failed", { sequence, reason, error: compactLogValue(error) });
    }
  }

  async function refreshActionState(reason = "manual") {
    const sequence = ++actionStateRefreshSequence;
    logClientEvent("action state refresh started", { sequence, reason, selected_app_id: selectedGame?.app_id ?? "", surface });
    const [nextJobs, nextCandidates] = await Promise.all([
      getJSON<Job[]>("/api/jobs"),
      getJSON<InstallCandidate[]>("/api/install-candidates")
    ]);
    jobs = nextJobs;
    globalInstallCandidates = nextCandidates;
    reconcileBusyState();
    logClientEvent("action state refresh completed", {
      sequence,
      reason,
      jobs: nextJobs.length,
      actions: nextJobs.filter((job) => isActionJob(job) && !["completed", "canceled"].includes(job.status)).length,
      candidates: nextCandidates.length,
      selected_app_id: selectedGame?.app_id ?? ""
    });
  }

  async function refreshJobsAndSelectedGame(reason = "mutation", refreshPreview = deployPlan !== null) {
    if (refreshJobsInFlight) {
      refreshJobsQueued = true;
      logClientEvent("jobs selected-game refresh queued", { reason, selected_app_id: selectedGame?.app_id ?? "" });
      return;
    }
    refreshJobsInFlight = true;
    logClientEvent("jobs selected-game refresh started", { reason, selected_app_id: selectedGame?.app_id ?? "", refresh_preview: refreshPreview });
    try {
      await refreshActionState(`${reason}:jobs`);
      if (selectedGame) await refreshSelectedGame({ refreshPreview, reason });
      reconcileBusyState();
      logClientEvent("jobs selected-game refresh completed", { reason, selected_app_id: selectedGame?.app_id ?? "" });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      logClientEvent("jobs selected-game refresh failed", { reason, selected_app_id: selectedGame?.app_id ?? "", error: compactLogValue(error) });
    } finally {
      refreshJobsInFlight = false;
      if (refreshJobsQueued) {
        refreshJobsQueued = false;
        void refreshJobsAndSelectedGame(`${reason}:queued`);
      }
    }
  }

  function scheduleActionStateRefresh(refreshSelectedGame = false, refreshPreview = false, reason = "event", event?: DomainEvent) {
    actionStateRefreshReasons.add(reason);
    actionStateRefreshNeedsSelectedGame = actionStateRefreshNeedsSelectedGame || refreshSelectedGame;
    actionStateRefreshNeedsPreview = actionStateRefreshNeedsPreview || refreshPreview;
    logClientEvent("action state refresh scheduled", {
      ...(event ? eventDiagnosticDetail(event) : {}),
      reason,
      refresh_selected_game: actionStateRefreshNeedsSelectedGame,
      refresh_preview: actionStateRefreshNeedsPreview
    });
    if (actionStateRefreshTimer !== null) return;
    actionStateRefreshTimer = window.setTimeout(() => {
      actionStateRefreshTimer = null;
      void flushActionStateRefresh();
    }, 150);
  }

  async function flushActionStateRefresh() {
    if (actionStateRefreshInFlight) {
      actionStateRefreshQueued = true;
      logClientEvent("action state refresh queued while running", {
        reasons: joinedRefreshReasons(actionStateRefreshReasons),
        selected_app_id: selectedGame?.app_id ?? ""
      });
      return;
    }
    actionStateRefreshInFlight = true;
    const shouldRefreshSelectedGame = actionStateRefreshNeedsSelectedGame;
    const shouldRefreshPreview = actionStateRefreshNeedsPreview;
    const reasons = joinedRefreshReasons(actionStateRefreshReasons) || "event";
    actionStateRefreshReasons.clear();
    actionStateRefreshNeedsSelectedGame = false;
    actionStateRefreshNeedsPreview = false;
    logClientEvent("action state event refresh started", {
      reasons,
      refresh_selected_game: shouldRefreshSelectedGame,
      refresh_preview: shouldRefreshPreview,
      selected_app_id: selectedGame?.app_id ?? ""
    });
    try {
      await refreshActionState(`event:${reasons}`);
      if (shouldRefreshSelectedGame && selectedGame) {
        await refreshSelectedGame({ refreshPreview: shouldRefreshPreview, reason: `event:${reasons}` });
      }
      logClientEvent("action state event refresh completed", {
        reasons,
        refresh_selected_game: shouldRefreshSelectedGame,
        refresh_preview: shouldRefreshPreview,
        selected_app_id: selectedGame?.app_id ?? ""
      });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      logClientEvent("action state event refresh failed", { reasons, selected_app_id: selectedGame?.app_id ?? "", error: compactLogValue(error) });
    } finally {
      actionStateRefreshInFlight = false;
      if (actionStateRefreshQueued) {
        actionStateRefreshQueued = false;
        void flushActionStateRefresh();
      }
    }
  }

  function eventSocketURL() {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const after = lastEventID > 0 ? `?after=${lastEventID}` : "";
    return `${protocol}//${window.location.host}/api/events/ws${after}`;
  }

  function connectEvents() {
    if (eventSocket && [WebSocket.CONNECTING, WebSocket.OPEN].includes(eventSocket.readyState)) return;
    if (eventReconnectTimer !== null) {
      window.clearTimeout(eventReconnectTimer);
      eventReconnectTimer = null;
    }

    logClientEvent("events connecting", { after_id: lastEventID, host: window.location.host });
    const socket = new WebSocket(eventSocketURL());
    eventSocket = socket;
    socket.onopen = () => {
      eventReconnectDelay = 1000;
      logClientEvent("events connected", { after_id: lastEventID, host: window.location.host });
    };
    socket.onmessage = (message) => {
      if (typeof message.data !== "string") return;
      try {
        const event = JSON.parse(message.data) as DomainEvent;
        logDomainEvent(event);
        handleDomainEvent(event);
      } catch (err) {
        error = err instanceof Error ? err.message : String(err);
        logClientEvent("events message failed", { error });
      }
    };
    socket.onerror = () => {
      logClientEvent("events socket error", { after_id: lastEventID, host: window.location.host });
      socket.close();
    };
    socket.onclose = (event) => {
      logClientEvent("events closed", { after_id: lastEventID, code: event.code, clean: event.wasClean, reason: event.reason });
      if (eventSocket !== socket) return;
      eventSocket = null;
      scheduleEventReconnect();
    };
  }

  function logDomainEvent(event: DomainEvent) {
    const detail: ClientEventDetail = {
      id: event.id,
      type: event.type,
      app_id: event.app_id,
      job_id: event.job_id
    };
    if (isJob(event.payload)) {
      detail.job_type = event.payload.type;
      detail.job_status = event.payload.status;
      detail.job_message = compactLogValue(event.payload.message);
    }
    logClientEvent("events received", detail);
  }

  function scheduleEventReconnect() {
    if (eventReconnectTimer !== null) return;
    const delay = eventReconnectDelay;
    eventReconnectDelay = Math.min(eventReconnectDelay * 2, 10000);
    logClientEvent("events reconnect scheduled", { delay_ms: delay, after_id: lastEventID });
    eventReconnectTimer = window.setTimeout(() => {
      eventReconnectTimer = null;
      scheduleActionStateRefresh(true, deployPlan !== null, "event-reconnect");
      connectEvents();
    }, delay);
  }

  function closeEventSocket() {
    if (eventReconnectTimer !== null) {
      window.clearTimeout(eventReconnectTimer);
      eventReconnectTimer = null;
    }
    if (eventSocket) {
      const socket = eventSocket;
      eventSocket = null;
      socket.close();
    }
  }

  function handleDomainEvent(event: DomainEvent) {
    if (event.id > lastEventID) lastEventID = event.id;
    if (event.type === "ui.changed") {
      if (isUISettings(event.payload)) applyUIPreferencesFromUI(event.payload);
      return;
    }
    if (event.type === "jobs.snapshot") {
      if (Array.isArray(event.payload)) {
        jobs = event.payload as Job[];
        reconcileBusyState();
        scheduleActionStateRefresh(Boolean(selectedGame), deployPlan !== null, "jobs.snapshot", event);
      }
      return;
    }
    if (event.type === "job.updated") {
      if (isJob(event.payload)) {
        upsertJob(event.payload);
        const matchesSelectedGame = Boolean(selectedGame && jobMatchesGame(event.payload, selectedGame));
        if (isActionJob(event.payload) || matchesSelectedGame) {
          scheduleActionStateRefresh(
            matchesSelectedGame,
            matchesSelectedGame && (event.payload.status === "completed" || deployPlan !== null || event.payload.type === "installer-choice"),
            "job.updated",
            event
          );
        } else {
          logClientEvent("events refresh skipped", { ...eventDiagnosticDetail(event), reason: "job did not match action center or selected game" });
        }
      }
      return;
    }
    if (event.type === "install.changed") {
      const matchesSelectedGame = eventMatchesSelectedGame(event);
      scheduleActionStateRefresh(matchesSelectedGame, matchesSelectedGame, "install.changed", event);
      return;
    }
    if (event.type === "launch.changed") {
      if (selectedGame && eventMatchesSelectedGame(event)) {
        if (isGameLaunchStatus(event.payload)) gameLaunchStatus = event.payload;
        scheduleSelectedGameRefresh(false, true, "launch.changed", event);
      } else {
        logClientEvent("events refresh skipped", { ...eventDiagnosticDetail(event), reason: "launch event did not match selected game" });
      }
      return;
    }
    if (event.type === "game.changed") {
      void refresh("game.changed");
      return;
    }
    if (event.type === "workshop.changed" && eventMatchesSelectedGame(event)) {
      scheduleSelectedGameRefresh(false, true, "workshop.changed", event);
      return;
    }
    if (["profile_mods.changed", "deployment.changed", "mod_updates.changed"].includes(event.type) && eventMatchesSelectedGame(event)) {
      scheduleSelectedGameRefresh(true, true, event.type, event);
    } else if (["workshop.changed", "profile_mods.changed", "deployment.changed", "mod_updates.changed"].includes(event.type)) {
      logClientEvent("events refresh skipped", { ...eventDiagnosticDetail(event), reason: "game event did not match selected game" });
    }
  }

  function isActionJob(job: Job) {
    return ["captured-install", "installer-choice", "steam-workshop-action"].includes(job.type);
  }

  function eventMatchesSelectedGame(event: DomainEvent) {
    return Boolean(selectedGame && (!event.app_id || event.app_id === selectedGame.app_id));
  }

  function isJob(value: unknown): value is Job {
    return Boolean(value && typeof value === "object" && typeof (value as Job).id === "string" && typeof (value as Job).type === "string");
  }

  function isGameLaunchStatus(value: unknown): value is GameLaunchStatus {
    return Boolean(value && typeof value === "object" && "required" in value && "configured" in value);
  }

  function isUISettings(value: unknown): value is UISettings {
    return Boolean(value && typeof value === "object");
  }

  function scheduleSelectedGameRefresh(refreshPreview = false, refreshJobs = false, reason = "event", event?: DomainEvent) {
    if (!selectedGame || !initialRefreshComplete) return;
    selectedGameRefreshReasons.add(reason);
    selectedGameRefreshNeedsPreview = selectedGameRefreshNeedsPreview || refreshPreview;
    selectedGameRefreshNeedsJobs = selectedGameRefreshNeedsJobs || refreshJobs;
    logClientEvent("selected game refresh scheduled", {
      ...(event ? eventDiagnosticDetail(event) : {}),
      reason,
      refresh_preview: selectedGameRefreshNeedsPreview,
      refresh_jobs: selectedGameRefreshNeedsJobs,
      selected_app_id: selectedGame.app_id
    });
    if (selectedGameRefreshTimer !== null) return;
    selectedGameRefreshTimer = window.setTimeout(async () => {
      const shouldRefreshPreview = selectedGameRefreshNeedsPreview;
      const shouldRefreshJobs = selectedGameRefreshNeedsJobs;
      const reasons = joinedRefreshReasons(selectedGameRefreshReasons) || "event";
      const sequence = ++selectedGameRefreshSequence;
      selectedGameRefreshTimer = null;
      selectedGameRefreshNeedsPreview = false;
      selectedGameRefreshNeedsJobs = false;
      selectedGameRefreshReasons.clear();
      logClientEvent("selected game event refresh started", {
        sequence,
        reasons,
        selected_app_id: selectedGame?.app_id ?? "",
        refresh_preview: shouldRefreshPreview,
        refresh_jobs: shouldRefreshJobs
      });
      try {
        if (shouldRefreshJobs) {
          await refreshActionState(`selected-game:${reasons}:jobs`);
        }
        await refreshSelectedGame({ refreshPreview: shouldRefreshPreview, reason: `event:${reasons}` });
        logClientEvent("selected game event refresh completed", {
          sequence,
          reasons,
          selected_app_id: selectedGame?.app_id ?? "",
          refresh_preview: shouldRefreshPreview,
          refresh_jobs: shouldRefreshJobs
        });
      } catch (err) {
        error = err instanceof Error ? err.message : String(err);
        logClientEvent("selected game event refresh failed", { sequence, reasons, selected_app_id: selectedGame?.app_id ?? "", error: compactLogValue(error) });
      } finally {
        reconcileBusyState();
      }
    }, 250);
  }

  async function selectGame(game: Game) {
    markGameRecent(game.app_id);
    selectedGame = game;
    surface = "game";
    activeGameModule = "plugins";
    drawer = null;
    resolvedCapture = "";
    nexusFiles = [];
    nexusSearchResults = [];
    nexusSearchTotal = 0;
    selectedNexusModID = null;
    nexusFilesByMod = {};
    busyNexusFileKey = "";
    downloadLinks = [];
    deployPlan = null;
    deploymentStatus = null;
    deploymentSettings = null;
    deploymentHistory = [];
    gameDiagnostics = null;
    gameLaunchStatus = null;
    installCandidates = [];
    installerChoicePresets = [];
    await loadGameState(game);
    await previewDeploy();
  }

  function openGameModule(module: GameModule) {
    activeGameModule = module;
    if (module === "actions") {
      scheduleActionStateRefresh(true, false);
    }
  }

  function applyUIPreferences(nextStatus: Status) {
    applyUIPreferencesFromUI(nextStatus.ui);
  }

  function applyUIPreferencesFromUI(ui?: UISettings) {
    favoriteGameIDs = new Set((ui?.favorite_game_ids ?? []).filter((item) => typeof item === "string" && item.trim() !== ""));
    gameRecent = ui?.recent_games ?? {};
    gameSort = ui?.game_sort === "az" || ui?.game_sort === "za" ? ui.game_sort : "recent";
  }

  async function patchGamePreferences(patch: Record<string, string | number | boolean>) {
    const response = await fetch("/api/settings/ui", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(patch)
    });
    if (!response.ok) {
      logClientEvent("ui preferences save failed", { status: response.status });
      return;
    }
    const nextStatus = await response.json() as Status;
    status = nextStatus;
    applyUIPreferences(nextStatus);
  }

  function setGameSort(nextSort: GameSort) {
    gameSort = nextSort;
    void patchGamePreferences({ game_sort: nextSort });
  }

  function isFavoriteGame(appID: string) {
    return favoriteGameIDs.has(appID);
  }

  function toggleFavoriteGame(appID: string) {
    const next = new Set(favoriteGameIDs);
    if (next.has(appID)) {
      next.delete(appID);
    } else {
      next.add(appID);
    }
    favoriteGameIDs = next;
    void patchGamePreferences({ favorite_game_id: appID, favorite: next.has(appID) });
  }

  function markGameRecent(appID: string) {
    const next = { ...gameRecent, [appID]: Date.now() };
    gameRecent = Object.fromEntries(Object.entries(next).sort((a, b) => b[1] - a[1]).slice(0, 50));
    void patchGamePreferences({ recent_game_id: appID, recent_at: gameRecent[appID] });
  }

  function openSettings(page: SettingsPage) {
    activeSettingsPage = page;
    surface = "settings";
    drawer = null;
  }

  function openActionCenter() {
    surface = "actions";
    drawer = null;
  }

  async function loadProfiles(game: Game) {
    profiles = await getJSON<Profile[]>(`/api/games/${game.app_id}/profiles`);
  }

  async function loadInstalledMods(game: Game) {
    installedMods = await getJSON<InstalledMod[]>(`/api/games/${game.app_id}/mods`);
  }

  async function loadGameState(game: Game) {
    const [nextProfiles, nextMods, nextCandidates, nextPresets, nextDeploymentStatus, nextDeploymentSettings, nextDeploymentHistory, nextPluginLoadOrder, nextDiagnostics, nextLaunchStatus, nextWorkshopState] = await Promise.all([
      getJSON<Profile[]>(`/api/games/${game.app_id}/profiles`),
      getJSON<InstalledMod[]>(`/api/games/${game.app_id}/mods`),
      getJSON<InstallCandidate[]>(`/api/games/${game.app_id}/install-candidates`),
      getJSON<InstallerChoicePreset[]>(`/api/games/${game.app_id}/installer-choice-presets`),
      getJSON<DeploymentStatus>(`/api/games/${game.app_id}/deploy/status`),
      getJSON<DeploymentSettings>(`/api/games/${game.app_id}/deploy/settings`),
      getJSON<{ deployments: DeploymentHistoryItem[] }>(`/api/games/${game.app_id}/deploy/history?limit=5`),
      getJSON<PluginLoadOrder>(`/api/games/${game.app_id}/load-order`),
      getJSON<GameDiagnostics>(`/api/games/${game.app_id}/diagnostics`),
      getJSON<GameLaunchStatus>(`/api/games/${game.app_id}/launch`),
      getJSON<WorkshopState>(`/api/games/${game.app_id}/workshop`)
    ]);
    profiles = nextProfiles;
    installedMods = nextMods;
    installCandidates = nextCandidates;
    installerChoicePresets = nextPresets;
    deploymentStatus = nextDeploymentStatus;
    deploymentSettings = nextDeploymentSettings;
    deploymentHistory = nextDeploymentHistory.deployments ?? [];
    pluginLoadOrder = nextPluginLoadOrder;
    gameDiagnostics = nextDiagnostics;
    gameLaunchStatus = nextLaunchStatus;
    workshopState = nextWorkshopState;
    workshopItems = nextWorkshopState.items ?? [];
    globalInstallCandidates = mergeInstallCandidatesForGame(globalInstallCandidates, game.app_id, nextCandidates);
    reconcileBusyState();
  }

  async function refreshSelectedGame(options: { refreshPreview?: boolean; refreshJobs?: boolean; reason?: string } = {}) {
    if (!selectedGame) return;
    const sequence = ++selectedGameRefreshSequence;
    const appID = selectedGame.app_id;
    const reason = options.reason ?? "manual";
    logClientEvent("selected game refresh started", {
      sequence,
      reason,
      selected_app_id: appID,
      refresh_preview: Boolean(options.refreshPreview),
      refresh_jobs: Boolean(options.refreshJobs)
    });
    if (options.refreshJobs) await refreshActionState(`${reason}:selected-game-jobs`);
    await loadGameState(selectedGame);
    if (options.refreshPreview) await previewDeploy();
    logClientEvent("selected game refresh completed", {
      sequence,
      reason,
      selected_app_id: selectedGame?.app_id ?? appID,
      mods: installedMods.length,
      candidates: installCandidates.length,
      workshop_items: workshopItems.length,
      preview_actions: deployableActions.length
    });
  }

  async function updateDeploymentStrategy(strategy: string) {
    if (!selectedGame) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/deploy/settings`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ strategy, profile_id: selectedProfile?.id, scope: "profile" })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    deploymentSettings = await response.json();
    await refreshSelectedGame({ refreshPreview: true });
  }

  async function updateDownloadConcurrency(maxDownloads: number) {
    error = "";
    const response = await fetch("/api/settings/downloads", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ max_concurrent_captured_downloads: maxDownloads })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const nextStatus = await response.json() as Status;
    status = nextStatus;
    applyUIPreferences(nextStatus);
  }

  async function updatePerGameDownloadConcurrency(maxDownloads: number) {
    error = "";
    const response = await fetch("/api/settings/downloads", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ max_concurrent_captured_downloads_per_game: maxDownloads })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const nextStatus = await response.json() as Status;
    status = nextStatus;
    applyUIPreferences(nextStatus);
  }

  async function updateCatalogCredential(provider: "modio" | "curseforge", apiKey: string) {
    catalogSettingsBusy = provider;
    catalogSettingsMessage = "";
    error = "";
    const body = provider === "modio"
      ? { modio: { api_key: apiKey } }
      : { curseforge: { api_key: apiKey } };
    try {
      const response = await fetch("/api/settings/catalogs", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body)
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json() as { catalogs: CatalogStatus[] };
      catalogs = result.catalogs ?? [];
      if (provider === "modio") modIOAPIKey = "";
      if (provider === "curseforge") curseForgeAPIKey = "";
      catalogSettingsMessage = `${provider === "modio" ? "mod.io" : "CurseForge"} key saved.`;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      catalogSettingsBusy = "";
    }
  }

  async function createProfile() {
    if (!selectedGame || !profileName.trim()) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/profiles`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: profileName })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    profileName = "";
    await loadProfiles(selectedGame);
  }

  async function setDefaultProfile(profile: Profile) {
    if (!selectedGame || profile.is_default) return;
    error = "";
    const response = await fetch(`/api/profiles/${profile.id}/default`, { method: "PUT" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result: SetDefaultProfileResult = await response.json();
    handleProfileApplyResult(result.apply);
    await loadProfiles(selectedGame);
    await loadInstalledMods(selectedGame);
    await refreshSelectedGame({ refreshPreview: true });
  }

  async function setModEnabled(mod: InstalledMod, enabled: boolean) {
    if (!selectedProfile) return;
    error = "";
    setBusyMod(mod.id, "toggle");
    try {
      const response = await fetch(`/api/profiles/${selectedProfile.id}/mods/${mod.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: ProfileModUpdateResult = await response.json();
      installedMods = installedMods.map((item) => (item.id === result.mod.id ? result.mod : item));
      handleProfileApplyResult(result.apply);
      await refreshSelectedGame({ refreshPreview: true });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      clearBusyMod(mod.id);
    }
  }

  async function moveModInProfile(mod: InstalledMod, direction: -1 | 1) {
    if (!selectedProfile) return;
    const current = [...installedMods].sort((a, b) => a.priority - b.priority || a.name.localeCompare(b.name));
    const from = current.findIndex((item) => item.id === mod.id);
    const to = from + direction;
    if (from < 0 || to < 0 || to >= current.length) return;
    [current[from], current[to]] = [current[to], current[from]];
    error = "";
    const response = await fetch(`/api/profiles/${selectedProfile.id}/mods/order`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ mod_ids: current.map((item) => item.id) })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result: ProfileModOrderUpdateResult = await response.json();
    installedMods = result.mods.sort((a, b) => a.priority - b.priority || a.name.localeCompare(b.name));
    handleProfileApplyResult(result.apply);
    await refreshSelectedGame({ refreshPreview: true });
  }

  async function removeInstalledMod(mod: InstalledMod) {
    if (!selectedGame) return;
    error = "";
    setBusyMod(mod.id, "remove");
    try {
      const response = await fetch(`/api/games/${selectedGame.app_id}/mods/${mod.id}`, { method: "DELETE" });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: { removed: InstalledMod; apply: ProfileApplyResult } = await response.json();
      installedMods = installedMods.filter((item) => item.id !== mod.id);
      handleProfileApplyResult(result.apply);
      await refreshSelectedGame({ refreshPreview: true });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      clearBusyMod(mod.id);
    }
  }

  async function reinstallInstalledMod(mod: InstalledMod) {
    if (!selectedGame) return;
    error = "";
    setBusyMod(mod.id, "reinstall");
    try {
      const response = await fetch(`/api/games/${selectedGame.app_id}/mods/${mod.id}/reinstall`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: { job?: Job; mod?: InstalledMod } = await response.json();
      if (result.job) upsertJob(result.job);
      if (result.mod) {
        installedMods = installedMods.map((item) => (item.id === mod.id ? result.mod as InstalledMod : item));
        if (!installedMods.some((item) => item.id === result.mod?.id)) installedMods = [...installedMods, result.mod];
      }
      await refreshSelectedGame({ refreshPreview: true, refreshJobs: true });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      clearBusyMod(mod.id);
    }
  }

  async function updateInstalledMod(mod: InstalledMod) {
    if (!selectedGame || mod.update?.status !== "available") return;
    error = "";
    modUpdateMessage = "";
    modUpdateBrowserURL = "";
    setBusyMod(mod.id, "update");
    try {
      const response = await fetch(`/api/games/${selectedGame.app_id}/mods/${mod.id}/update`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        await refreshJobsAndSelectedGame("mod-update-error");
        return;
      }
      const result: { job?: Job; browser_required?: boolean; file_url?: string; resolved?: { source_url?: string } } = await response.json();
      if (result.job) {
        upsertJob(result.job);
        modUpdateMessage = result.job.message || result.job.title;
        if (result.job.status === "failed" && !result.browser_required) {
          error = result.job.message || "Unable to install this update.";
        }
      }
      if (result.browser_required) {
        modUpdateBrowserURL = result.file_url ?? result.resolved?.source_url ?? "";
        modUpdateMessage = "Open the provider file page and use its Mod Manager Download flow for this update.";
        if (!modUpdateBrowserURL) error = modUpdateMessage;
      }
      await refreshJobsAndSelectedGame("mod-update");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      clearBusyMod(mod.id);
    }
  }

  async function checkModUpdates() {
    if (!selectedGame || modUpdateBusy) return;
    error = "";
    modUpdateMessage = "";
    modUpdateBrowserURL = "";
    modUpdateBusy = true;
    const response = await fetch(`/api/games/${selectedGame.app_id}/mods/check-updates`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      modUpdateBusy = false;
      return;
    }
    const result: { checked: number; results: Array<{ installed_mod_id: number } & ModUpdate> } = await response.json();
    const updates = new Map(result.results.map((item) => [item.installed_mod_id, {
      catalog: item.catalog,
      source_tag: item.source_tag,
      status: item.status,
      latest_file_id: item.latest_file_id,
      latest_file_name: item.latest_file_name,
      latest_version: item.latest_version,
      latest_uploaded_at: item.latest_uploaded_at,
      message: item.message,
      checked_at: item.checked_at
    } as ModUpdate]));
    installedMods = installedMods.map((mod) => updates.has(mod.id) ? { ...mod, update: updates.get(mod.id) } : mod);
    const available = result.results.filter((item) => item.status === "available").length;
    modUpdateMessage = available > 0 ? `${available} update${available === 1 ? "" : "s"} available.` : `Checked ${result.checked} supported mod${result.checked === 1 ? "" : "s"}.`;
    modUpdateBusy = false;
    await refreshSelectedGame();
  }

  function handleProfileApplyResult(result: ProfileApplyResult | null | undefined) {
    if (!result) return;
    if (result.job) upsertJob(result.job);
    if (result.plan) deployPlan = result.plan;
    if (result.status === "blocked" || result.status === "failed") {
      error = result.message;
    }
  }

  function askRemoveInstalledMod(mod: InstalledMod) {
    confirmation = {
      title: "Remove profile mod",
      message: `Remove ${mod.name} from this profile. The cached download is kept so it can be recovered later.`,
      detail: `${mod.source_game_domain}/mods/${mod.source_mod_id}/files/${mod.source_file_id}`,
      confirmLabel: "Remove Mod",
      danger: true,
      run: () => removeInstalledMod(mod)
    };
  }

  async function resolveCapturedInstall() {
    if (!selectedGame || !captureURL.trim()) return;
    error = "";
    lastCaptureURL = captureURL;
    nexusFiles = [];
    downloadLinks = [];
    const response = await fetch("/api/captured-installs/resolve", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url: captureURL, steam_app_id: selectedGame.app_id })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    resolvedCapture = `${result.resolved.catalog}:${result.resolved.game_domain || result.resolved.steam_app_id}/mods/${result.resolved.mod_id}${result.resolved.file_id ? `/files/${result.resolved.file_id}` : ""}`;
    nexusFiles = result.files ?? [];
    downloadLinks = result.download_links ?? [];
    captureURL = "";
  }

  async function resolveFile(file: NexusFile) {
    if (!lastCaptureURL) return;
    const nextURL = new URL(lastCaptureURL);
    nextURL.searchParams.set("file_id", String(file.file_id));
    captureURL = nextURL.toString();
    await resolveCapturedInstall();
  }

  function handleLocalArchiveChange(event: Event) {
    const input = event.currentTarget as HTMLInputElement;
    localArchiveFile = input.files?.[0] ?? null;
    localArchiveMessage = "";
  }

  async function uploadLocalArchive() {
    if (!selectedGame || !localArchiveFile) return;
    error = "";
    localArchiveMessage = "";
    localArchiveBusy = true;
    try {
      const form = new FormData();
      form.append("archive", localArchiveFile);
      const response = await fetch(`/api/games/${selectedGame.app_id}/local-archives`, {
        method: "POST",
        body: form
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      localArchiveMessage = result.install_started ? "Upload received; installing archive." : "Upload received; install it from Action Center.";
      localArchiveFile = null;
      if (localArchiveInput) localArchiveInput.value = "";
      await refreshJobsAndSelectedGame("local-archive");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      localArchiveBusy = false;
    }
  }

  function selectedNexusDomain() {
    return selectedGame?.nexus_domains?.[0] ?? "";
  }

  function nexusFileURL(modID: number, fileID: number) {
    const domain = selectedNexusDomain();
    return `https://www.nexusmods.com/${encodeURIComponent(domain)}/mods/${modID}?file_id=${fileID}`;
  }

  function openNexusFilePage(modID: number, fileID: number) {
    window.open(nexusFileURL(modID, fileID), "_blank", "noopener");
  }

  function nextNexusSort(current: NexusSearchSort): NexusSearchSort {
    if (current === "downloads") return "unique_downloads";
    if (current === "unique_downloads") return "popular";
    if (current === "popular") return "updated";
    if (current === "updated") return "name";
    if (current === "name") return "relevance";
    return "downloads";
  }

  function nexusSortLabel(sort: NexusSearchSort) {
    if (sort === "unique_downloads") return "Unique Downloads";
    if (sort === "popular") return "Popular";
    if (sort === "updated") return "Updated";
    if (sort === "name") return "Name";
    if (sort === "relevance") return "Relevance";
    return "Downloads";
  }

  function nextNexusTimeWindow(current: NexusTimeWindow): NexusTimeWindow {
    if (current === "all") return "one_week";
    if (current === "one_week") return "three_weeks";
    if (current === "three_weeks") return "one_month";
    if (current === "one_month") return "three_months";
    if (current === "three_months") return "one_year";
    return "all";
  }

  function nexusTimeWindowLabel(window: NexusTimeWindow) {
    if (window === "one_week") return "1 Week";
    if (window === "three_weeks") return "3 Weeks";
    if (window === "one_month") return "1 Month";
    if (window === "three_months") return "3 Months";
    if (window === "one_year") return "1 Year";
    return "All Time";
  }

  function compactNumber(value: number | undefined) {
    if (!value) return "0";
    if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
    if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
    return String(value);
  }

  function formatBytes(value: number | undefined) {
    if (!value || value < 0) return "unknown size";
    if (value >= 1024 * 1024 * 1024) return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
    if (value >= 1024 * 1024) return `${(value / (1024 * 1024)).toFixed(1)} MB`;
    if (value >= 1024) return `${(value / 1024).toFixed(1)} KB`;
    return `${value} B`;
  }

  async function searchNexusMods(nextSort = nexusSearchSort, nextWindow = nexusSearchTimeWindow) {
    if (!selectedGame) return;
    nexusSearchBusy = true;
    nexusSearchError = "";
    try {
      const params = new URLSearchParams({
        q: nexusSearchQuery,
        sort: nextSort,
        time_window: nextWindow,
        count: "20",
        offset: "0",
        vortex_only: nexusSearchVortexOnly ? "true" : "false"
      });
      const result = await getJSON<{ mods: NexusModResult[]; total_count: number }>(`/api/games/${selectedGame.app_id}/nexus/mods?${params.toString()}`);
      nexusSearchResults = result.mods ?? [];
      nexusSearchTotal = result.total_count ?? nexusSearchResults.length;
      if (nexusSearchResults.length === 0) {
        nexusSearchError = nexusSearchVortexOnly ? "No Vortex-compatible Nexus mods matched this search." : "No Nexus mods matched this search.";
      }
    } catch (err) {
      nexusSearchError = err instanceof Error ? err.message : String(err);
      nexusSearchResults = [];
      nexusSearchTotal = 0;
    } finally {
      nexusSearchBusy = false;
    }
  }

  function cycleNexusSort() {
    const next = nextNexusSort(nexusSearchSort);
    nexusSearchSort = next;
    void searchNexusMods(next, nexusSearchTimeWindow);
  }

  function cycleNexusTimeWindow() {
    const next = nextNexusTimeWindow(nexusSearchTimeWindow);
    nexusSearchTimeWindow = next;
    void searchNexusMods(nexusSearchSort, next);
  }

  function toggleNexusCompatibilityFilter() {
    nexusSearchVortexOnly = !nexusSearchVortexOnly;
    void searchNexusMods();
  }

  async function loadNexusModFiles(mod: NexusModResult) {
    if (!selectedGame) return;
    selectedNexusModID = mod.mod_id;
    nexusSearchError = "";
    if (nexusFilesByMod[mod.mod_id]) return;
    busyNexusFileKey = `files:${mod.mod_id}`;
    try {
      const result = await getJSON<{ files: NexusFile[] }>(`/api/games/${selectedGame.app_id}/nexus/mods/${mod.mod_id}/files`);
      nexusFilesByMod = { ...nexusFilesByMod, [mod.mod_id]: result.files ?? [] };
      if ((result.files ?? []).length === 0) nexusSearchError = "This Nexus mod did not return installable files.";
    } catch (err) {
      nexusSearchError = err instanceof Error ? err.message : String(err);
    } finally {
      busyNexusFileKey = "";
    }
  }

  async function addNexusSearchFile(mod: NexusModResult, file: NexusFile) {
    if (!selectedGame || !selectedNexusDomain()) return;
    const key = `${mod.mod_id}:${file.file_id}`;
    busyNexusFileKey = key;
    nexusSearchError = "";
    try {
      const response = await fetch("/api/captured-installs/resolve", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url: nexusFileURL(mod.mod_id, file.file_id), steam_app_id: selectedGame.app_id })
      });
      if (!response.ok) {
        nexusSearchError = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      resolvedCapture = `${selectedNexusDomain()}/mods/${mod.mod_id}/files/${file.file_id}`;
      await refreshJobsAndSelectedGame("nexus-search-file");
    } catch (err) {
      nexusSearchError = err instanceof Error ? err.message : String(err);
    } finally {
      busyNexusFileKey = "";
    }
  }

  async function clearCapturedInstallActions() {
    error = "";
    const response = await fetch("/api/captured-installs", { method: "DELETE" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    jobs = jobs.filter((job) => job.type !== "captured-install" || ["completed", "canceled"].includes(job.status));
    await refresh();
  }

  async function clearBlockedInstallCandidates() {
    if (!selectedGame) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/install-candidates`, { method: "DELETE" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    installCandidates = [];
    globalInstallCandidates = globalInstallCandidates.filter((candidate) => candidate.steam_app_id !== selectedGame.app_id);
    candidateSelections = {};
    await refreshSelectedGame({ refreshPreview: deployPlan !== null });
  }

  async function deleteInstallerChoicePreset(preset: InstallerChoicePreset) {
    if (!selectedGame) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/installer-choice-presets/${preset.id}`, { method: "DELETE" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    installerChoicePresets = installerChoicePresets.filter((item) => item.id !== preset.id);
    await refreshSelectedGame();
  }

  function installerForCandidate(candidate: InstallCandidate): FomodInstaller | null {
    if (!candidate.installer_json) return null;
    try {
      return JSON.parse(candidate.installer_json) as FomodInstaller;
    } catch (_err) {
      return null;
    }
  }

  function storedCandidateSelections(candidate: InstallCandidate) {
    if (!candidate.choices_json) return null;
    try {
      const parsed = JSON.parse(candidate.choices_json) as Record<string, string[]>;
      return parsed && typeof parsed === "object" ? parsed : null;
    } catch (_err) {
      return null;
    }
  }

  function selectionCount(selections: Record<string, string[]> | null | undefined) {
    if (!selections) return 0;
    return Object.values(selections).reduce((total, group) => total + (Array.isArray(group) ? group.length : 0), 0);
  }

  function fomodGroupType(group: FomodGroup) {
    return (group.type ?? "").trim().toLowerCase();
  }

  function fomodGroupInputType(group: FomodGroup) {
    const type = fomodGroupType(group);
    return type === "selectexactlyone" || type === "selectatmostone" ? "radio" : "checkbox";
  }

  function visibleFomodSteps(installer: FomodInstaller) {
    return (installer.steps ?? []).filter((step) => step.visible !== false);
  }

  function installerRequiresSelections(installer: FomodInstaller) {
    return visibleFomodSteps(installer).some((step) => (step.groups ?? []).some((group) => (group.plugins ?? []).length > 0));
  }

  function candidateCurrentSelections(candidate: InstallCandidate) {
    return candidateSelections[candidate.id] ?? storedCandidateSelections(candidate) ?? {};
  }

  function isCandidatePluginSelected(candidate: InstallCandidate, group: FomodGroup, plugin: FomodPlugin) {
    return (candidateCurrentSelections(candidate)[group.id] ?? []).includes(plugin.id);
  }

  function fomodPluginSelectable(plugin: FomodPlugin) {
    return fomodPluginType(plugin).toLowerCase() !== "notusable";
  }

  function fomodPluginLocked(group: FomodGroup, plugin: FomodPlugin) {
    const type = fomodPluginType(plugin).toLowerCase();
    return fomodGroupType(group) === "selectall" || type === "required" || type === "notusable";
  }

  function fomodPluginType(plugin: FomodPlugin) {
    return (plugin.effective_type ?? plugin.type ?? "").trim();
  }

  function clearCandidateGroupSelection(candidate: InstallCandidate, group: FomodGroup) {
    const current = candidateCurrentSelections(candidate);
    const next = { ...current, [group.id]: [] };
    candidateSelections = { ...candidateSelections, [candidate.id]: next };
    void saveCandidateSelections(candidate, next);
  }

  function setCandidatePluginSelection(candidate: InstallCandidate, group: FomodGroup, plugin: FomodPlugin, checked: boolean) {
    const current = candidateCurrentSelections(candidate);
    const next = { ...current };
    const type = fomodGroupType(group);
    if (fomodPluginLocked(group, plugin)) return;
    if (type === "selectexactlyone" || type === "selectatmostone") {
      next[group.id] = checked ? [plugin.id] : [];
    } else {
      const selected = new Set(next[group.id] ?? []);
      if (checked) selected.add(plugin.id);
      else selected.delete(plugin.id);
      next[group.id] = Array.from(selected);
    }
    candidateSelections = { ...candidateSelections, [candidate.id]: next };
    void saveCandidateSelections(candidate, next);
  }

  function candidateStepIndex(candidate: InstallCandidate, steps: FomodStep[]) {
    if (steps.length === 0) return 0;
    const requested = candidateStepIndices[candidate.id] ?? 0;
    return Math.max(0, Math.min(requested, steps.length - 1));
  }

  function setCandidateStepIndex(candidate: InstallCandidate, steps: FomodStep[], index: number) {
    const nextIndex = steps.length === 0 ? 0 : Math.max(0, Math.min(index, steps.length - 1));
    candidateStepIndices = { ...candidateStepIndices, [candidate.id]: nextIndex };
  }

  function candidateGroupValid(candidate: InstallCandidate, group: FomodGroup) {
    const selected = (candidateCurrentSelections(candidate)[group.id] ?? []).filter((id) => {
      const plugin = (group.plugins ?? []).find((item) => item.id === id);
      return plugin ? fomodPluginSelectable(plugin) : false;
    });
    const selectableCount = (group.plugins ?? []).filter(fomodPluginSelectable).length;
    switch (fomodGroupType(group)) {
      case "selectall":
        return selected.length === selectableCount;
      case "selectexactlyone":
        return selected.length === 1;
      case "selectatleastone":
        return selected.length >= 1;
      case "selectatmostone":
        return selected.length <= 1;
      default:
        return true;
    }
  }

  function candidateStepValid(candidate: InstallCandidate, step: FomodStep | undefined) {
    if (!step) return true;
    return (step.groups ?? []).every((group) => candidateGroupValid(candidate, group));
  }

  function candidateInstallerValid(candidate: InstallCandidate, installer: FomodInstaller | null) {
    if (!installer) return false;
    return visibleFomodSteps(installer).every((step) => candidateStepValid(candidate, step));
  }

  async function saveCandidateSelections(candidate: InstallCandidate, selections: Record<string, string[]>) {
    if (!selectedGame) return;
    try {
      const response = await fetch(`/api/games/${selectedGame.app_id}/install-candidates/${candidate.id}/choices`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ selections })
      });
      if (!response.ok) {
        logClientEvent("installer choices save failed", { candidate_id: candidate.id, status: response.status });
        return;
      }
      const result = await response.json() as { candidate?: InstallCandidate };
      if (result.candidate) replaceInstallCandidate(result.candidate);
    } catch (err) {
      logClientEvent("installer choices save failed", { candidate_id: candidate.id, error: err instanceof Error ? err.message : String(err) });
    }
  }

  async function applyInstallCandidate(candidate: InstallCandidate) {
    if (!selectedGame) return;
    const installer = installerForCandidate(candidate);
    if (!installer) {
      error = "Installer choices are not available for this candidate.";
      return;
    }
    error = "";
    setInstallCandidateBusy(candidate.id, true);
    try {
      const selections = candidateCurrentSelections(candidate);
      if (installerRequiresSelections(installer) && Object.keys(selections).length === 0) {
        error = "Installer choices are missing from backend state. Retry this installer item so DMM can rebuild the choices.";
        return;
      }
      const response = await fetch(`/api/games/${selectedGame.app_id}/install-candidates/${candidate.id}/apply`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ selections })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      if (result.mod) {
        installedMods = [result.mod, ...installedMods.filter((mod) => mod.id !== result.mod.id)];
      }
      installCandidates = installCandidates.filter((item) => item.id !== candidate.id);
      globalInstallCandidates = globalInstallCandidates.filter((item) => item.id !== candidate.id);
      candidateSelections = Object.fromEntries(Object.entries(candidateSelections).filter(([id]) => Number(id) !== candidate.id));
      candidateStepIndices = Object.fromEntries(Object.entries(candidateStepIndices).filter(([id]) => Number(id) !== candidate.id));
      await refreshJobsAndSelectedGame("installer-choice-apply", true);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setInstallCandidateBusy(candidate.id, false);
    }
  }

  async function retryInstallCandidate(candidate: InstallCandidate) {
    if (!selectedGame) return;
    error = "";
    setInstallCandidateBusy(candidate.id, true);
    try {
      const response = await fetch(`/api/games/${selectedGame.app_id}/install-candidates/${candidate.id}/retry`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      if (result.mod) {
        installedMods = [result.mod, ...installedMods.filter((mod) => mod.id !== result.mod.id)];
        installCandidates = installCandidates.filter((item) => item.id !== candidate.id);
        globalInstallCandidates = globalInstallCandidates.filter((item) => item.id !== candidate.id);
      }
      await refreshJobsAndSelectedGame("installer-choice-retry", true);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setInstallCandidateBusy(candidate.id, false);
    }
  }

  async function cancelJob(job: Job) {
    if (!canCancelJob(job)) return;
    error = "";
    setJobBusy(job.id, true);
    markJobProcessing(job, "Canceling job...");
    try {
      const response = await fetch(`/api/jobs/${job.id}/cancel`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        await refreshJobsAndSelectedGame("job-cancel-error");
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      await refreshJobsAndSelectedGame("job-cancel");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setJobBusy(job.id, false);
    }
  }

  async function installCapturedMod(action: Job) {
    if (action.status !== "waiting") return;
    error = "";
    setJobBusy(action.id, true);
    markJobProcessing(action, "Installing downloaded archive...");
    try {
      const response = await fetch(`/api/captured-installs/${action.id}/install`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        await refreshJobsAndSelectedGame("captured-install-confirm-error", true);
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      await refreshJobsAndSelectedGame("captured-install-confirm", true);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setJobBusy(action.id, false);
    }
  }

  async function retryCapturedInstallAction(action: Job) {
    if (action.status !== "failed") return;
    error = "";
    setJobBusy(action.id, true);
    markJobProcessing(action, "Retrying captured install...");
    try {
      const response = await fetch(`/api/captured-installs/${action.id}/retry`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        await refreshJobsAndSelectedGame("captured-install-retry-error", true);
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      await refreshJobsAndSelectedGame("captured-install-retry", true);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setJobBusy(action.id, false);
    }
  }

  async function retryWorkshopAction(action: Job) {
    if (action.type !== "steam-workshop-action" || action.status !== "failed") return;
    error = "";
    setJobBusy(action.id, true);
    markJobProcessing(action, "Waiting for Decky to retry this Workshop action...");
    try {
      const response = await fetch(`/api/workshop/actions/${action.id}/retry`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        await refreshJobsAndSelectedGame("workshop-action-retry-error");
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      await refreshJobsAndSelectedGame("workshop-action-retry");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setJobBusy(action.id, false);
    }
  }

  async function previewDeploy() {
    if (!selectedGame) return;
    error = "";
    deployPlan = null;
    const response = await fetch(`/api/games/${selectedGame.app_id}/deploy/preview`);
    if (!response.ok) {
      error = await response.text();
      return;
    }
    deployPlan = await response.json();
  }

  async function ensureDeployPlan() {
    if (deployPlan) return deployPlan;
    if (!selectedGame) return null;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/deploy/preview`);
    if (!response.ok) {
      error = await response.text();
      return null;
    }
    deployPlan = await response.json();
    return deployPlan;
  }

  async function applyPendingProfileChanges() {
    if (!selectedGame || !deployPlan || deployPlan.conflicts.length > 0 || deployableActions.length === 0) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/deploy`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    deployPlan = result.plan;
    await refreshSelectedGame({ refreshPreview: true });
  }

  async function askApplyPendingProfileChanges() {
    if (!selectedGame) return;
    const plan = await ensureDeployPlan();
    if (!plan) return;
    const actions = getDeployableActions(plan);
    if (plan.conflicts.length > 0 || actions.length === 0) return;
    const adds = actions.filter((action) => action.operation === "add").length;
    const replaces = actions.filter((action) => action.operation === "replace").length;
    const removes = actions.filter((action) => action.operation === "remove").length;
    confirmation = {
      title: "Apply enabled mods",
      message: `DMM will update ${selectedGame.name}'s game folder to match the enabled mods in the selected profile.`,
      detail: `${adds} add, ${replaces} update, ${removes} remove. Advanced file details remain available before or after applying.`,
      confirmLabel: "Apply Enabled Mods",
      run: applyPendingProfileChanges
    };
  }

  async function setFileConflictWinner(target: ConflictChoiceTarget, winnerInstalledModID: number) {
    if (!selectedProfile) return;
    error = "";
    const response = await fetch(`/api/profiles/${selectedProfile.id}/conflicts/winner`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        target_path: target.target_path,
        winner_installed_mod_id: winnerInstalledModID
      })
    });
    if (!response.ok) {
      error = await response.text();
      await refreshSelectedGame({ refreshPreview: true });
      return;
    }
    const result = await response.json();
    if (result.apply?.plan) deployPlan = result.apply.plan;
    await refreshSelectedGame({ refreshPreview: true });
  }

  async function clearFileConflictWinner(target: ConflictChoiceTarget) {
    if (!selectedProfile) return;
    error = "";
    const response = await fetch(`/api/profiles/${selectedProfile.id}/conflicts/winner?target_path=${encodeURIComponent(target.target_path)}`, {
      method: "DELETE"
    });
    if (!response.ok) {
      error = await response.text();
      await refreshSelectedGame({ refreshPreview: true });
      return;
    }
    const result = await response.json();
    if (result.apply?.plan) deployPlan = result.apply.plan;
    await refreshSelectedGame({ refreshPreview: true });
  }

  async function purgeDeployment() {
    if (!selectedGame || !deploymentStatus?.deployed) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/deploy`, { method: "DELETE" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    deployPlan = null;
    await refreshSelectedGame();
  }

  async function resetManagedMods() {
    if (!selectedGame) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/reset`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    if (result.job) upsertJob(result.job);
    deployPlan = null;
    await refreshSelectedGame({ refreshPreview: true });
  }

  function askPurgeDeployment() {
    if (!selectedGame || !deploymentStatus?.deployed) return;
    confirmation = {
      title: "Remove DMM-applied files",
      message: `DMM will remove only the files it applied to ${selectedGame.name}.`,
      detail: `${deploymentStatus.file_count} DMM-owned file${deploymentStatus.file_count === 1 ? "" : "s"} will be removed. Unmanaged files and parent game directories are left alone.`,
      confirmLabel: "Remove DMM Files",
      danger: true,
      run: purgeDeployment
    };
  }

  function askResetManagedMods() {
    if (!selectedGame) return;
    confirmation = {
      title: "Reset managed mods",
      message: `DMM will remove its managed mods for ${selectedGame.name}.`,
      detail: "This removes DMM-applied files, removes installed mod rows, deletes local installed mod files, and clears pending installer work. Cached downloads are kept for recovery.",
      confirmLabel: "Reset Managed Mods",
      danger: true,
      run: resetManagedMods
    };
  }

  async function repairDeployment() {
    if (!selectedGame || !deploymentStatus?.deployed) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/deploy/repair`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    if (deployPlan) await previewDeploy();
  }

  async function restoreDeployment() {
    if (!selectedGame || !deploymentStatus?.restore_available) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/deploy/restore`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    await refreshSelectedGame({ refreshPreview: deployPlan !== null });
  }

  function askRestoreDeployment() {
    if (!selectedGame || !deploymentStatus?.restore_available) return;
    confirmation = {
      title: "Restore last applied state",
      message: `DMM will restore ${selectedGame.name}'s DMM-owned files to the last applied state.`,
      detail: deploymentStatus.restore_summary ?? "Only files recorded by DMM are touched. Unmanaged files are left alone.",
      confirmLabel: "Restore State",
      run: restoreDeployment
    };
  }

  async function applyLaunchSetup() {
    if (!selectedGame || !launchSetupAvailable) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/launch/apply`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    if (result.job) upsertJob(result.job);
    gameLaunchStatus = result.status;
    await refreshSelectedGame({ refreshPreview: deployPlan !== null });
  }

  async function recoverDownloads() {
    if (!selectedGame) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/mods/recover-downloads`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    await refreshSelectedGame();
    deployPlan = null;
  }

  function workshopItemName(item: WorkshopItem) {
    return (item.title || item.published_file_id || "Workshop item").trim();
  }

  function workshopItemStatus(item: WorkshopItem) {
    if (!item.disabled_known) return "Steam managed";
    return item.disabled_locally ? "Disabled" : "Enabled";
  }

  function workshopItemDetail(item: WorkshopItem) {
    const installState = item.downloaded ? "Downloaded" : item.subscribed ? "Subscribed" : "Not subscribed";
    return `${installState} · ${item.published_file_id}`;
  }

  function workshopActionBusyKey(item: WorkshopItem, kind: string) {
    return `${item.published_file_id}:${kind}`;
  }

  function setWorkshopActionBusy(item: WorkshopItem, kind: string, busy: boolean) {
    const key = workshopActionBusyKey(item, kind);
    if (busy) {
      busyWorkshopActions = { ...busyWorkshopActions, [key]: true };
      return;
    }
    const next = { ...busyWorkshopActions };
    delete next[key];
    busyWorkshopActions = next;
  }

  function isWorkshopActionBusy(item: WorkshopItem, kind: string) {
    return Boolean(busyWorkshopActions[workshopActionBusyKey(item, kind)]);
  }

  async function queueWorkshopAction(item: WorkshopItem, kind: "enable" | "disable" | "unsubscribe") {
    if (!selectedGame || !workshopState?.supported) return;
    error = "";
    setWorkshopActionBusy(item, kind, true);
    try {
      const response = await fetch(
        `/api/games/${encodeURIComponent(selectedGame.app_id)}/workshop/items/${encodeURIComponent(item.published_file_id)}/actions/${kind}`,
        { method: "POST" }
      );
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      await refreshJobsAndSelectedGame("workshop-action");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setWorkshopActionBusy(item, kind, false);
    }
  }

  async function moveWorkshopItem(index: number, direction: -1 | 1) {
    if (!selectedGame || !workshopState?.supported || workshopOrderBusy) return;
    const to = index + direction;
    if (index < 0 || to < 0 || to >= workshopItems.length) return;
    const nextItems = workshopItems.map((item) => ({ ...item }));
    [nextItems[index], nextItems[to]] = [nextItems[to], nextItems[index]];
    const itemIDs = nextItems.map((item) => item.published_file_id);
    error = "";
    workshopOrderBusy = true;
    try {
      const response = await fetch(`/api/games/${encodeURIComponent(selectedGame.app_id)}/workshop/order`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ item_ids: itemIDs })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      workshopItems = nextItems.map((item, position) => ({ ...item, position }));
      await refreshJobsAndSelectedGame("workshop-order");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      workshopOrderBusy = false;
    }
  }

  function askUnsubscribeWorkshopItem(item: WorkshopItem) {
    confirmation = {
      title: "Unsubscribe Workshop Item",
      message: `Unsubscribe ${workshopItemName(item)} from Steam Workshop for this game.`,
      detail: `${selectedGame?.name ?? "Selected game"} · ${item.published_file_id}`,
      confirmLabel: "Unsubscribe",
      danger: true,
      run: () => queueWorkshopAction(item, "unsubscribe")
    };
  }

  function upsertJob(job: Job) {
    if (job.type === "captured-install" && job.status === "canceled" && job.message === "Cleared") {
      jobs = jobs.filter((item) => item.id !== job.id);
      return;
    }
    jobs = [job, ...jobs.filter((item) => item.id !== job.id)];
    reconcileBusyState();
  }

  function markJobProcessing(job: Job, message: string) {
    upsertJob({
      ...job,
      status: "running",
      message,
      updated_at: new Date().toISOString()
    });
  }

  function reconcileBusyState() {
    const activeJobIDs = new Set(jobs.filter((job) => !["completed", "failed", "canceled"].includes(job.status)).map((job) => job.id));
    busyJobs = Object.fromEntries(Object.entries(busyJobs).filter(([jobID]) => activeJobIDs.has(jobID)));
    const activeCandidateIDs = new Set(installCandidates.map((candidate) => candidate.id));
    busyInstallCandidates = Object.fromEntries(Object.entries(busyInstallCandidates).filter(([candidateID]) => activeCandidateIDs.has(Number(candidateID))));
  }

  function setJobBusy(jobID: string, busy: boolean) {
    if (busy) {
      busyJobs = { ...busyJobs, [jobID]: true };
      return;
    }
    const next = { ...busyJobs };
    delete next[jobID];
    busyJobs = next;
  }

  function isJobBusy(job: Job) {
    return Boolean(busyJobs[job.id]);
  }

  function setInstallCandidateBusy(candidateID: number, busy: boolean) {
    if (busy) {
      busyInstallCandidates = { ...busyInstallCandidates, [candidateID]: true };
      return;
    }
    const next = { ...busyInstallCandidates };
    delete next[candidateID];
    busyInstallCandidates = next;
  }

  function isInstallCandidateBusy(candidate: InstallCandidate) {
    return Boolean(busyInstallCandidates[candidate.id]);
  }

  function canCancelJob(job: Job) {
    if (job.type === "steam-workshop-action" && job.status === "failed") return true;
    if (job.type === "captured-install" && job.status === "failed") return true;
    return !["completed", "failed", "canceled"].includes(job.status);
  }

  function getDeployableActions(plan: DeployPlan | null) {
    return plan?.actions.filter((action) => action.operation !== "keep" && action.operation !== "skip") ?? [];
  }

  function sortDrawerGames(items: Game[]) {
    return [...items].sort((a, b) => {
      const favoriteDelta = Number(favoriteGameIDs.has(b.app_id)) - Number(favoriteGameIDs.has(a.app_id));
      if (favoriteDelta !== 0) return favoriteDelta;
      if (gameSort === "az") return a.name.localeCompare(b.name);
      if (gameSort === "za") return b.name.localeCompare(a.name);
      const recentDelta = (gameRecent[b.app_id] ?? 0) - (gameRecent[a.app_id] ?? 0);
      if (recentDelta !== 0) return recentDelta;
      return a.name.localeCompare(b.name);
    });
  }

  function stateLabel(state: string) {
    return state === "clean_candidate" ? "Clean" : "Review";
  }

  function gameImage(appID: string) {
    return `https://cdn.cloudflare.steamstatic.com/steam/apps/${appID}/header.jpg`;
  }

  function settingsTitle(page: SettingsPage) {
    if (page === "jobs") return "Jobs";
    if (page === "install") return "Install";
    if (page === "sources") return "Sources";
    if (page === "nexus") return "Nexus";
    return "Settings";
  }

  function catalogStatusLabel(status: string) {
    if (status === "ready") return "Ready";
    if (status === "needs_credentials") return "Needs Key";
    if (status === "planned") return "Planned";
    if (status === "deferred") return "Deferred";
    return status.replace(/[_-]+/g, " ");
  }

  function catalogStatusClass(status: string) {
    const normalized = status.trim().toLowerCase().replace(/_/g, "-");
    if (normalized === "ready") return "catalog-status-ready";
    if (normalized === "needs-credentials") return "catalog-status-needs";
    if (normalized === "planned") return "catalog-status-planned";
    if (normalized === "deferred") return "catalog-status-deferred";
    return "catalog-status-unknown";
  }

  function catalogDetail(catalog: CatalogStatus) {
    if (catalog.capabilities?.length) {
      return catalog.capabilities.map(catalogCapabilityLabel).join(" · ");
    }
    const capabilities: string[] = [];
    if (catalog.url_import) capabilities.push("URL import");
    if (catalog.browse || catalog.search) capabilities.push("Browse/search");
    if (catalog.download) capabilities.push("Downloads");
    if (catalog.archive_upload) capabilities.push("Archive upload");
    if (catalog.installed_management) capabilities.push("Installed management");
    if (capabilities.length === 0) return "Not active in the current MVP build.";
    return capabilities.join(" · ");
  }

  function catalogCapabilityLabel(capability: string) {
    const normalized = capability.trim().toLowerCase().replace(/-/g, "_");
    if (normalized === "url_import") return "URL import";
    if (normalized === "browse_search") return "Browse/search";
    if (normalized === "download") return "Downloads";
    if (normalized === "archive_upload") return "Archive upload";
    if (normalized === "installed_management") return "Installed management";
    return capability.replace(/[_-]+/g, " ");
  }

  function actionMatchesGame(job: Job, game: Game) {
    if (jobMatchesGame(job, game)) return true;
    const haystack = `${job.title} ${job.message}`.toLowerCase().replace(/[^a-z0-9]/g, "");
    const gameName = game.name.toLowerCase().replace(/[^a-z0-9]/g, "");
    if (gameName && haystack.includes(gameName)) return true;
    return nexusDomainsForGame(game).some((domain) => haystack.includes(domain.toLowerCase().replace(/[^a-z0-9]/g, "")));
  }

  function gameForJob(job: Job) {
    const payloadAppID = job.payload?.app_id;
    if (payloadAppID) {
      const exact = games.find((game) => game.app_id === payloadAppID);
      if (exact) return exact;
    }
    return games.find((game) => actionMatchesGame(job, game)) ?? null;
  }

  function gameForInstallCandidate(candidate: InstallCandidate) {
    return games.find((game) => game.app_id === candidate.steam_app_id) ?? null;
  }

  function hasOpenInstallerChoiceJob(candidate: InstallCandidate) {
    const candidateID = String(candidate.id);
    return actionItems.some((job) => job.type === "installer-choice" && job.payload?.candidate_id === candidateID);
  }

  async function openInstallCandidate(candidate: InstallCandidate) {
    const game = gameForInstallCandidate(candidate);
    if (!game) {
      error = "DMM could not find the game for this installer item. Refresh games and try again.";
      return;
    }
    await selectGame(game);
    openGameModule("actions");
  }

  function mergeInstallCandidatesForGame(current: InstallCandidate[], appID: string, candidates: InstallCandidate[]) {
    return [
      ...candidates,
      ...current.filter((candidate) => candidate.steam_app_id !== appID)
    ].sort((a, b) => b.id - a.id);
  }

  function replaceInstallCandidate(candidate: InstallCandidate) {
    const replace = (items: InstallCandidate[]) => items.map((item) => (item.id === candidate.id ? candidate : item));
    installCandidates = replace(installCandidates);
    globalInstallCandidates = replace(globalInstallCandidates);
  }

  async function openActionItem(job: Job) {
    const game = gameForJob(job);
    if (!game) return;
    await selectGame(game);
    openGameModule("actions");
  }

  function jobMatchesGame(job: Job, game: Game) {
    if (job.payload?.app_id && job.payload.app_id === game.app_id) return true;
    const domain = job.payload?.game_domain?.toLowerCase();
    if (!domain) return false;
    return nexusDomainsForGame(game).includes(domain);
  }

  function nexusDomainsForGame(game: Game) {
    return (game.nexus_domains ?? []).map((domain) => domain.toLowerCase()).filter(Boolean);
  }

  function modStatusText(mod: InstalledMod) {
    if (mod.status === "needs_recovery") return "Needs repair";
    if (mod.status === "installed") return mod.enabled ? "Enabled" : "Installed";
    return mod.status;
  }

  function modProfileStateText(mod: InstalledMod) {
    if (mod.status === "needs_recovery") return "Needs repair before it can be used";
    return mod.enabled ? "Enabled in this profile" : "Installed, disabled in this profile";
  }

  function sourceKey(catalog: string | undefined) {
    const source = (catalog ?? "").trim().toLowerCase().replace(/_/g, "-");
    if (source === "github-releases") return "github";
    if (source === "steam-workshop" || source === "workshop") return "steam-workshop";
    if (source === "mod.io") return "modio";
    return source || "unknown";
  }

  function sourceForMod(mod: InstalledMod) {
    return mod.source_tag ?? mod.catalog;
  }

  function sourceForCandidate(candidate: { catalog?: string; source_tag?: string }) {
    return candidate.source_tag ?? candidate.catalog;
  }

  function hasSourceTag(catalog: string | undefined) {
    return sourceKey(catalog) !== "unknown";
  }

  function sourceOptionsForMods(mods: InstalledMod[], items: WorkshopItem[], workshop: SteamWorkshop | null) {
    const byKey = new Map<string, string>();
    for (const mod of mods) {
      addSourceOption(byKey, sourceForMod(mod));
    }
    if ((workshop?.detected || items.length > 0) && !byKey.has("steam-workshop")) {
      byKey.set("steam-workshop", sourceLabel("steam_workshop"));
    }
    return [...byKey.entries()].sort((a, b) => a[1].localeCompare(b[1]));
  }

  function sourceOptionsForActions(actions: Job[], candidates: InstallCandidate[]) {
    const byKey = new Map<string, string>();
    for (const action of actions) {
      addSourceOption(byKey, actionSource(action));
    }
    for (const candidate of candidates) {
      addSourceOption(byKey, sourceForCandidate(candidate));
    }
    return [...byKey.entries()].sort((a, b) => a[1].localeCompare(b[1]));
  }

  function sourceOptionsForJobs(items: Job[]) {
    const byKey = new Map<string, string>();
    for (const item of items) {
      addSourceOption(byKey, actionSource(item));
    }
    return [...byKey.entries()].sort((a, b) => a[1].localeCompare(b[1]));
  }

  function addSourceOption(byKey: Map<string, string>, source: string | undefined) {
    if (!hasSourceTag(source)) return;
    const key = sourceKey(source);
    if (!byKey.has(key)) byKey.set(key, sourceLabel(source));
  }

  function filterJobsBySource(items: Job[], sourceFilter: string) {
    if (sourceFilter === "all") return items;
    return items.filter((item) => sourceKey(actionSource(item)) === sourceFilter);
  }

  function filterCandidatesBySource(items: InstallCandidate[], sourceFilter: string) {
    if (sourceFilter === "all") return items;
    return items.filter((item) => sourceKey(sourceForCandidate(item)) === sourceFilter);
  }

  function sortModsForList(mods: InstalledMod[]) {
    return [...mods].sort((a, b) => {
      if (modListSort === "source") {
        const sourceDelta = sourceLabel(sourceForMod(a)).localeCompare(sourceLabel(sourceForMod(b)));
        if (sourceDelta !== 0) return sourceDelta;
      }
      if (modListSort === "az") return a.name.localeCompare(b.name) || a.priority - b.priority;
      if (modListSort === "enabled") {
        const enabledDelta = Number(b.enabled) - Number(a.enabled);
        if (enabledDelta !== 0) return enabledDelta;
      }
      return a.priority - b.priority || a.name.localeCompare(b.name);
    });
  }

  function modListSortLabel(sort: ModListSort) {
    if (sort === "source") return "Source";
    if (sort === "az") return "A-Z";
    if (sort === "enabled") return "Enabled";
    return "Profile";
  }

  function sourceLabel(catalog: string | undefined) {
    const source = (catalog ?? "").trim().toLowerCase();
    if (source === "nexus") return "Nexus";
    if (source === "steam_workshop" || source === "steam-workshop" || source === "workshop") return "Steam Workshop";
    if (source === "thunderstore") return "Thunderstore";
    if (source === "modrinth") return "Modrinth";
    if (source === "gamebanana") return "GameBanana";
    if (source === "modio" || source === "mod.io") return "mod.io";
    if (source === "curseforge") return "CurseForge";
    if (source === "moddb") return "ModDB";
    if (source === "github" || source === "github_releases") return "GitHub";
    if (source === "direct") return "Direct";
    if (source === "local") return "Local";
    if (source === "native") return "Native";
    return source ? source.replace(/[_-]+/g, " ") : "Unknown";
  }

  function sourceClass(catalog: string | undefined) {
    const source = (catalog ?? "").trim().toLowerCase().replace(/_/g, "-");
    if (source === "nexus") return "source-nexus";
    if (source === "steam-workshop" || source === "workshop") return "source-workshop";
    if (source === "thunderstore") return "source-thunderstore";
    if (source === "modrinth") return "source-modrinth";
    if (source === "gamebanana") return "source-gamebanana";
    if (source === "modio" || source === "mod.io") return "source-modio";
    if (source === "curseforge") return "source-curseforge";
    if (source === "moddb") return "source-moddb";
    if (source === "github" || source === "github-releases") return "source-github";
    if (source === "direct") return "source-direct";
    if (source === "local") return "source-local";
    if (source === "native") return "source-native";
    return "source-unknown";
  }

  function modUpdateLabel(update: ModUpdate | undefined) {
    if (!update) return "Not checked";
    if (update.status === "available") return "Update Available";
    if (update.status === "current") return "Current";
    if (update.status === "error") return "Check Failed";
    if (update.status === "unsupported") return "Not Supported";
    return "Unknown";
  }

  function modUpdateDetail(update: ModUpdate | undefined) {
    if (!update) return "Update status has not been checked yet.";
    if (update.message) return update.message;
    if (update.status === "available") return `Latest file ${update.latest_version || update.latest_file_id || "available"}`;
    if (update.status === "current") return "Installed file is current.";
    return "Review the mod page before updating.";
  }

  function deployActionDetail(action: DeployAction) {
    if (action.conflict) return action.conflict_reason || "Conflict";
    const base = `${action.operation || "add"} · ${action.strategy || "managed"}`;
    if (action.operation !== "skip" || !action.winner_installed_mod_id) return base;
    const loser = installedMods.find((mod) => mod.id === action.installed_mod_id)?.name || action.mod_id || "Lower priority mod";
    const winner = installedMods.find((mod) => mod.id === action.winner_installed_mod_id)?.name || action.winner_mod_id || "Higher priority mod";
    return `${base} · ${loser} loses to ${winner}`;
  }

  function getConflictChoiceTargets(plan: DeployPlan | null): ConflictChoiceTarget[] {
    if (!plan) return [];
    const groups = new Map<string, ConflictChoiceTarget>();
    for (const action of plan.actions) {
      if (action.operation !== "skip" || !action.target_path || !action.winner_installed_mod_id) continue;
      let group = groups.get(action.target_path);
      if (!group) {
        const winner = installedMods.find((mod) => mod.id === action.winner_installed_mod_id);
        group = {
          target_path: action.target_path,
          target_relative: action.target_relative,
          current_winner_id: action.winner_installed_mod_id,
          current_winner_name: winner?.name || action.winner_mod_id || "Selected mod",
          reason: action.conflict_reason || "Resolved by profile order",
          candidates: []
        };
        groups.set(action.target_path, group);
      }
      addConflictCandidate(group, action.installed_mod_id, action.priority, false);
      addConflictCandidate(group, action.winner_installed_mod_id, action.winner_priority, true);
    }
    return Array.from(groups.values()).map((group) => ({
      ...group,
      candidates: group.candidates.sort((a, b) => Number(b.current) - Number(a.current) || (a.priority ?? 0) - (b.priority ?? 0) || a.name.localeCompare(b.name))
    }));
  }

  function addConflictCandidate(group: ConflictChoiceTarget, installedModID: number | undefined, priority: number | undefined, current: boolean) {
    if (!installedModID) return;
    const existing = group.candidates.find((candidate) => candidate.id === installedModID);
    if (existing) {
      existing.current = existing.current || current;
      if (existing.priority === undefined) existing.priority = priority;
      return;
    }
    const mod = installedMods.find((item) => item.id === installedModID);
    group.candidates.push({
      id: installedModID,
      name: mod?.name || `Mod ${installedModID}`,
      catalog: mod?.catalog,
      priority,
      current
    });
  }

  function primaryModMetadata(mod: InstalledMod) {
    return mod.metadata?.find((metadata) => metadata.unique_id || metadata.name) ?? null;
  }

  function modDependencyLabels(mod: InstalledMod) {
    const labels: string[] = [];
    for (const metadata of mod.metadata ?? []) {
      if (metadata.content_pack_for?.unique_id) {
        labels.push(`Requires ${metadata.content_pack_for.unique_id}${metadata.content_pack_for.minimum_version ? ` ${metadata.content_pack_for.minimum_version}+` : ""}`);
      }
      for (const dependency of metadata.dependencies ?? []) {
        if (!dependency.required || !dependency.unique_id) continue;
        labels.push(`Requires ${dependency.unique_id}${dependency.minimum_version ? ` ${dependency.minimum_version}+` : ""}`);
      }
    }
    return Array.from(new Set(labels));
  }

  function actionNextStep(action: Job) {
    if (action.type === "steam-workshop-action") {
      if (action.status === "waiting" || action.status === "queued") return "Waiting for Decky to apply this Steam Workshop change through Steam.";
      if (action.status === "running") return "Decky is applying this Steam Workshop change through Steam.";
      if (action.status === "failed") return "Decky could not apply this Steam Workshop change. Make sure the Deck is online with Steam running, then retry or cancel.";
      return "This Workshop action is retained in job history for diagnostics.";
    }
    if (action.type === "installer-choice") {
      return "Choose installer options to finish adding this mod to the selected profile.";
    }
    if (action.status === "waiting") {
      return "Add this downloaded mod to the selected profile, or cancel it and keep the archive cache untouched.";
    }
    if (action.status === "running" || action.status === "queued") {
      return "DMM is downloading or installing this mod. It will appear in the profile when ready.";
    }
    if (action.status === "failed") {
      return "The mod was not added. Retry from the cached download when available, or clear it if this action is no longer needed.";
    }
    return "This action is retained in job history for diagnostics.";
  }

  function actionStatusLabel(action: Job) {
    if (action.type === "steam-workshop-action" && (action.status === "waiting" || action.status === "queued")) return "Waiting for Decky";
    if (action.type === "installer-choice" && action.status === "waiting") return "Needs choices";
    if (action.status === "waiting") return "Ready to install";
    if (action.status === "running") return "Processing";
    if (action.status === "queued") return "Queued";
    if (action.status === "failed") return "Failed";
    return action.status;
  }

  function actionSource(action: Job) {
    if (action.source_tag) return action.source_tag;
    if (action.catalog) return action.catalog;
    if (action.type === "steam-workshop-action") return "steam_workshop";
    return action.payload?.catalog;
  }

  function installerPresetScopeLabel(preset: InstallerChoicePreset) {
    if (preset.reuse_scope === "exact_file") return "Exact file only";
    return "Manual review";
  }

  function candidateStatusLabel(candidate: InstallCandidate) {
    if (candidate.status === "needs_choices") return "Needs choices";
    if (candidate.status === "blocked") return "Blocked";
    return candidate.status;
  }

  function displayValidationWarnings(diagnostics: GameDiagnostics | null) {
    const warnings = diagnostics?.validation_warnings ?? [];
    const requirements = diagnostics?.runtime_requirements ?? [];
    if (warnings.length === 0 || requirements.length === 0) return warnings;
    return warnings.filter((warning) => {
      const normalized = warning.toLowerCase();
      return !requirements.some((requirement) => {
        const name = requirement.name.toLowerCase();
        const kind = requirement.kind.toLowerCase();
        const readableKind = kind.replace(/-/g, " ");
        return name !== "" && (
          normalized.includes(`${name} runtime requirement`) ||
          (kind !== "" && normalized.includes(`${name} ${kind} requirement`)) ||
          (readableKind !== "" && normalized.includes(`${name} ${readableKind} requirement`))
        );
      });
    });
  }

  async function confirmCurrentAction() {
    if (!confirmation) return;
    const action = confirmation;
    confirmation = null;
    await action.run();
  }

  onMount(() => {
    refresh();
    connectEvents();
    const refreshOnFocus = () => {
      scheduleActionStateRefresh(true, deployPlan !== null);
    };
    const refreshOnVisibility = () => {
      if (!document.hidden) scheduleActionStateRefresh(true, deployPlan !== null);
    };
    window.addEventListener("focus", refreshOnFocus);
    document.addEventListener("visibilitychange", refreshOnVisibility);
    return () => {
      closeEventSocket();
      window.removeEventListener("focus", refreshOnFocus);
      document.removeEventListener("visibilitychange", refreshOnVisibility);
      if (selectedGameRefreshTimer !== null) window.clearTimeout(selectedGameRefreshTimer);
      if (actionStateRefreshTimer !== null) window.clearTimeout(actionStateRefreshTimer);
    };
  });
</script>

<main class="app-shell">
  <header class="app-header">
    <button type="button" class="icon-button" aria-label="Open games" on:click={() => (drawer = "games")}>☰</button>
    <div class="title-block">
      {#if surface === "game" && selectedGame}
        <img src={gameImage(selectedGame.app_id)} alt="" />
      {/if}
      <div>
        <p class="eyebrow">Decky Mod Manager</p>
        <h1>{title}</h1>
      </div>
    </div>
    <button type="button" class="icon-button" aria-label="Open settings" on:click={() => (drawer = "settings")}>⚙</button>
  </header>

  {#if drawer}
    <button type="button" class="scrim" aria-label="Close menu" on:click={() => (drawer = null)}></button>
    <aside class="drawer">
      {#if drawer === "games"}
        <div class="drawer-heading">
          <h2>Games</h2>
          <button type="button" class="icon-button small" aria-label="Refresh games" on:click={refresh}>R</button>
        </div>
        <input bind:value={gameQuery} aria-label="Search games" placeholder="Search games" />
        <div class="game-drawer-controls">
          <label>
            <span>Sort</span>
            <select aria-label="Sort games" value={gameSort} on:change={(event) => setGameSort(event.currentTarget.value as GameSort)}>
              <option value="recent">Recent</option>
              <option value="az">A-Z</option>
              <option value="za">Z-A</option>
            </select>
          </label>
          <span>{favoriteGameIDs.size} pinned</span>
        </div>
        <div class="drawer-list game-list">
          {#each filteredGames as game}
            <div
              class="game-row"
              class:selected={selectedGame?.app_id === game.app_id}
              class:needs-review={game.state !== "clean_candidate"}
            >
              <button type="button" class="game-select" on:click={() => selectGame(game)}>
                <img src={gameImage(game.app_id)} alt="" loading="lazy" />
                <span>
                  <strong>{game.name}</strong>
                  <small>{game.app_id} · {stateLabel(game.state)}</small>
                </span>
              </button>
              <button
                type="button"
                class="favorite-game"
                class:favorited={isFavoriteGame(game.app_id)}
                aria-label={isFavoriteGame(game.app_id) ? `Unfavorite ${game.name}` : `Favorite ${game.name}`}
                on:click={() => toggleFavoriteGame(game.app_id)}
              >
                {isFavoriteGame(game.app_id) ? "★" : "☆"}
              </button>
            </div>
          {/each}
        </div>
      {:else}
        <div class="drawer-heading">
          <h2>Settings</h2>
          <button type="button" class="icon-button small" aria-label="Close settings" on:click={() => (drawer = null)}>×</button>
        </div>
        <nav class="settings-nav" aria-label="Settings">
          <button type="button" class:active={activeSettingsPage === "overview"} on:click={() => openSettings("overview")}>Overview</button>
          <button type="button" on:click={openActionCenter}>Action Center</button>
          <button type="button" class:active={activeSettingsPage === "jobs"} on:click={() => openSettings("jobs")}>Jobs</button>
          <button type="button" class:active={activeSettingsPage === "install"} on:click={() => openSettings("install")}>Install Behavior</button>
          <button type="button" class:active={activeSettingsPage === "sources"} on:click={() => openSettings("sources")}>Sources</button>
          <button type="button" class:active={activeSettingsPage === "nexus"} on:click={() => openSettings("nexus")}>Nexus</button>
        </nav>
      {/if}
    </aside>
  {/if}

  <section class="status-strip" aria-label="App status">
    <span>{status?.nexus.api_key_configured ? "Nexus ready" : "Nexus missing"}</span>
    <span>{cleanCount} clean</span>
    <span>{reviewCount} review</span>
  </section>

  {#if error}
    <section class="alert">{error}</section>
  {/if}

  {#if loading}
    <section class="empty-state">Loading...</section>
  {:else if surface === "actions"}
    <section class="settings-screen">
      <article class="workspace-panel">
        <div class="panel-heading">
          <h2>Action Center</h2>
          <span>{globalActionCount} open</span>
        </div>
        {#if actionSourceOptions.length > 1}
          <div class="mod-list-controls compact-controls">
            <label>
              <span>Source</span>
              <select bind:value={actionSourceFilter}>
                <option value="all">All Sources</option>
                {#each actionSourceOptions as [key, label]}
                  <option value={key}>{label}</option>
                {/each}
              </select>
            </label>
          </div>
        {/if}
        {#if capturedInstallActions.length > 0}
          <button type="button" class="secondary-action" on:click={clearCapturedInstallActions}>Clear Install Actions</button>
        {/if}
        {#if globalActionCount === 0}
          <div class="action-home">
            <div class="empty-state inline-empty">
              <h2>No Actions Needed</h2>
              <p class="hint">Open a game to paste a mod URL, or capture an nxm:// link from the Deck browser flow.</p>
            </div>
            <button type="button" on:click={() => (drawer = "games")}>Choose Game</button>
          </div>
        {/if}
        {#if globalActionCount > 0 && visibleActionItems.length === 0 && visibleActionCenterCandidates.length === 0}
          <p class="hint">No open actions match the selected source.</p>
        {/if}
        {#if visibleActionItems.length > 0}
          <div class="action-list">
            {#each visibleActionItems as action}
              <article class:failed-action={action.status === "failed"}>
                <div>
                  <div class="mod-title-line">
                    <strong>{action.title}</strong>
                    <span class={`source-pill ${sourceClass(actionSource(action))}`}>{sourceLabel(actionSource(action))}</span>
                  </div>
	                    {#if action.message}<p>{action.message}</p>{/if}
	                    <p class="action-next-step">{actionNextStep(action)}</p>
	                    <small>{new Date(action.updated_at).toLocaleString()}</small>
	                  </div>
	                  <div class="action-controls">
	                    <span>{actionStatusLabel(action)}</span>
	                    {#if action.type === "installer-choice"}
	                      <button type="button" on:click={() => openActionItem(action)}>{gameForJob(action) ? "Open Choices" : "Choose Game"}</button>
	                    {/if}
	                    {#if action.status === "waiting"}
	                      {#if action.type === "captured-install"}
	                        <button type="button" on:click={() => installCapturedMod(action)} disabled={isJobBusy(action)}>{isJobBusy(action) ? "Working..." : "Install"}</button>
	                      {/if}
	                    {/if}
	                    {#if action.type === "captured-install" && action.status === "failed"}
	                      <button type="button" on:click={() => retryCapturedInstallAction(action)} disabled={isJobBusy(action)}>{isJobBusy(action) ? "Working..." : "Retry"}</button>
	                    {/if}
	                    {#if action.type === "steam-workshop-action" && action.status === "failed"}
	                      <button type="button" on:click={() => retryWorkshopAction(action)} disabled={isJobBusy(action)}>{isJobBusy(action) ? "Working..." : "Retry"}</button>
	                    {/if}
	                    {#if canCancelJob(action)}
	                      <button type="button" class="secondary-action compact" on:click={() => cancelJob(action)} disabled={isJobBusy(action)}>Cancel</button>
	                    {/if}
	                  </div>
	                </article>
            {/each}
          </div>
        {/if}
        {#if visibleActionCenterCandidates.length > 0}
          <section class="blocked-candidates" aria-label="Installer items">
            <div class="panel-heading compact-heading">
              <h3>Installer Items</h3>
              <span>{visibleActionCenterCandidates.length}</span>
            </div>
            <p class="hint">These downloaded archives need installer choices or review before they can be added to a profile.</p>
            <div class="action-list">
              {#each visibleActionCenterCandidates as candidate}
                {@const candidateGame = gameForInstallCandidate(candidate)}
                <article class:failed-action={candidate.status === "blocked"}>
                  <div>
                    <div class="mod-title-line">
                      <strong>{candidate.name}</strong>
                      <span class={`source-pill ${sourceClass(sourceForCandidate(candidate))}`}>{sourceLabel(sourceForCandidate(candidate))}</span>
                    </div>
                    <p>{candidate.reason}</p>
                    <small>{candidateGame?.name ?? `App ${candidate.steam_app_id}`} · {candidate.source_game_domain}/mods/{candidate.source_mod_id}/files/{candidate.source_file_id}</small>
                  </div>
                  <div class="action-controls">
                    <span>{candidateStatusLabel(candidate)}</span>
                    <button type="button" on:click={() => openInstallCandidate(candidate)}>{candidate.status === "needs_choices" ? "Open Choices" : "Review"}</button>
                  </div>
                </article>
              {/each}
            </div>
          </section>
        {/if}
      </article>
    </section>
  {:else if surface === "settings"}
    <section class="settings-screen">
      {#if activeSettingsPage === "overview"}
        <article class="workspace-panel">
          <h2>Overview</h2>
          <dl class="settings-list">
            <div><dt>Games</dt><dd>{status?.game_count ?? games.length}</dd></div>
            <div><dt>Clean</dt><dd>{cleanCount}</dd></div>
            <div><dt>Review</dt><dd>{reviewCount}</dd></div>
            <div><dt>Nexus</dt><dd>{status?.nexus.api_key_configured ? "Configured" : "Missing API key"}</dd></div>
            <div><dt>Captured installs</dt><dd>{status?.install.auto_install_captured_downloads ? "Install automatically" : "Manual install"}</dd></div>
            <div><dt>Auto enable</dt><dd>{status?.install.auto_enable_installed_mods ? "Enabled" : "Disabled"}</dd></div>
            <div><dt>Downloads</dt><dd>{status?.download?.active_captured_downloads ?? 0}/{status?.download?.max_concurrent_captured_downloads ?? 2} active</dd></div>
            <div><dt>Sources</dt><dd>{readySourceCatalogCount}/{sourceCatalogCount} ready</dd></div>
          </dl>
        </article>
      {:else if activeSettingsPage === "jobs"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Jobs</h2>
            <span>{jobs.length}</span>
          </div>
          {#if jobSourceOptions.length > 1}
            <div class="mod-list-controls compact-controls">
              <label>
                <span>Source</span>
                <select bind:value={jobSourceFilter}>
                  <option value="all">All Sources</option>
                  {#each jobSourceOptions as [key, label]}
                    <option value={key}>{label}</option>
                  {/each}
                </select>
              </label>
            </div>
          {/if}
          {#if jobs.length === 0}
            <p class="hint">No jobs yet.</p>
          {:else if visibleJobs.length === 0}
            <p class="hint">No jobs match the selected source.</p>
          {:else}
            <div class="jobs">
              {#each visibleJobs as job}
                <article class="job">
	                  <div>
	                    <div class="mod-title-line">
	                      <strong>{job.title}</strong>
	                      {#if hasSourceTag(actionSource(job))}
	                        <span class={`source-pill ${sourceClass(actionSource(job))}`}>{sourceLabel(actionSource(job))}</span>
	                      {/if}
	                    </div>
	                    {#if job.message}<p>{job.message}</p>{/if}
	                  </div>
	                  <div class="job-actions">
	                    <span>{job.status}</span>
	                    {#if canCancelJob(job)}
	                      <button type="button" class="secondary-action compact" on:click={() => cancelJob(job)} disabled={isJobBusy(job)}>Cancel</button>
	                    {/if}
	                  </div>
	                </article>
              {/each}
            </div>
          {/if}
        </article>
      {:else if activeSettingsPage === "install"}
        <article class="workspace-panel">
          <h2>Install Behavior</h2>
          <dl class="settings-list">
            <div><dt>Captured installs</dt><dd>{status?.install.auto_install_captured_downloads ? "Install automatically" : "Manual install"}</dd></div>
            <div><dt>New mod state</dt><dd>{status?.install.auto_enable_installed_mods ? "Enable automatically" : "Install disabled"}</dd></div>
            <div><dt>Downloads</dt><dd>{status?.download?.active_captured_downloads ?? 0}/{status?.download?.max_concurrent_captured_downloads ?? 2} active</dd></div>
            <div><dt>Per game</dt><dd>{status?.download?.max_concurrent_captured_downloads_per_game ?? 1} active download{(status?.download?.max_concurrent_captured_downloads_per_game ?? 1) === 1 ? "" : "s"}</dd></div>
          </dl>
          <label class="settings-control">
            <span>Concurrent downloads</span>
            <select value={status?.download?.max_concurrent_captured_downloads ?? 2} on:change={(event) => updateDownloadConcurrency(Number(event.currentTarget.value))}>
              <option value="1">1 download</option>
              <option value="2">2 downloads</option>
              <option value="3">3 downloads</option>
              <option value="4">4 downloads</option>
            </select>
          </label>
          <label class="settings-control">
            <span>Per-game downloads</span>
            <select value={status?.download?.max_concurrent_captured_downloads_per_game ?? 1} on:change={(event) => updatePerGameDownloadConcurrency(Number(event.currentTarget.value))}>
              <option value="1">1 per game</option>
              <option value="2" disabled={(status?.download?.max_concurrent_captured_downloads ?? 2) < 2}>2 per game</option>
              <option value="3" disabled={(status?.download?.max_concurrent_captured_downloads ?? 2) < 3}>3 per game</option>
              <option value="4" disabled={(status?.download?.max_concurrent_captured_downloads ?? 2) < 4}>4 per game</option>
            </select>
          </label>
          <p class="hint">These Deck behavior switches are managed from the Decky sidebar settings.</p>
        </article>
      {:else if activeSettingsPage === "sources"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Sources</h2>
            <span>{readyCatalogCount} ready</span>
          </div>
          <div class="catalog-key-grid">
            <form class="provider-key-form" on:submit|preventDefault={() => updateCatalogCredential("modio", modIOAPIKey)}>
              <label>
                <span>mod.io API key</span>
                <input type="password" bind:value={modIOAPIKey} autocomplete="off" placeholder="Paste key to enable mod.io imports" />
              </label>
              <button type="submit" disabled={catalogSettingsBusy === "modio" || modIOAPIKey.trim() === ""}>{catalogSettingsBusy === "modio" ? "Saving..." : "Save"}</button>
            </form>
            <form class="provider-key-form" on:submit|preventDefault={() => updateCatalogCredential("curseforge", curseForgeAPIKey)}>
              <label>
                <span>CurseForge API key</span>
                <input type="password" bind:value={curseForgeAPIKey} autocomplete="off" placeholder="Paste key to enable CurseForge imports" />
              </label>
              <button type="submit" disabled={catalogSettingsBusy === "curseforge" || curseForgeAPIKey.trim() === ""}>{catalogSettingsBusy === "curseforge" ? "Saving..." : "Save"}</button>
            </form>
          </div>
          {#if catalogSettingsMessage}<p class="hint">{catalogSettingsMessage}</p>{/if}
          <div class="catalog-list">
            {#each catalogs as catalog}
              <article>
                <div class="catalog-title">
                  <div>
                    <strong>{catalog.name}</strong>
                    <p>{catalogDetail(catalog)}</p>
                  </div>
                  <span class={`catalog-status ${catalogStatusClass(catalog.status)}`}>{catalogStatusLabel(catalog.status)}</span>
                </div>
                <div class="catalog-meta">
                  <span class={`source-pill ${sourceClass(catalog.source_tag)}`}>{sourceLabel(catalog.source_tag)}</span>
                  <span>{catalog.kind}</span>
                  {#if catalog.credentials_required}
                    <span>{catalog.configured ? "Credentials configured" : "Credentials needed"}</span>
                  {/if}
                </div>
                {#if catalog.notes?.length}
                  <ul class="provider-notes">
                    {#each catalog.notes as note}
                      <li>{note}</li>
                    {/each}
                  </ul>
                {/if}
              </article>
            {/each}
          </div>
        </article>
      {:else}
        <article class="workspace-panel">
          <h2>Nexus</h2>
          <dl class="settings-list">
            <div><dt>API key</dt><dd>{status?.nexus.api_key_configured ? "Configured" : "Missing"}</dd></div>
          </dl>
        </article>
      {/if}
    </section>
  {:else if selectedGame}
    <section class="game-workspace">
      <article class="game-hero">
        <img src={gameImage(selectedGame.app_id)} alt="" />
        <div>
          <h2>{selectedGame.name}</h2>
          <span class:review-badge={selectedGame.state !== "clean_candidate"}>{stateLabel(selectedGame.state)}</span>
        </div>
      </article>

      <nav class="module-tabs" aria-label="Game modules">
        <button type="button" class:active={activeGameModule === "plugins"} on:click={() => openGameModule("plugins")}>Mods</button>
        <button type="button" class:active={activeGameModule === "actions"} on:click={() => openGameModule("actions")}>Actions</button>
        <button type="button" class:active={activeGameModule === "profiles"} on:click={() => openGameModule("profiles")}>Profiles</button>
        <button type="button" class:active={activeGameModule === "review"} on:click={() => openGameModule("review")}>Review</button>
        <button type="button" class:active={activeGameModule === "paths"} on:click={() => openGameModule("paths")}>Paths</button>
      </nav>

      {#if activeGameModule === "plugins"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>{selectedProfile?.name ?? "Default"} Profile</h2>
            <span>{enabledMods.length} enabled · {disabledMods.length} disabled</span>
          </div>
          {#if selectedGameActivity.length > 0}
            <section class="activity-strip" aria-label="Game activity">
              {#each selectedGameActivity.slice(0, 3) as job}
                <article class:failed-action={job.status === "failed"}>
                  <div class="mod-title-line">
                    <strong>{job.title}</strong>
                    {#if hasSourceTag(actionSource(job))}
                      <span class={`source-pill ${sourceClass(actionSource(job))}`}>{sourceLabel(actionSource(job))}</span>
                    {/if}
                  </div>
                  <span>{job.status}</span>
                  {#if job.message}<small>{job.message}</small>{/if}
                </article>
              {/each}
            </section>
          {/if}
          {#if deploymentStatus?.restore_available}
            <section class="profile-recovery-banner" aria-label="Restore last applied state">
              <div>
                <strong>Restore available</strong>
                <p>{deploymentStatus.restore_summary ?? "Return this game to the last DMM-applied state."}</p>
              </div>
              <button type="button" on:click={askRestoreDeployment}>Restore</button>
            </section>
          {/if}
          <section class="management-grid">
            <div class="management-card profile-card">
              <div class="card-heading">
                <h3>Selected Profile</h3>
                <span>{selectedProfile?.name ?? "Default"}</span>
              </div>
              <div class="profile-summary">
                <div><strong>{enabledMods.length}</strong><span>On</span></div>
                <div><strong>{disabledMods.length}</strong><span>Off</span></div>
                <button type="button" class="summary-action" on:click={() => openGameModule("actions")} aria-label={`Open Action Center for this game; ${selectedGameActionCount} open`}>
                  <strong>{selectedGameActionCount}</strong>
                  <span>{selectedGameActionCount === 1 ? "Action" : "Actions"}</span>
                  <em>Open</em>
                </button>
              </div>
              {#if hasDeployConflicts}
                <p class="deploy-message danger">This profile has conflicts that need review before it can be applied.</p>
              {:else if hasPendingProfileChanges}
                <p class="deploy-message">Enabled-mod changes are ready. Toggling a mod applies them automatically; Advanced Profile Tools can apply them now.</p>
              {:else if deploymentStatus?.deployed}
                <p class="deploy-message success">Enabled mods are applied to the game.</p>
              {:else if enabledMods.length === 0}
                <p class="deploy-message">No enabled mods are applied for this profile.</p>
              {:else}
                <p class="deploy-message">Enable or disable a mod to apply enabled mods.</p>
              {/if}
            </div>

            <div class="management-card import-card">
              <div class="card-heading">
                <h3>Add Mod</h3>
                <span>{selectedGameActionItems.length} open</span>
              </div>
              <form class="stacked-form" on:submit|preventDefault={resolveCapturedInstall}>
                <textarea bind:value={captureURL} rows="4" aria-label="Mod URL" placeholder="Nexus, nxm://, Thunderstore, GitHub, Modrinth, GameBanana, mod.io, CurseForge, or direct archive URL"></textarea>
                <button type="submit">Add URL</button>
              </form>
              <form class="stacked-form local-archive-form" on:submit|preventDefault={uploadLocalArchive}>
                <label class="local-archive-picker">
                  <span>Local Archive</span>
                  <input bind:this={localArchiveInput} type="file" accept=".zip,.7z,.rar" on:change={handleLocalArchiveChange} />
                </label>
                {#if localArchiveFile}
                  <p class="hint">{localArchiveFile.name} · {formatBytes(localArchiveFile.size)}</p>
                {/if}
                <button type="submit" class="secondary-action" disabled={!localArchiveFile || localArchiveBusy}>{localArchiveBusy ? "Uploading..." : "Upload Archive"}</button>
              </form>
              {#if localArchiveMessage}
                <p class="hint success-copy">{localArchiveMessage}</p>
              {/if}
              <p class="hint">Mods that need choices or review will appear in Action Center. Installed mods remain disabled until you enable them in the selected profile.</p>
              {#if selectedNexusDomain()}
                <details class="nexus-browser">
                  <summary>
                    <span>Browse Nexus</span>
                    <small>{selectedNexusDomain()}</small>
                  </summary>
                  <form class="nexus-search-form" on:submit|preventDefault={() => searchNexusMods()}>
                    <input bind:value={nexusSearchQuery} aria-label="Search Nexus mods" placeholder="Search Nexus mods" />
                    <button type="button" class="secondary-action compact" on:click={cycleNexusSort}>{nexusSortLabel(nexusSearchSort)}</button>
                    <button type="button" class="secondary-action compact" on:click={cycleNexusTimeWindow}>{nexusTimeWindowLabel(nexusSearchTimeWindow)}</button>
                    <button type="button" class="secondary-action compact" on:click={toggleNexusCompatibilityFilter}>{nexusSearchVortexOnly ? "Vortex Only" : "All Mods"}</button>
                    <button type="submit" disabled={nexusSearchBusy}>{nexusSearchBusy ? "Searching..." : "Search"}</button>
                  </form>
                  {#if nexusSearchResults.length > 0}
                    <p class="hint">Showing {nexusSearchResults.length} of {compactNumber(nexusSearchTotal)} {nexusSearchVortexOnly ? "Vortex-compatible" : "Nexus"} results.</p>
                    <div class="nexus-results">
                      {#each nexusSearchResults as mod}
                        {@const files = nexusFilesByMod[mod.mod_id] ?? []}
                        {@const filesOpen = selectedNexusModID === mod.mod_id}
                        <article class:open={filesOpen}>
                          <button type="button" class="nexus-result-main" on:click={() => loadNexusModFiles(mod)}>
                            <span>
                              <span class="mod-title-line">
                                <strong>{mod.name}</strong>
                                <span class={`source-pill ${sourceClass("nexus")}`}>{sourceLabel("nexus")}</span>
                              </span>
                              {#if mod.summary}<small>{mod.summary}</small>{/if}
                              <em>{compactNumber(mod.downloads)} downloads · {compactNumber(mod.endorsements)} endorsements</em>
                            </span>
                            <b>{busyNexusFileKey === `files:${mod.mod_id}` ? "Loading" : filesOpen ? "Hide" : "Files"}</b>
                          </button>
                          {#if filesOpen && files.length > 0}
                            <div class="nexus-file-list">
                              {#each files as file}
                                <article class="nexus-file-card">
                                  <span>
                                    <span class="mod-title-line">
                                      <strong>{file.name || file.file_name}</strong>
                                      <span class={`source-pill ${sourceClass("nexus")}`}>{sourceLabel("nexus")}</span>
                                    </span>
                                    <small>{file.file_name || "Nexus file"} · {formatBytes(file.size)} · v{file.version || "unknown"}</small>
                                  </span>
                                  <div class="nexus-file-actions">
                                    <button type="button" class="secondary-action compact" on:click={() => openNexusFilePage(mod.mod_id, file.file_id)}>Open Page</button>
                                    <button type="button" on:click={() => addNexusSearchFile(mod, file)} disabled={busyNexusFileKey === `${mod.mod_id}:${file.file_id}`}>
                                      <em>{busyNexusFileKey === `${mod.mod_id}:${file.file_id}` ? "Adding" : "Add"}</em>
                                    </button>
                                  </div>
                                </article>
                              {/each}
                            </div>
                          {/if}
                        </article>
                      {/each}
                    </div>
                  {/if}
                  {#if nexusSearchError}
                    <p class="hint warning-copy">{nexusSearchError}</p>
                  {/if}
                </details>
              {:else}
                <p class="hint">This game does not have an extension-owned Nexus domain yet.</p>
              {/if}
              {#if resolvedCapture}
                <p class="hint">Resolved {resolvedCapture}</p>
              {/if}
              {#if nexusFiles.length > 0}
                <div class="file-list">
                  {#each nexusFiles as file}
                    <button type="button" on:click={() => resolveFile(file)}>
                      <span>
                        <strong>{file.name || file.file_name}</strong>
                        <small>{file.file_name} · v{file.version || "unknown"}</small>
                      </span>
                      <em>Use</em>
                    </button>
                  {/each}
                </div>
              {/if}
            </div>
          </section>

          {#if installedMods.length === 0 && !selectedWorkshop}
            <p class="hint">No profile mods yet. Capture or paste a mod link to add a supported mod to this profile.</p>
          {:else}
            <section class="mod-section">
              <div class="card-heading">
                <h3>Mods</h3>
                <div class="card-heading-actions">
                  <span>{selectedProfile?.name ?? "Default"}</span>
                  <button type="button" class="secondary-action compact" on:click={checkModUpdates} disabled={modUpdateBusy || installedMods.length === 0}>{modUpdateBusy ? "Checking..." : "Check Updates"}</button>
                </div>
              </div>
              {#if modUpdateMessage}<p class="hint">{modUpdateMessage}</p>{/if}
              {#if modUpdateBrowserURL}
                <button type="button" class="secondary-action compact" on:click={() => window.open(modUpdateBrowserURL, "_blank", "noopener")}>Open Provider File Page</button>
              {/if}
              <div class="mod-list-controls">
                <label>
                  <span>Source</span>
                  <select bind:value={modSourceFilter}>
                    <option value="all">All Sources</option>
                    {#each modSourceOptions as [key, label]}
                      <option value={key}>{label}</option>
                    {/each}
                  </select>
                </label>
                <label>
                  <span>Sort</span>
                  <select bind:value={modListSort}>
                    <option value="profile">{modListSortLabel("profile")}</option>
                    <option value="source">{modListSortLabel("source")}</option>
                    <option value="az">{modListSortLabel("az")}</option>
                    <option value="enabled">{modListSortLabel("enabled")}</option>
                  </select>
                </label>
              </div>
              {#if !hasVisibleModRows}
                <p class="hint">No mods match the selected source.</p>
              {/if}
              {#if visibleInstalledMods.length > 0}
                <div class="mod-list">
                {#each visibleInstalledMods as mod}
                  {@const metadata = primaryModMetadata(mod)}
                  {@const dependencyLabels = modDependencyLabels(mod)}
                  {@const orderIndex = installedMods.findIndex((item) => item.id === mod.id)}
                  <article>
                    <div>
                      <div class="mod-title-line">
                        <strong>{mod.name}</strong>
                        <span class={`source-pill ${sourceClass(sourceForMod(mod))}`}>{sourceLabel(sourceForMod(mod))}</span>
                      </div>
                      {#if metadata || mod.mod_type}
                        <div class="mod-meta">
                          {#if metadata?.unique_id}<span>{metadata.unique_id}</span>{/if}
                          {#if metadata?.version}<span>v{metadata.version}</span>{/if}
                          {#if mod.mod_type}<span>{mod.mod_type}</span>{/if}
                        </div>
                      {/if}
                      {#if dependencyLabels.length}
                        <div class="mod-requirements">
                          {#each dependencyLabels.slice(0, 3) as dependency}
                            <span>{dependency}</span>
                          {/each}
                          {#if dependencyLabels.length > 3}<span>{dependencyLabels.length - 3} more</span>{/if}
                        </div>
                      {/if}
                      <div class:available-update={mod.update?.status === "available"} class:failed-update={mod.update?.status === "error"} class="mod-update-status">
                        <span>{modUpdateLabel(mod.update)}</span>
                        <small>{modUpdateDetail(mod.update)}</small>
                      </div>
                      <small>{modProfileStateText(mod)}</small>
                    </div>
                    <div class="mod-actions">
                      <span class:warning-status={mod.status === "needs_recovery"}>{busyMods[mod.id] ? "Working" : modStatusText(mod)}</span>
                      <label class="mod-toggle">
                        <input type="checkbox" checked={mod.enabled} disabled={Boolean(busyMods[mod.id])} on:change={(event) => setModEnabled(mod, event.currentTarget.checked)} />
                        <em>{busyMods[mod.id] === "toggle" ? "Saving" : mod.enabled ? "On" : "Off"}</em>
                      </label>
                      {#if mod.update?.status === "available"}
                        <button type="button" class="secondary-action compact" disabled={Boolean(busyMods[mod.id])} on:click={() => updateInstalledMod(mod)}>
                          {busyMods[mod.id] === "update" ? "Queueing..." : "Install Update"}
                        </button>
                      {/if}
                      <button type="button" class="secondary-action compact" disabled={Boolean(busyMods[mod.id])} on:click={() => reinstallInstalledMod(mod)}>
                        {busyMods[mod.id] === "reinstall" ? "Reinstalling..." : "Reinstall"}
                      </button>
                      <button type="button" class="secondary-action compact danger-action" disabled={Boolean(busyMods[mod.id])} on:click={() => askRemoveInstalledMod(mod)}>
                        {busyMods[mod.id] === "remove" ? "Removing..." : "Remove"}
                      </button>
                    </div>
                    <details class="mod-advanced">
                      <summary>Advanced</summary>
                      <p>{mod.source_game_domain}/mods/{mod.source_mod_id}/files/{mod.source_file_id} · Priority {mod.priority}</p>
                      <div class="mod-advanced-actions">
                        <button type="button" class="secondary-action compact" on:click={() => moveModInProfile(mod, -1)} disabled={orderIndex <= 0}>Move Up</button>
                        <button type="button" class="secondary-action compact" on:click={() => moveModInProfile(mod, 1)} disabled={orderIndex < 0 || orderIndex >= installedMods.length - 1}>Move Down</button>
                      </div>
                    </details>
                  </article>
                {/each}
                </div>
              {/if}
              {#if showWorkshopModRows}
                <section class="platform-mod-panel" aria-label="Steam Workshop mods">
                  <div class="panel-heading compact-heading">
                    <h3>Steam Workshop</h3>
                    <span>{workshopItems.length} item{workshopItems.length === 1 ? "" : "s"}</span>
                  </div>
                  <p class="hint">{selectedWorkshop?.message ?? "Steam owns these subscriptions; DMM queues supported Steam actions through Decky."}</p>
                  {#if workshopState?.supported && workshopItems.length === 0}
                    <div class="mod-list">
                      <article>
                        <div>
                          <div class="mod-title-line">
                            <strong>No synced Workshop items</strong>
                            <span class={`source-pill ${sourceClass("steam_workshop")}`}>{sourceLabel("steam_workshop")}</span>
                          </div>
                          <p>Open DMM in the Decky sidebar while Steam is running to sync subscribed Workshop items.</p>
                        </div>
                        <div class="mod-actions">
                          <span>Sync needed</span>
                        </div>
                      </article>
                    </div>
                  {:else if workshopItems.length > 0}
                    <div class="mod-list">
                      {#each workshopItems as item, index}
                        {@const disabled = item.disabled_known && item.disabled_locally}
                        {@const toggleKind = disabled ? "enable" : "disable"}
                        {@const toggleSupported = Boolean(workshopState?.supported && item.disabled_known)}
                        <article>
                          <div>
                            <div class="mod-title-line">
                              <strong>{workshopItemName(item)}</strong>
                              <span class={`source-pill ${sourceClass(item.source_tag ?? item.catalog ?? "steam_workshop")}`}>{sourceLabel(item.source_tag ?? item.catalog ?? "steam_workshop")}</span>
                            </div>
                            <p>{workshopItemDetail(item)}</p>
                            <small>{workshopItemStatus(item)}</small>
                          </div>
                          <div class="mod-actions">
                            <span>{workshopItemStatus(item)}</span>
                            <button type="button" class="secondary-action compact" disabled={!workshopState?.supported || workshopOrderBusy || index === 0} on:click={() => moveWorkshopItem(index, -1)}>
                              Up
                            </button>
                            <button type="button" class="secondary-action compact" disabled={!workshopState?.supported || workshopOrderBusy || index === workshopItems.length - 1} on:click={() => moveWorkshopItem(index, 1)}>
                              Down
                            </button>
                            <button type="button" class="secondary-action compact" disabled={!toggleSupported || isWorkshopActionBusy(item, toggleKind)} on:click={() => queueWorkshopAction(item, toggleKind)}>
                              {isWorkshopActionBusy(item, toggleKind) ? "Queueing..." : !item.disabled_known ? "Sync Needed" : disabled ? "Enable" : "Disable"}
                            </button>
                            <button type="button" class="secondary-action compact danger-action" disabled={!workshopState?.supported || isWorkshopActionBusy(item, "unsubscribe")} on:click={() => askUnsubscribeWorkshopItem(item)}>
                              {isWorkshopActionBusy(item, "unsubscribe") ? "Queueing..." : "Unsubscribe"}
                            </button>
                          </div>
                        </article>
                      {/each}
                    </div>
                  {/if}
                </section>
              {/if}
            </section>
          {/if}

          <details class="deploy-preview">
            <summary>
              <span>Advanced Profile Tools</span>
              <small>
                {#if hasDeployConflicts}
                  Conflicts
                {:else if hasPendingProfileChanges}
                  Pending
                {:else if deploymentStatus?.deployed}
                  Applied
                {:else}
                  Not applied
                {/if}
              </small>
            </summary>
            {#if deployPlan && (deployableActions.length > 0 || deployPlan.conflicts.length > 0)}
              <div class="profile-change-summary" class:has-conflicts={deployPlan.conflicts.length > 0}>
                <div><strong>{deployAdds}</strong><span>Add</span></div>
                <div><strong>{deployReplaces}</strong><span>Update</span></div>
                <div><strong>{deployRemoves}</strong><span>Remove</span></div>
                <div><strong>{deployPlan.conflicts.length}</strong><span>Conflict</span></div>
              </div>
            {/if}
            <div class="deployment-summary">
              <div><strong>{enabledMods.length}</strong><span>Enabled</span></div>
              <div><strong>{deploymentStatus?.file_count ?? 0}</strong><span>Applied</span></div>
              <div><strong>{deployPlan?.conflicts.length ?? 0}</strong><span>Conflicts</span></div>
            </div>
            <section class="deployment-settings-card">
              <div>
                <strong>Deployment Strategy</strong>
                <small>
                  Profile: {deploymentSettings?.profile_name ?? selectedProfile?.name ?? "Default"} ·
                  Effective: {deploymentSettings?.effective_strategy ?? deployPlan?.strategy ?? "symlink"}
                </small>
                <small>
                  Source: {deploymentSettings?.source ?? "extension"} ·
                  Game default: {deploymentSettings?.game_strategy ?? "extension"} ·
                  Extension: {deploymentSettings?.extension_default ?? "symlink"}
                </small>
                {#if deploymentSettings?.recommended_strategy}
                  <small>Recommended: {deploymentSettings.recommended_strategy}</small>
                {/if}
              </div>
              <select aria-label="Deployment strategy" value={deploymentSettings?.strategy ?? "extension"} on:change={(event) => updateDeploymentStrategy(event.currentTarget.value)}>
                <option value="extension">Inherit Default</option>
                <option value="symlink">Symlink</option>
                <option value="hardlink">Hardlink</option>
                <option value="copy">Copy</option>
              </select>
              {#if deploymentSettings?.strategy_warnings?.length}
                <div class="deployment-strategy-warnings">
                  {#each deploymentSettings.strategy_warnings as warning}
                    <p>{warning}</p>
                  {/each}
                </div>
              {/if}
              {#if deploymentSettings?.capabilities?.length}
                <div class="deployment-capabilities" aria-label="Deployment strategy capabilities">
                  {#each deploymentSettings.capabilities as capability}
                    <article class:unsupported={!capability.supported} class:recommended={capability.recommended}>
                      <strong>{capability.strategy}</strong>
                      <span>{capability.supported ? "Supported" : "Not recommended"}{capability.recommended ? " · Recommended" : ""}</span>
                      <small>{capability.reason}</small>
                    </article>
                  {/each}
                </div>
              {/if}
            </section>
            {#if conflictChoiceTargets.length > 0}
              <section class="conflict-choice-card">
                <div class="panel-heading compact-heading">
                  <h3>File Winners</h3>
                  <span>{conflictChoiceTargets.length}</span>
                </div>
                <p class="hint">When enabled mods write the same file, DMM can use profile order or a file-specific winner.</p>
                <div class="conflict-choice-list">
                  {#each conflictChoiceTargets as target}
                    {@const currentCandidate = target.candidates.find((candidate) => candidate.current)}
                    <article>
                      <div>
                        <strong>{target.target_relative}</strong>
                        <small class="conflict-current-line">
                          <span>Current: {target.current_winner_name}</span>
                          {#if currentCandidate?.catalog}
                            <span class={`source-pill ${sourceClass(currentCandidate.catalog)}`}>{sourceLabel(currentCandidate.catalog)}</span>
                          {/if}
                          <span>{target.reason}</span>
                        </small>
                      </div>
                      <div class="conflict-choice-actions">
                        {#each target.candidates as candidate}
                          <button type="button" class="secondary-action compact" disabled={candidate.current} on:click={() => setFileConflictWinner(target, candidate.id)}>
                            <span>{candidate.current ? "Using" : "Use"} {candidate.name}</span>
                            {#if sourceForCandidate(candidate)}
                              <span class={`source-pill ${sourceClass(sourceForCandidate(candidate))}`}>{sourceLabel(sourceForCandidate(candidate))}</span>
                            {/if}
                          </button>
                        {/each}
                        <button type="button" class="secondary-action compact" on:click={() => clearFileConflictWinner(target)}>Use Profile Order</button>
                      </div>
                    </article>
                  {/each}
                </div>
              </section>
            {/if}
            {#if pluginLoadOrder?.supported}
              <section class="plugin-load-order-card">
                <div class="panel-heading compact-heading">
                  <h3>Plugin Load Order</h3>
                  <span>{pluginLoadOrder.plugins.length} plugin{pluginLoadOrder.plugins.length === 1 ? "" : "s"}</span>
                </div>
                <p>{pluginLoadOrder.name ?? "Extension plugin activation"} writes {pluginLoadOrder.plugins_file ?? "plugins.txt"} and {pluginLoadOrder.load_order_file ?? "loadorder.txt"} for enabled DMM plugins.</p>
                {#if pluginLoadOrder.target_root}
                  <small>{pluginLoadOrder.target_root}</small>
                {/if}
                {#if pluginLoadOrder.warnings?.length}
                  <div class="deployment-strategy-warnings">
                    {#each pluginLoadOrder.warnings as warning}
                      <p>{warning}</p>
                    {/each}
                  </div>
                {/if}
                {#if pluginLoadOrder.plugins.length > 0}
                  <div class="plugin-load-order-list">
                    {#each pluginLoadOrder.plugins as plugin, index}
                      <article>
                        <span>{index + 1}</span>
                        <div>
                          <div class="mod-title-line">
                            <strong>{plugin.name}</strong>
                            <span class={`source-pill ${sourceClass(plugin.catalog ?? plugin.source)}`}>{sourceLabel(plugin.catalog ?? plugin.source)}</span>
                          </div>
                          <small>{plugin.source === "native" ? "Game/native plugin" : `DMM mod ${plugin.mod_id ?? plugin.installed_mod_id ?? ""}`} · Priority {plugin.priority}</small>
                        </div>
                      </article>
                    {/each}
                  </div>
                {:else}
                  <p class="hint">No active plugin files are currently part of this profile.</p>
                {/if}
              </section>
            {/if}
            <section class="recovery-card">
              <div class="panel-heading compact-heading">
                <h3>Recovery</h3>
                <span>{deploymentStatus?.apply_rollback_on_failure ? "Auto-rollback" : "Manual review"}</span>
              </div>
              <p>{deploymentStatus?.recovery_summary ?? "Failed applies are restored before DMM reports the job as failed."}</p>
              {#if deploymentStatus?.sample_files?.length}
                <div class="sample-file-list">
                  {#each deploymentStatus.sample_files as file}
                    <small>{file}</small>
                  {/each}
                </div>
              {/if}
              {#if deploymentHistory.length > 0}
                <div class="deployment-history-list" aria-label="Deployment history">
                  {#each deploymentHistory as deployment}
                    <article>
                      <div>
                        <strong>{deployment.status === "deployed" ? "Active deployment" : "Previous deployment"}</strong>
                        <small>{deployment.profile_name} · {deployment.file_count} file{deployment.file_count === 1 ? "" : "s"} · {deployment.strategy}</small>
                      </div>
                      <time datetime={deployment.created_at}>{new Date(deployment.created_at).toLocaleString()}</time>
                    </article>
                  {/each}
                </div>
              {/if}
              <div class="deploy-actions utility-actions">
                <button type="button" class="secondary-action" on:click={askApplyPendingProfileChanges} disabled={!deployPlan || deployableActions.length === 0 || hasDeployConflicts}>Apply Enabled Mods</button>
                <button type="button" class="secondary-action" on:click={previewDeploy} disabled={installedMods.length === 0 && !deploymentStatus?.deployed}>Preview Managed Files</button>
                <button type="button" class="secondary-action" on:click={askRestoreDeployment} disabled={!deploymentStatus?.restore_available}>Restore Last Applied State</button>
                <button type="button" class="secondary-action" on:click={repairDeployment} disabled={!deploymentStatus?.repair_available}>Repair Managed Files</button>
                <button type="button" class="secondary-action" on:click={recoverDownloads}>Recover Downloads</button>
                <button type="button" class="secondary-action danger-action" on:click={askPurgeDeployment} disabled={!deploymentStatus?.purge_available}>Remove DMM Files</button>
                <button type="button" class="secondary-action danger-action" on:click={askResetManagedMods} disabled={installedMods.length === 0 && !deploymentStatus?.deployed && installCandidates.length === 0 && selectedGameActionItems.length === 0}>Reset Managed Mods</button>
              </div>
            </section>
            {#if deployPlan}
              <div class="panel-heading">
                <h3>File Operations</h3>
                <span>{deployPlan.actions.length} files · {deployPlan.conflicts.length} conflicts</span>
              </div>
              <div class="deploy-list">
                {#each deployPlan.actions.slice(0, 24) as action}
                  <article class:failed-action={action.conflict}>
                    <strong>{action.target_relative}</strong>
                    <small>{deployActionDetail(action)}</small>
                  </article>
                {/each}
              </div>
            {/if}
          </details>
        </article>
      {:else if activeGameModule === "actions"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Action Center</h2>
            <span>{selectedGameActionItems.length} open · {installCandidates.length} installers</span>
          </div>
          {#if selectedGameCapturedInstallActions.length > 0}
            <button type="button" class="secondary-action" on:click={clearCapturedInstallActions}>Clear Install Actions</button>
          {/if}
          {#if selectedGameActionItems.length === 0 && installCandidates.length === 0}
            <div class="empty-action-panel">
              <p class="hint">No install actions need attention for this game. Add a mod or return to the profile mod list.</p>
              <div class="empty-action-buttons">
                <button type="button" on:click={() => openGameModule("plugins")}>Add Mod</button>
                <button type="button" class="secondary-action compact" on:click={() => openGameModule("plugins")}>Profile Mods</button>
              </div>
            </div>
          {/if}
          {#if selectedGameActionItems.length > 0}
            <div class="action-list">
              {#each selectedGameActionItems as action}
                <article class:failed-action={action.status === "failed"}>
                  <div>
                    <div class="mod-title-line">
                      <strong>{action.title}</strong>
                      <span class={`source-pill ${sourceClass(actionSource(action))}`}>{sourceLabel(actionSource(action))}</span>
                    </div>
                    {#if action.message}<p>{action.message}</p>{/if}
                    <p class="action-next-step">{actionNextStep(action)}</p>
                    <small>{new Date(action.updated_at).toLocaleString()}</small>
                  </div>
                    <div class="action-controls">
                      <span>{actionStatusLabel(action)}</span>
                      {#if action.type === "installer-choice"}
                        <button type="button" on:click={() => openActionItem(action)}>Open Choices</button>
                      {/if}
                      {#if action.type === "captured-install" && action.status === "waiting"}
                        <button type="button" on:click={() => installCapturedMod(action)} disabled={isJobBusy(action)}>{isJobBusy(action) ? "Working..." : "Install"}</button>
                      {/if}
                      {#if action.type === "captured-install" && action.status === "failed"}
                        <button type="button" on:click={() => retryCapturedInstallAction(action)} disabled={isJobBusy(action)}>{isJobBusy(action) ? "Working..." : "Retry"}</button>
                      {/if}
                      {#if action.type === "steam-workshop-action" && action.status === "failed"}
                        <button type="button" on:click={() => retryWorkshopAction(action)} disabled={isJobBusy(action)}>{isJobBusy(action) ? "Working..." : "Retry"}</button>
                      {/if}
                      {#if canCancelJob(action)}
                        <button type="button" class="secondary-action compact" on:click={() => cancelJob(action)} disabled={isJobBusy(action)}>Cancel</button>
                      {/if}
                  </div>
                </article>
              {/each}
            </div>
          {/if}
          {#if installCandidates.length > 0}
            <section class="blocked-candidates" aria-label="Blocked install candidates">
              <div class="panel-heading compact-heading">
                <h3>Installer Choices</h3>
                <span>{installCandidates.length}</span>
              </div>
              <button type="button" class="secondary-action" on:click={clearBlockedInstallCandidates}>Clear Items</button>
              <div class="action-list">
                {#each installCandidates as candidate}
                  {@const installer = installerForCandidate(candidate)}
                  {@const selectedChoiceCount = selectionCount(candidateCurrentSelections(candidate))}
                  <article class="failed-action">
                    <div>
                      <div class="mod-title-line">
                        <strong>{candidate.name}</strong>
                        <span class={`source-pill ${sourceClass(sourceForCandidate(candidate))}`}>{sourceLabel(sourceForCandidate(candidate))}</span>
                      </div>
                      <p>{candidate.reason}</p>
                      <small>{candidate.source_game_domain}/mods/{candidate.source_mod_id}/files/{candidate.source_file_id}</small>
                      {#if selectedChoiceCount > 0}
                        <p class="preset-selection-note">{selectedChoiceCount} choice{selectedChoiceCount === 1 ? "" : "s"} preselected from DMM's saved/default installer state.</p>
                      {/if}
                      {#if installer}
                        {@const steps = visibleFomodSteps(installer)}
                        {@const stepIndex = candidateStepIndex(candidate, steps)}
                        {@const step = steps[stepIndex]}
                        <div class="installer-wizard">
                          <div class="installer-wizard-header">
                            <div>
                              <strong>{installer.name || "Installer Choices"}</strong>
                              <small>{steps.length > 0 ? `Step ${stepIndex + 1} of ${steps.length}` : "No choices"}</small>
                            </div>
                            <span>{candidateStepValid(candidate, step) ? "Ready" : "Needs selection"}</span>
                          </div>
                          {#if step}
                            <section class="installer-step">
                              <h4>{step.name}</h4>
                              {#each step.groups ?? [] as group}
                                <fieldset class:invalid-group={!candidateGroupValid(candidate, group)}>
                                  <legend>{group.name}</legend>
                                  {#if fomodGroupType(group) === "selectatmostone"}
                                    <label class="installer-none-option">
                                      <input
                                        type="radio"
                                        name={`candidate-${candidate.id}-${group.id}`}
                                        checked={(candidateCurrentSelections(candidate)[group.id] ?? []).length === 0}
                                        disabled={isInstallCandidateBusy(candidate)}
                                        on:change={() => clearCandidateGroupSelection(candidate, group)}
                                      />
                                      <span>
                                        <strong>None</strong>
                                        <small>Do not install an option from this group.</small>
                                      </span>
                                    </label>
                                  {/if}
                                  {#each group.plugins ?? [] as plugin}
                                    <label class:option-disabled={!fomodPluginSelectable(plugin)}>
                                      <input
                                        type={fomodGroupInputType(group)}
                                        name={`candidate-${candidate.id}-${group.id}`}
                                        checked={isCandidatePluginSelected(candidate, group, plugin)}
                                        disabled={fomodPluginLocked(group, plugin) || isInstallCandidateBusy(candidate)}
                                        on:change={(event) => setCandidatePluginSelection(candidate, group, plugin, event.currentTarget.checked)}
                                      />
                                      <span>
                                        <strong>{plugin.name}</strong>
                                        {#if fomodPluginType(plugin)}<small>{fomodPluginType(plugin)}</small>{/if}
                                        {#if plugin.description}<em>{plugin.description}</em>{/if}
                                      </span>
                                    </label>
                                  {/each}
                                  {#if !candidateGroupValid(candidate, group)}
                                    <p class="installer-validation">This group needs a valid selection before continuing.</p>
                                  {/if}
                                </fieldset>
                              {/each}
                            </section>
                            <div class="installer-wizard-actions">
                              <button type="button" class="secondary-action compact" disabled={stepIndex === 0 || isInstallCandidateBusy(candidate)} on:click={() => setCandidateStepIndex(candidate, steps, stepIndex - 1)}>Back</button>
                              <button type="button" class="secondary-action compact" disabled={stepIndex >= steps.length - 1 || !candidateStepValid(candidate, step) || isInstallCandidateBusy(candidate)} on:click={() => setCandidateStepIndex(candidate, steps, stepIndex + 1)}>Next</button>
                            </div>
                          {:else}
                            <p class="hint">This installer has no visible choices. Apply it to add the mod to this profile.</p>
                          {/if}
                        </div>
                      {/if}
                    </div>
                    <div class="action-controls">
                      <span>{candidateStatusLabel(candidate)}</span>
                      {#if installer}
                        <button type="button" on:click={() => applyInstallCandidate(candidate)} disabled={isInstallCandidateBusy(candidate) || !candidateInstallerValid(candidate, installer)}>
                          {isInstallCandidateBusy(candidate) ? "Applying..." : "Apply Choices"}
                        </button>
                      {:else if candidate.status === "blocked"}
                        <button type="button" on:click={() => retryInstallCandidate(candidate)} disabled={isInstallCandidateBusy(candidate)}>
                          {isInstallCandidateBusy(candidate) ? "Retrying..." : "Retry Install"}
                        </button>
                      {/if}
                    </div>
                  </article>
                {/each}
              </div>
            </section>
          {/if}
          {#if installerChoicePresets.length > 0}
            <section class="blocked-candidates" aria-label="Saved installer choices">
              <div class="panel-heading compact-heading">
                <h3>Saved Installer Choices</h3>
                <span>{installerChoicePresets.length}</span>
              </div>
              <p class="hint">DMM reuses these choices only for the same exact mod file. Forget a preset if you want the installer to ask again.</p>
              <div class="action-list">
                {#each installerChoicePresets as preset}
                  <article>
                    <div>
                      <div class="mod-title-line">
                        <strong>{preset.installer_kind.toUpperCase()} choices</strong>
                        <span class={`source-pill ${sourceClass(preset.catalog)}`}>{sourceLabel(preset.catalog)}</span>
                      </div>
                      <p>{preset.source_game_domain}/mods/{preset.source_mod_id}/files/{preset.source_file_id}</p>
                      <small>{installerPresetScopeLabel(preset)} · Updated {new Date(preset.updated_at).toLocaleString()}</small>
                    </div>
                    <div class="action-controls">
                      <span>Saved</span>
                      <button type="button" class="secondary-action compact" on:click={() => deleteInstallerChoicePreset(preset)}>Forget</button>
                    </div>
                  </article>
                {/each}
              </div>
            </section>
          {/if}
        </article>
      {:else if activeGameModule === "profiles"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Profiles</h2>
            <span>{selectedProfile?.name ?? "None"}</span>
          </div>
          <form class="inline-form" on:submit|preventDefault={createProfile}>
            <input bind:value={profileName} aria-label="Profile name" placeholder="Profile name" />
            <button type="submit">Create</button>
          </form>
          <div class="profile-list">
            {#each profiles as profile}
              <button type="button" class:active-profile={profile.is_default} on:click={() => setDefaultProfile(profile)}>
                <span>{profile.name}</span>
                <strong>{profile.is_default ? "Default" : "Set default"}</strong>
              </button>
            {/each}
          </div>
        </article>
      {:else if activeGameModule === "review"}
        <article class="workspace-panel">
          <h2>Review</h2>
          {#if gameDiagnostics?.runtime_requirements?.length}
            <section class="requirement-list" aria-label="Runtime requirements">
              <div class="panel-heading compact-heading">
                <h3>Runtime Requirements</h3>
                <span>{gameDiagnostics.runtime_requirements.length}</span>
              </div>
              {#each gameDiagnostics.runtime_requirements as requirement}
                <article class:requirement-missing={requirement.status !== "ok"}>
                  <div>
                    <strong>{requirement.name}</strong>
                    <p>{requirement.message}</p>
                    {#if requirement.details?.length}
                      <ul class="requirement-details">
                        {#each requirement.details as detail}
                          <li>{detail}</li>
                        {/each}
                      </ul>
                    {/if}
                    {#if requirement.install_hint}<small>{requirement.install_hint}</small>{/if}
                    {#if requirement.help_url}<a href={requirement.help_url} target="_blank" rel="noreferrer">Open help</a>{/if}
                    {#if requirement.kind === "launch-tool" && requirement.status !== "ok" && launchSetupAvailable}
                      <div class="requirement-actions">
                        <button type="button" on:click={applyLaunchSetup}>Configure Launch Tool</button>
                        {#if gameLaunchStatus?.action?.risk === "replaces-existing-launch-options"}
                          <small>Existing Steam launch options will be replaced.</small>
                        {/if}
                      </div>
                    {/if}
                  </div>
                  <span>{requirement.status}</span>
                </article>
              {/each}
            </section>
          {/if}
          {#if selectedWorkshop}
            <section class="requirement-list" aria-label="Steam Workshop state">
              <div class="panel-heading compact-heading">
                <h3>Steam Workshop</h3>
                <span>{selectedWorkshop.item_count} item{selectedWorkshop.item_count === 1 ? "" : "s"}</span>
              </div>
              <article class:requirement-missing={!selectedWorkshop.coexistence_allowed}>
                <div>
                  <strong>{selectedWorkshop.coexistence_allowed ? "Workshop content detected" : "Workshop content needs review"}</strong>
                  <p>{selectedWorkshop.message ?? "Steam Workshop content is present for this game."}</p>
                  <ul class="requirement-details">
                    {#if selectedWorkshop.content_path}<li>{selectedWorkshop.content_path}</li>{/if}
                    {#if selectedWorkshop.manifest_path}<li>{selectedWorkshop.manifest_path}</li>{/if}
                    {#if selectedWorkshop.sample_item_ids?.length}<li>Sample items: {selectedWorkshop.sample_item_ids.join(", ")}</li>{/if}
                  </ul>
                  <small>{selectedWorkshop.management_supported ? "Workshop management is supported by this extension." : "DMM leaves Workshop subscription and enable state to Steam for now."}</small>
                </div>
                <span>{selectedWorkshop.coexistence_allowed ? "External" : "Review"}</span>
              </article>
              {#if workshopState?.supported}
                <article>
                  <div>
                    <strong>Managed from Mods</strong>
                    <p>Use the Mods view to reorder, enable, disable, or unsubscribe synced Workshop items alongside DMM-managed mods.</p>
                  </div>
                  <span>Mods</span>
                </article>
              {/if}
            </section>
          {/if}
          {#if visibleValidationWarnings.length}
            <section class="requirement-list" aria-label="Validation warnings">
              <div class="panel-heading compact-heading">
                <h3>Warnings</h3>
                <span>{visibleValidationWarnings.length}</span>
              </div>
              {#each visibleValidationWarnings as warning}
                <article class="requirement-missing">
                  <div>
                    <strong>Needs attention</strong>
                    <p>{warning}</p>
                  </div>
                </article>
              {/each}
            </section>
          {/if}
          {#if selectedGame.markers?.length}
            <div class="markers">
              {#each selectedGame.markers as marker}
                <span>{marker}</span>
              {/each}
            </div>
          {:else if !selectedWorkshop && !gameDiagnostics?.runtime_requirements?.length && !visibleValidationWarnings.length}
            <p class="hint">No review markers for this game.</p>
          {/if}
        </article>
      {:else}
        <article class="workspace-panel path-panel">
          <h2>Paths</h2>
          <dl class="settings-list">
            <div><dt>Install</dt><dd>{selectedGame.path}</dd></div>
            <div><dt>Library</dt><dd>{selectedGame.library_path}</dd></div>
          </dl>
        </article>
      {/if}
    </section>
  {:else}
    <section class="empty-state select-game">
      <h2>Select a Game</h2>
      <p class="hint">Open the games menu to choose a Steam game before importing mods or managing profiles.</p>
      <button type="button" on:click={() => (drawer = "games")}>Open Games</button>
    </section>
  {/if}
</main>

{#if confirmation}
  <section class="confirm-layer" aria-label="Confirm action">
    <button type="button" class="confirm-scrim" aria-label="Cancel confirmation" on:click={() => (confirmation = null)}></button>
    <article class:danger-confirm={confirmation.danger} class="confirm-dialog">
      <div>
        <p class="eyebrow">Confirm</p>
        <h2>{confirmation.title}</h2>
      </div>
      <p>{confirmation.message}</p>
      {#if confirmation.detail}
        <p class="confirm-detail">{confirmation.detail}</p>
      {/if}
      <div class="confirm-actions">
        <button type="button" class="secondary-action" on:click={() => (confirmation = null)}>Cancel</button>
        <button type="button" class:danger-action={confirmation.danger} on:click={confirmCurrentAction}>{confirmation.confirmLabel}</button>
      </div>
    </article>
  </section>
{/if}
