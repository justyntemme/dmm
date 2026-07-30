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
    nexus: { api_key_configured: boolean };
    ui?: UISettings;
  };

  type Game = {
    app_id: string;
    name: string;
    path: string;
    library_path: string;
    state: string;
    markers?: string[];
    nexus_domains?: string[];
  };

  type Profile = {
    id: number;
    game_id: number;
    name: string;
    is_default: boolean;
  };

  type Job = {
    id: string;
    type: string;
    title: string;
    status: string;
    message: string;
    payload?: Record<string, string>;
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

  type InstalledMod = {
    id: number;
    name: string;
    profile_id: number;
    catalog: string;
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
    source_game_domain: string;
    source_mod_id: string;
    source_file_id: string;
    archive_path: string;
    status: string;
    reason: string;
    installer_json?: string;
    choices_json?: string;
  };

  type FomodInstaller = {
    name: string;
    steps?: FomodStep[];
  };

  type FomodStep = {
    id: string;
    name: string;
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
  };

  type DeployAction = {
    source_path: string;
    target_path: string;
    target_relative: string;
    strategy: string;
    operation: string;
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

  type DeploymentStatus = {
    deployed: boolean;
    file_count: number;
    strategy?: string;
    sample_files?: string[];
    apply_rollback_on_failure: boolean;
    repair_available: boolean;
    purge_available: boolean;
    recovery_summary?: string;
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
  type SettingsPage = "overview" | "jobs" | "install" | "nexus";
  type GameSort = "recent" | "az" | "za";

  let status: Status | null = null;
  let games: Game[] = [];
  let jobs: Job[] = [];
  let selectedGame: Game | null = null;
  let profiles: Profile[] = [];
  let installedMods: InstalledMod[] = [];
  let installCandidates: InstallCandidate[] = [];
  let profileName = "";
  let importURL = "";
  let lastImportURL = "";
  let resolvedImport = "";
  let nexusFiles: NexusFile[] = [];
  let downloadLinks: DownloadLink[] = [];
  let deployPlan: DeployPlan | null = null;
  let deploymentStatus: DeploymentStatus | null = null;
  let gameDiagnostics: GameDiagnostics | null = null;
  let gameLaunchStatus: GameLaunchStatus | null = null;
  let loading = true;
  let error = "";
  let drawer: Drawer = null;
  let confirmation: Confirmation | null = null;
  let surface: Surface = "actions";
  let activeGameModule: GameModule = "plugins";
  let activeSettingsPage: SettingsPage = "overview";
  let gameQuery = "";
  let gameSort: GameSort = "recent";
  let favoriteGameIDs = new Set<string>();
  let gameRecent: Record<string, number> = {};
  let busyJobs: Record<string, boolean> = {};
  let busyInstallCandidates: Record<number, boolean> = {};
  let initialRefreshComplete = false;
  let selectedGameRefreshTimer: number | null = null;
  let selectedGameRefreshNeedsPreview = false;
  let selectedGameRefreshNeedsJobs = false;
  let candidateSelections: Record<number, Record<string, string[]>> = {};
  let refreshJobsInFlight = false;
  let refreshJobsQueued = false;
  let eventSocket: WebSocket | null = null;
  let eventReconnectTimer: number | null = null;
  let eventReconnectDelay = 1000;
  let lastEventID = 0;

  $: cleanCount = games.filter((game) => game.state === "clean_candidate").length;
  $: reviewCount = games.length - cleanCount;
  $: selectedProfile = profiles.find((profile) => profile.is_default) ?? profiles[0] ?? null;
  $: installRequests = jobs.filter((job) => job.type === "pending-import" && !["completed", "canceled"].includes(job.status));
  $: actionItems = jobs.filter((job) => ["pending-import", "installer-choice"].includes(job.type) && !["completed", "canceled"].includes(job.status));
  $: selectedGameRequests = selectedGame ? installRequests.filter((job) => requestMatchesGame(job, selectedGame)) : installRequests;
  $: selectedGameActionItems = selectedGame ? actionItems.filter((job) => requestMatchesGame(job, selectedGame)) : actionItems;
  $: selectedGameActivity = selectedGame
    ? jobs.filter((job) => {
        if (job.type === "pending-import") return requestMatchesGame(job, selectedGame) && !["completed", "canceled"].includes(job.status);
        return ["installer-choice", "deploy", "purge", "repair", "recover-downloads", "rollback"].includes(job.type) && jobMatchesGame(job, selectedGame) && !["completed", "canceled"].includes(job.status);
      })
    : [];
  $: filteredGames = sortDrawerGames(games.filter((game) => {
    const query = gameQuery.trim().toLowerCase();
    if (!query) return true;
    return game.name.toLowerCase().includes(query) || game.app_id.includes(query);
  }));
  $: title = surface === "settings" ? settingsTitle(activeSettingsPage) : surface === "actions" ? "Action Center" : selectedGame?.name ?? "Select a Game";
  $: deployableActions = getDeployableActions(deployPlan);
  $: enabledMods = installedMods.filter((mod) => mod.enabled);
  $: disabledMods = installedMods.filter((mod) => !mod.enabled);
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

  function logClientEvent(message: string, detail: Record<string, string | number | boolean | null | undefined> = {}) {
    const body = JSON.stringify({ message, detail });
    fetch("/api/client-events", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body
    }).catch(() => {
      // Client diagnostics must not interfere with the mod-management flow.
    });
  }

  async function refresh() {
    error = "";
    try {
      const [nextStatus, nextGames] = await Promise.all([
        getJSON<Status>("/api/status"),
        getJSON<Game[]>("/api/games")
      ]);
      status = nextStatus;
      applyUIPreferences(nextStatus);
      games = nextGames;
      const previousSelection = selectedGame?.app_id;
      selectedGame = nextGames.find((game) => game.app_id === previousSelection) ?? null;
      if (selectedGame) await loadGameState(selectedGame);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
      initialRefreshComplete = true;
    }
    try {
      jobs = await getJSON<Job[]>("/api/jobs");
      reconcileBusyState();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
  }

  async function refreshJobsAndSelectedGame() {
    if (refreshJobsInFlight) {
      refreshJobsQueued = true;
      return;
    }
    refreshJobsInFlight = true;
    try {
      jobs = await getJSON<Job[]>("/api/jobs");
      if (selectedGame) await refreshSelectedGame({ refreshPreview: deployPlan !== null });
      reconcileBusyState();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      refreshJobsInFlight = false;
      if (refreshJobsQueued) {
        refreshJobsQueued = false;
        void refreshJobsAndSelectedGame();
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
    const detail: Record<string, string | number | boolean | null | undefined> = {
      id: event.id,
      type: event.type,
      app_id: event.app_id,
      job_id: event.job_id
    };
    if (isJob(event.payload)) {
      detail.job_type = event.payload.type;
      detail.job_status = event.payload.status;
      detail.job_message = event.payload.message;
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
      void refreshJobsAndSelectedGame();
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
      }
      return;
    }
    if (event.type === "job.updated") {
      if (isJob(event.payload)) {
        upsertJob(event.payload);
        if (selectedGame && jobMatchesGame(event.payload, selectedGame)) {
          scheduleSelectedGameRefresh(event.payload.status === "completed" || deployPlan !== null || event.payload.type === "installer-choice", event.payload.status === "completed");
        }
      }
      return;
    }
    if (event.type === "launch.changed") {
      if (selectedGame && eventMatchesSelectedGame(event)) {
        if (isGameLaunchStatus(event.payload)) gameLaunchStatus = event.payload;
        scheduleSelectedGameRefresh(false, true);
      }
      return;
    }
    if (event.type === "game.changed") {
      void refresh();
      return;
    }
    if (["profile_mods.changed", "deployment.changed", "install.changed"].includes(event.type) && eventMatchesSelectedGame(event)) {
      scheduleSelectedGameRefresh(true, true);
    }
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

  function scheduleSelectedGameRefresh(refreshPreview = false, refreshJobs = false) {
    if (!selectedGame || !initialRefreshComplete) return;
    selectedGameRefreshNeedsPreview = selectedGameRefreshNeedsPreview || refreshPreview;
    selectedGameRefreshNeedsJobs = selectedGameRefreshNeedsJobs || refreshJobs;
    if (selectedGameRefreshTimer !== null) return;
    selectedGameRefreshTimer = window.setTimeout(async () => {
      const shouldRefreshPreview = selectedGameRefreshNeedsPreview;
      const shouldRefreshJobs = selectedGameRefreshNeedsJobs;
      selectedGameRefreshTimer = null;
      selectedGameRefreshNeedsPreview = false;
      selectedGameRefreshNeedsJobs = false;
      if (shouldRefreshJobs) {
        jobs = await getJSON<Job[]>("/api/jobs");
      }
      await refreshSelectedGame({ refreshPreview: shouldRefreshPreview });
      reconcileBusyState();
    }, 250);
  }

  async function selectGame(game: Game) {
    markGameRecent(game.app_id);
    selectedGame = game;
    surface = "game";
    activeGameModule = "plugins";
    drawer = null;
    resolvedImport = "";
    nexusFiles = [];
    downloadLinks = [];
    deployPlan = null;
    deploymentStatus = null;
    gameDiagnostics = null;
    gameLaunchStatus = null;
    installCandidates = [];
    await loadGameState(game);
    await previewDeploy();
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
    const [nextProfiles, nextMods, nextCandidates, nextDeploymentStatus, nextDiagnostics, nextLaunchStatus] = await Promise.all([
      getJSON<Profile[]>(`/api/games/${game.app_id}/profiles`),
      getJSON<InstalledMod[]>(`/api/games/${game.app_id}/mods`),
      getJSON<InstallCandidate[]>(`/api/games/${game.app_id}/install-candidates`),
      getJSON<DeploymentStatus>(`/api/games/${game.app_id}/deploy/status`),
      getJSON<GameDiagnostics>(`/api/games/${game.app_id}/diagnostics`),
      getJSON<GameLaunchStatus>(`/api/games/${game.app_id}/launch`)
    ]);
    profiles = nextProfiles;
    installedMods = nextMods;
    installCandidates = nextCandidates;
    deploymentStatus = nextDeploymentStatus;
    gameDiagnostics = nextDiagnostics;
    gameLaunchStatus = nextLaunchStatus;
    reconcileBusyState();
  }

  async function refreshSelectedGame(options: { refreshPreview?: boolean } = {}) {
    if (!selectedGame) return;
    await loadGameState(selectedGame);
    if (options.refreshPreview) await previewDeploy();
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
  }

  async function setModPriority(mod: InstalledMod, priority: number) {
    if (!selectedProfile) return;
    error = "";
    const response = await fetch(`/api/profiles/${selectedProfile.id}/mods/${mod.id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ priority })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result: ProfileModUpdateResult = await response.json();
    installedMods = installedMods
      .map((item) => (item.id === result.mod.id ? result.mod : item))
      .sort((a, b) => a.priority - b.priority || a.name.localeCompare(b.name));
    handleProfileApplyResult(result.apply);
    await refreshSelectedGame({ refreshPreview: true });
  }

  async function removeInstalledMod(mod: InstalledMod) {
    if (!selectedGame) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/mods/${mod.id}`, { method: "DELETE" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result: { removed: InstalledMod; apply: ProfileApplyResult } = await response.json();
    installedMods = installedMods.filter((item) => item.id !== mod.id);
    handleProfileApplyResult(result.apply);
    await refreshSelectedGame({ refreshPreview: true });
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

  async function resolveImport() {
    if (!selectedGame || !importURL.trim()) return;
    error = "";
    lastImportURL = importURL;
    nexusFiles = [];
    downloadLinks = [];
    const response = await fetch("/api/imports/resolve", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url: importURL })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    resolvedImport = `${result.resolved.catalog}:${result.resolved.game_domain}/mods/${result.resolved.mod_id}${result.resolved.file_id ? `/files/${result.resolved.file_id}` : ""}`;
    nexusFiles = result.files ?? [];
    downloadLinks = result.download_links ?? [];
    importURL = "";
  }

  async function resolveFile(file: NexusFile) {
    if (!lastImportURL) return;
    const nextURL = new URL(lastImportURL);
    nextURL.searchParams.set("file_id", String(file.file_id));
    importURL = nextURL.toString();
    await resolveImport();
  }

  async function clearInstallRequests() {
    error = "";
    const response = await fetch("/api/imports/pending", { method: "DELETE" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    jobs = jobs.filter((job) => job.type !== "pending-import" || ["completed", "canceled"].includes(job.status));
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
    await refreshSelectedGame({ refreshPreview: deployPlan !== null });
  }

  function installerForCandidate(candidate: InstallCandidate): FomodInstaller | null {
    if (!candidate.installer_json) return null;
    try {
      return JSON.parse(candidate.installer_json) as FomodInstaller;
    } catch (_err) {
      return null;
    }
  }

  function defaultCandidateSelections(installer: FomodInstaller): Record<string, string[]> {
    const out: Record<string, string[]> = {};
    for (const step of installer.steps ?? []) {
      for (const group of step.groups ?? []) {
        const plugins = group.plugins ?? [];
        const type = group.type.toLowerCase();
        if (type === "selectall") {
          out[group.id] = plugins.map((plugin) => plugin.id);
          continue;
        }
        const preferred = plugins.find((plugin) => preferredFomodType(plugin.type)) ?? plugins[0];
        if (!preferred) continue;
        if (type === "selectexactlyone" || type === "selectatleastone" || type === "") {
          out[group.id] = [preferred.id];
          continue;
        }
        out[group.id] = plugins.filter((plugin) => preferredFomodType(plugin.type)).map((plugin) => plugin.id);
      }
    }
    return out;
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

  function preferredFomodType(type: string | undefined) {
    const normalized = (type ?? "").trim().toLowerCase();
    return normalized === "required" || normalized === "recommended";
  }

  function fomodGroupType(group: FomodGroup) {
    return (group.type ?? "").trim().toLowerCase();
  }

  function fomodGroupInputType(group: FomodGroup) {
    const type = fomodGroupType(group);
    return type === "selectexactlyone" || type === "selectatmostone" ? "radio" : "checkbox";
  }

  function candidateCurrentSelections(candidate: InstallCandidate, installer: FomodInstaller) {
    return candidateSelections[candidate.id] ?? storedCandidateSelections(candidate) ?? defaultCandidateSelections(installer);
  }

  function isCandidatePluginSelected(candidate: InstallCandidate, installer: FomodInstaller, group: FomodGroup, plugin: FomodPlugin) {
    return (candidateCurrentSelections(candidate, installer)[group.id] ?? []).includes(plugin.id);
  }

  function setCandidatePluginSelection(candidate: InstallCandidate, installer: FomodInstaller, group: FomodGroup, plugin: FomodPlugin, checked: boolean) {
    const current = candidateCurrentSelections(candidate, installer);
    const next = { ...current };
    const type = group.type.toLowerCase();
    if (type === "selectall") return;
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
      }
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
      const selections = candidateCurrentSelections(candidate, installer);
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
      upsertJob(result.job);
      if (result.mod) {
        installedMods = [result.mod, ...installedMods.filter((mod) => mod.id !== result.mod.id)];
      }
      installCandidates = installCandidates.filter((item) => item.id !== candidate.id);
      candidateSelections = Object.fromEntries(Object.entries(candidateSelections).filter(([id]) => Number(id) !== candidate.id));
      await refreshSelectedGame({ refreshPreview: true });
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
      }
      await refreshSelectedGame({ refreshPreview: true });
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
        await refreshJobsAndSelectedGame();
        return;
      }
      const result = await response.json();
      upsertJob(result.job);
      await refreshJobsAndSelectedGame();
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setJobBusy(job.id, false);
    }
  }

  async function approveInstallRequest(request: Job) {
    if (request.status !== "waiting") return;
    error = "";
    setJobBusy(request.id, true);
    markJobProcessing(request, "Installing downloaded archive...");
    try {
      const response = await fetch(`/api/imports/pending/${request.id}/approve`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        await refreshJobsAndSelectedGame();
        return;
      }
      const result = await response.json();
      upsertJob(result.job);
      await refreshJobsAndSelectedGame();
      if (selectedGame) await refreshSelectedGame({ refreshPreview: true });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setJobBusy(request.id, false);
    }
  }

  async function retryInstallRequest(request: Job) {
    if (request.status !== "failed") return;
    error = "";
    setJobBusy(request.id, true);
    markJobProcessing(request, "Retrying captured install...");
    try {
      const response = await fetch(`/api/imports/pending/${request.id}/retry`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        await refreshJobsAndSelectedGame();
        return;
      }
      const result = await response.json();
      upsertJob(result.job);
      await refreshJobsAndSelectedGame();
      if (selectedGame) await refreshSelectedGame({ refreshPreview: true });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      setJobBusy(request.id, false);
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
      title: "Apply profile",
      message: `DMM will update ${selectedGame.name}'s game folder to match the selected profile.`,
      detail: `${adds} add, ${replaces} replace, ${removes} remove. Advanced file details remain available before or after applying.`,
      confirmLabel: "Apply Profile",
      run: applyPendingProfileChanges
    };
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
      title: "Purge deployed files",
      message: `DMM will remove the active deployment manifest from ${selectedGame.name}'s game folder.`,
      detail: `${deploymentStatus.file_count} DMM-owned file${deploymentStatus.file_count === 1 ? "" : "s"} will be removed. Unmanaged files and parent game directories are left alone.`,
      confirmLabel: "Purge Deployment",
      danger: true,
      run: purgeDeployment
    };
  }

  function askResetManagedMods() {
    if (!selectedGame) return;
    confirmation = {
      title: "Reset managed mods",
      message: `DMM will remove its managed mods for ${selectedGame.name}.`,
      detail: "This purges DMM-owned deployed files, removes installed mod rows, deletes staging folders, and clears pending installer work. Cached downloads are kept for recovery.",
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

  function upsertJob(job: Job) {
    const existing = jobs.find((item) => item.id === job.id);
    const changed = !existing || existing.status !== job.status || existing.message !== job.message || existing.updated_at !== job.updated_at;
    if (job.type === "pending-import" && job.status === "canceled" && job.message === "Cleared") {
      jobs = jobs.filter((item) => item.id !== job.id);
      return;
    }
    jobs = [job, ...jobs.filter((item) => item.id !== job.id)];
    reconcileBusyState();
    if (!changed || !selectedGame || !jobMatchesGame(job, selectedGame)) return;
    if (["pending-import", "installer-choice", "deploy", "purge", "repair", "recover-downloads", "rollback"].includes(job.type)) {
      scheduleSelectedGameRefresh(job.status === "completed" || deployPlan !== null, job.status === "completed");
    }
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
    if (page === "nexus") return "Nexus";
    return "Settings";
  }

  function requestMatchesGame(job: Job, game: Game) {
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
    return games.find((game) => requestMatchesGame(job, game)) ?? null;
  }

  async function openActionItem(job: Job) {
    const game = gameForJob(job);
    if (!game) return;
    await selectGame(game);
    activeGameModule = "actions";
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
    if (mod.status === "staged") return mod.enabled ? "Enabled" : "Installed";
    return mod.status;
  }

  function modProfileStateText(mod: InstalledMod) {
    if (mod.status === "needs_recovery") return "Needs repair before it can be used";
    return mod.enabled ? "Enabled in this profile" : "Installed, disabled in this profile";
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

  function requestNextStep(request: Job) {
    if (request.type === "installer-choice") {
      return "Choose installer options to finish adding this mod to the selected profile.";
    }
    if (request.status === "waiting") {
      return "Add this downloaded mod to the selected profile, or cancel it and keep the archive cache untouched.";
    }
    if (request.status === "running" || request.status === "queued") {
      return "DMM is downloading or installing this mod. It will appear in the profile when ready.";
    }
    if (request.status === "failed") {
      return "The mod was not added. Retry from the cached download when available, or clear it if this action is no longer needed.";
    }
    return "This request is retained in job history for diagnostics.";
  }

  function requestStatusLabel(request: Job) {
    if (request.type === "installer-choice" && request.status === "waiting") return "Needs choices";
    if (request.status === "waiting") return "Ready to install";
    if (request.status === "running") return "Processing";
    if (request.status === "queued") return "Queued";
    if (request.status === "failed") return "Failed";
    return request.status;
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
      void refreshJobsAndSelectedGame();
    };
    const refreshOnVisibility = () => {
      if (!document.hidden) void refreshJobsAndSelectedGame();
    };
    window.addEventListener("focus", refreshOnFocus);
    document.addEventListener("visibilitychange", refreshOnVisibility);
    return () => {
      closeEventSocket();
      window.removeEventListener("focus", refreshOnFocus);
      document.removeEventListener("visibilitychange", refreshOnVisibility);
      if (selectedGameRefreshTimer !== null) window.clearTimeout(selectedGameRefreshTimer);
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
          <span>{actionItems.length} open</span>
        </div>
        {#if installRequests.length > 0}
          <button type="button" class="secondary-action" on:click={clearInstallRequests}>Clear Install Actions</button>
        {/if}
        {#if actionItems.length === 0}
          <div class="request-home">
            <div class="empty-state inline-empty">
              <h2>No Actions Needed</h2>
              <p class="hint">Open a game to paste a Nexus URL, or capture an nxm:// link from the Deck browser flow.</p>
            </div>
            <button type="button" on:click={() => (drawer = "games")}>Choose Game</button>
          </div>
        {:else}
          <div class="request-list">
            {#each actionItems as request}
              <article class:failed-request={request.status === "failed"}>
                <div>
                  <strong>{request.title}</strong>
	                    {#if request.message}<p>{request.message}</p>{/if}
	                    <p class="request-next-step">{requestNextStep(request)}</p>
	                    <small>{new Date(request.updated_at).toLocaleString()}</small>
	                  </div>
	                  <div class="request-actions">
	                    <span>{requestStatusLabel(request)}</span>
	                    {#if request.type === "installer-choice"}
	                      <button type="button" on:click={() => openActionItem(request)}>{gameForJob(request) ? "Open Choices" : "Choose Game"}</button>
	                    {/if}
	                    {#if request.status === "waiting"}
	                      {#if request.type === "pending-import"}
	                        <button type="button" on:click={() => approveInstallRequest(request)} disabled={isJobBusy(request)}>{isJobBusy(request) ? "Working..." : "Install"}</button>
	                      {/if}
	                    {/if}
	                    {#if request.type === "pending-import" && request.status === "failed"}
	                      <button type="button" on:click={() => retryInstallRequest(request)} disabled={isJobBusy(request)}>{isJobBusy(request) ? "Working..." : "Retry"}</button>
	                    {/if}
	                    {#if canCancelJob(request)}
	                      <button type="button" class="secondary-action compact" on:click={() => cancelJob(request)} disabled={isJobBusy(request)}>Cancel</button>
	                    {/if}
	                  </div>
	                </article>
            {/each}
          </div>
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
            <div><dt>Captured installs</dt><dd>{status?.install.auto_install_captured_downloads ? "Install automatically" : "Ask first"}</dd></div>
            <div><dt>Auto enable</dt><dd>{status?.install.auto_enable_installed_mods ? "Enabled" : "Disabled"}</dd></div>
          </dl>
        </article>
      {:else if activeSettingsPage === "jobs"}
        <article class="workspace-panel">
          <h2>Jobs</h2>
          {#if jobs.length === 0}
            <p class="hint">No jobs yet.</p>
          {:else}
            <div class="jobs">
              {#each jobs as job}
                <article class="job">
	                  <div>
	                    <strong>{job.title}</strong>
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
            <div><dt>Captured installs</dt><dd>{status?.install.auto_install_captured_downloads ? "Install automatically" : "Ask first"}</dd></div>
            <div><dt>New mod state</dt><dd>{status?.install.auto_enable_installed_mods ? "Enable automatically" : "Install disabled"}</dd></div>
          </dl>
          <p class="hint">These Deck behavior switches are managed from the Decky sidebar settings.</p>
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
        <button type="button" class:active={activeGameModule === "plugins"} on:click={() => (activeGameModule = "plugins")}>Mods</button>
        <button type="button" class:active={activeGameModule === "actions"} on:click={() => (activeGameModule = "actions")}>Actions</button>
        <button type="button" class:active={activeGameModule === "profiles"} on:click={() => (activeGameModule = "profiles")}>Profiles</button>
        <button type="button" class:active={activeGameModule === "review"} on:click={() => (activeGameModule = "review")}>Review</button>
        <button type="button" class:active={activeGameModule === "paths"} on:click={() => (activeGameModule = "paths")}>Paths</button>
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
                <article class:failed-request={job.status === "failed"}>
                  <strong>{job.title}</strong>
                  <span>{job.status}</span>
                  {#if job.message}<small>{job.message}</small>{/if}
                </article>
              {/each}
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
                <div><strong>{selectedGameActionItems.length}</strong><span>Actions</span></div>
              </div>
              {#if hasDeployConflicts}
                <p class="deploy-message danger">This profile has conflicts that need review before it can be applied.</p>
              {:else if hasPendingProfileChanges}
                <p class="deploy-message">This profile has changes ready. Enabling or disabling a mod applies the profile automatically; Advanced Deployment Tools can apply it now.</p>
              {:else if deploymentStatus?.deployed}
                <p class="deploy-message success">This profile is applied to the game.</p>
              {:else if enabledMods.length === 0}
                <p class="deploy-message">No enabled mods are applied for this profile.</p>
              {:else}
                <p class="deploy-message">Enable or disable a mod to apply this profile.</p>
              {/if}
            </div>

            <div class="management-card import-card">
              <div class="card-heading">
                <h3>Add From Nexus</h3>
                <span>{selectedGameActionItems.length} open</span>
              </div>
              <form class="stacked-form" on:submit|preventDefault={resolveImport}>
                <textarea bind:value={importURL} rows="4" aria-label="Nexus URL" placeholder="Nexus mod URL or nxm:// Mod Manager Download link"></textarea>
                <button type="submit">Add Mod</button>
              </form>
              <p class="hint">Use a Nexus mod page URL or a Mod Manager Download nxm:// link. Downloads that need unsupported installer logic will be kept as blocked install candidates.</p>
              {#if resolvedImport}
                <p class="hint">Resolved {resolvedImport}</p>
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

          {#if installedMods.length === 0}
            <p class="hint">No profile mods yet. Capture or paste a Nexus mod link to add a supported mod to this profile.</p>
          {:else}
            <section class="mod-section">
              <div class="card-heading">
                <h3>Mods</h3>
                <span>{selectedProfile?.name ?? "Default"}</span>
              </div>
              <div class="mod-list">
                {#each installedMods as mod}
                  {@const metadata = primaryModMetadata(mod)}
                  {@const dependencyLabels = modDependencyLabels(mod)}
                  <article>
                    <div>
                      <strong>{mod.name}</strong>
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
                      <small>{modProfileStateText(mod)}</small>
                    </div>
                    <div class="mod-actions">
                      <span class:warning-status={mod.status === "needs_recovery"}>{modStatusText(mod)}</span>
                      <label class="mod-toggle">
                        <input type="checkbox" checked={mod.enabled} on:change={(event) => setModEnabled(mod, event.currentTarget.checked)} />
                        <em>{mod.enabled ? "On" : "Off"}</em>
                      </label>
                    </div>
                    <details class="mod-advanced">
                      <summary>Advanced</summary>
                      <p>{mod.source_game_domain}/mods/{mod.source_mod_id}/files/{mod.source_file_id} · Priority {mod.priority}</p>
                      <div class="mod-advanced-actions">
                        <button type="button" class="secondary-action compact" on:click={() => setModPriority(mod, mod.priority - 1)}>Higher Priority</button>
                        <button type="button" class="secondary-action compact" on:click={() => setModPriority(mod, mod.priority + 1)}>Lower Priority</button>
                        <button type="button" class="secondary-action compact danger-action" on:click={() => askRemoveInstalledMod(mod)}>Remove</button>
                      </div>
                    </details>
                  </article>
                {/each}
              </div>
            </section>
          {/if}

          <details class="deploy-preview">
            <summary>
              <span>Advanced Deployment Tools</span>
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
              <div class="deploy-actions utility-actions">
                <button type="button" class="secondary-action" on:click={askApplyPendingProfileChanges} disabled={!deployPlan || deployableActions.length === 0 || hasDeployConflicts}>Apply Profile Now</button>
                <button type="button" class="secondary-action" on:click={previewDeploy} disabled={installedMods.length === 0 && !deploymentStatus?.deployed}>Preview Files</button>
                <button type="button" class="secondary-action" on:click={repairDeployment} disabled={!deploymentStatus?.repair_available}>Repair Managed Files</button>
                <button type="button" class="secondary-action" on:click={recoverDownloads}>Recover Downloads</button>
                <button type="button" class="secondary-action danger-action" on:click={askPurgeDeployment} disabled={!deploymentStatus?.purge_available}>Purge Managed Files</button>
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
                  <article class:failed-request={action.conflict}>
                    <strong>{action.target_relative}</strong>
                    <small>{action.conflict ? action.conflict_reason : `${action.operation || "add"} · ${action.strategy || "managed"}`}</small>
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
          {#if selectedGameRequests.length > 0}
            <button type="button" class="secondary-action" on:click={clearInstallRequests}>Clear Install Actions</button>
          {/if}
          {#if selectedGameActionItems.length === 0 && installCandidates.length === 0}
            <p class="hint">No install actions need attention for this game.</p>
          {/if}
          {#if selectedGameActionItems.length > 0}
            <div class="request-list">
              {#each selectedGameActionItems as request}
                <article class:failed-request={request.status === "failed"}>
                  <div>
                    <strong>{request.title}</strong>
                    {#if request.message}<p>{request.message}</p>{/if}
                    <p class="request-next-step">{requestNextStep(request)}</p>
                    <small>{new Date(request.updated_at).toLocaleString()}</small>
                  </div>
                    <div class="request-actions">
                      <span>{requestStatusLabel(request)}</span>
                      {#if request.type === "installer-choice"}
                        <button type="button" on:click={() => openActionItem(request)}>Open Choices</button>
                      {/if}
                      {#if request.type === "pending-import" && request.status === "waiting"}
                        <button type="button" on:click={() => approveInstallRequest(request)} disabled={isJobBusy(request)}>{isJobBusy(request) ? "Working..." : "Install"}</button>
                      {/if}
                      {#if request.type === "pending-import" && request.status === "failed"}
                        <button type="button" on:click={() => retryInstallRequest(request)} disabled={isJobBusy(request)}>{isJobBusy(request) ? "Working..." : "Retry"}</button>
                      {/if}
                      {#if canCancelJob(request)}
                        <button type="button" class="secondary-action compact" on:click={() => cancelJob(request)} disabled={isJobBusy(request)}>Cancel</button>
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
              <div class="request-list">
                {#each installCandidates as candidate}
                  {@const installer = installerForCandidate(candidate)}
                  <article class="failed-request">
                    <div>
                      <strong>{candidate.name}</strong>
                      <p>{candidate.reason}</p>
                      <small>{candidate.source_game_domain}/mods/{candidate.source_mod_id}/files/{candidate.source_file_id}</small>
                      {#if installer}
                        <div class="installer-choices">
                          {#each installer.steps ?? [] as step}
                            <section>
                              <h4>{step.name}</h4>
                              {#each step.groups ?? [] as group}
                                <fieldset>
                                  <legend>{group.name}</legend>
                                  {#each group.plugins ?? [] as plugin}
                                    <label>
                                      <input
                                        type={fomodGroupInputType(group)}
                                        name={`candidate-${candidate.id}-${group.id}`}
                                        checked={isCandidatePluginSelected(candidate, installer, group, plugin)}
                                        disabled={fomodGroupType(group) === "selectall" || isInstallCandidateBusy(candidate)}
                                        on:change={(event) => setCandidatePluginSelection(candidate, installer, group, plugin, event.currentTarget.checked)}
                                      />
                                      <span>
                                        <strong>{plugin.name}</strong>
                                        {#if plugin.type}<small>{plugin.type}</small>{/if}
                                        {#if plugin.description}<em>{plugin.description}</em>{/if}
                                      </span>
                                    </label>
                                  {/each}
                                </fieldset>
                              {/each}
                            </section>
                          {/each}
                        </div>
                      {/if}
                    </div>
                    <div class="request-actions">
                      <span>{candidateStatusLabel(candidate)}</span>
                      {#if installer}
                        <button type="button" on:click={() => applyInstallCandidate(candidate)} disabled={isInstallCandidateBusy(candidate)}>
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
          {:else if !gameDiagnostics?.runtime_requirements?.length && !visibleValidationWarnings.length}
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
      <p class="hint">Open the games menu to choose a Steam game before importing Nexus mods or managing profiles.</p>
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
