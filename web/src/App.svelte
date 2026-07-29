<script lang="ts">
  import { onMount } from "svelte";

  type Status = {
    game_count: number;
    install: { auto_install_captured_downloads: boolean; auto_enable_installed_mods: boolean };
    nexus: { api_key_configured: boolean };
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

  type Confirmation = {
    title: string;
    message: string;
    detail?: string;
    confirmLabel: string;
    danger?: boolean;
    run: () => Promise<void>;
  };

  type Drawer = "games" | "settings" | null;
  type Surface = "requests" | "game" | "settings";
  type GameModule = "plugins" | "requests" | "profiles" | "review" | "paths";
  type SettingsPage = "overview" | "jobs" | "install" | "nexus";
  type GameSort = "recent" | "az" | "za";

  const gamePreferencesKey = "dmm.gamePreferences.v1";

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
  let surface: Surface = "requests";
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
  $: selectedGameRequests = selectedGame ? installRequests.filter((job) => requestMatchesGame(job, selectedGame)) : installRequests;
  $: selectedGameActivity = selectedGame
    ? jobs.filter((job) => {
        if (job.type === "pending-import") return requestMatchesGame(job, selectedGame) && !["completed", "canceled"].includes(job.status);
        return ["installer-choice", "deploy", "purge", "repair", "recover-downloads"].includes(job.type) && jobMatchesGame(job, selectedGame) && !["completed", "canceled"].includes(job.status);
      })
    : [];
  $: filteredGames = sortDrawerGames(games.filter((game) => {
    const query = gameQuery.trim().toLowerCase();
    if (!query) return true;
    return game.name.toLowerCase().includes(query) || game.app_id.includes(query);
  }));
  $: title = surface === "settings" ? settingsTitle(activeSettingsPage) : surface === "requests" ? "Install Requests" : selectedGame?.name ?? "Select a Game";
  $: deployableActions = getDeployableActions(deployPlan);
  $: enabledMods = installedMods.filter((mod) => mod.enabled);
  $: disabledMods = installedMods.filter((mod) => !mod.enabled);
  $: deployAdds = deployableActions.filter((action) => action.operation === "add").length;
  $: deployReplaces = deployableActions.filter((action) => action.operation === "replace").length;
  $: deployRemoves = deployableActions.filter((action) => action.operation === "remove").length;
  $: hasPendingInstallRequests = selectedGameRequests.length > 0;
  $: hasDeployConflicts = (deployPlan?.conflicts.length ?? 0) > 0;
  $: stagedReady = installedMods.length > 0 && enabledMods.length > 0;
  $: previewReady = Boolean(deployPlan && !hasDeployConflicts);
  $: deployedReady = Boolean(deploymentStatus?.deployed && (deploymentStatus.file_count ?? 0) > 0);
  $: visibleValidationWarnings = displayValidationWarnings(gameDiagnostics);
  $: launchSetupAvailable = Boolean(gameLaunchStatus?.required && !gameLaunchStatus.configured && gameLaunchStatus.can_configure && gameLaunchStatus.action);

  async function getJSON<T>(url: string): Promise<T> {
    const response = await fetch(url);
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  async function refresh() {
    error = "";
    try {
      const [nextStatus, nextGames] = await Promise.all([
        getJSON<Status>("/api/status"),
        getJSON<Game[]>("/api/games")
      ]);
      status = nextStatus;
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

    const socket = new WebSocket(eventSocketURL());
    eventSocket = socket;
    socket.onopen = () => {
      eventReconnectDelay = 1000;
    };
    socket.onmessage = (message) => {
      if (typeof message.data !== "string") return;
      try {
        handleDomainEvent(JSON.parse(message.data) as DomainEvent);
      } catch (err) {
        error = err instanceof Error ? err.message : String(err);
      }
    };
    socket.onerror = () => {
      socket.close();
    };
    socket.onclose = () => {
      if (eventSocket !== socket) return;
      eventSocket = null;
      scheduleEventReconnect();
    };
  }

  function scheduleEventReconnect() {
    if (eventReconnectTimer !== null) return;
    const delay = eventReconnectDelay;
    eventReconnectDelay = Math.min(eventReconnectDelay * 2, 10000);
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
    if (event.type === "jobs.snapshot") {
      if (Array.isArray(event.payload)) jobs = event.payload as Job[];
      return;
    }
    if (event.type === "job.updated") {
      if (isJob(event.payload)) upsertJob(event.payload);
      return;
    }
    if (event.type === "launch.changed") {
      if (selectedGame && eventMatchesSelectedGame(event)) {
        if (isGameLaunchStatus(event.payload)) gameLaunchStatus = event.payload;
        scheduleSelectedGameRefresh(false);
      }
      return;
    }
    if (event.type === "game.changed") {
      void refresh();
      return;
    }
    if (["profile_mods.changed", "deployment.changed", "install.changed"].includes(event.type) && eventMatchesSelectedGame(event)) {
      scheduleSelectedGameRefresh(true);
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

  function scheduleSelectedGameRefresh(refreshPreview = false) {
    if (!selectedGame || !initialRefreshComplete) return;
    selectedGameRefreshNeedsPreview = selectedGameRefreshNeedsPreview || refreshPreview;
    if (selectedGameRefreshTimer !== null) return;
    selectedGameRefreshTimer = window.setTimeout(async () => {
      const shouldRefreshPreview = selectedGameRefreshNeedsPreview;
      selectedGameRefreshTimer = null;
      selectedGameRefreshNeedsPreview = false;
      await refreshSelectedGame({ refreshPreview: shouldRefreshPreview });
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

  function loadGamePreferences() {
    try {
      const raw = localStorage.getItem(gamePreferencesKey);
      if (!raw) return;
      const parsed = JSON.parse(raw) as { favorites?: unknown; recent?: unknown; sort?: unknown };
      if (Array.isArray(parsed.favorites)) {
        favoriteGameIDs = new Set(parsed.favorites.filter((item): item is string => typeof item === "string" && item.trim() !== ""));
      }
      if (parsed.recent && typeof parsed.recent === "object") {
        const entries = Object.entries(parsed.recent as Record<string, unknown>)
          .map(([appID, value]) => [appID, Number(value)] as const)
          .filter(([appID, value]) => appID.trim() !== "" && Number.isFinite(value));
        gameRecent = Object.fromEntries(entries);
      }
      if (parsed.sort === "recent" || parsed.sort === "az" || parsed.sort === "za") {
        gameSort = parsed.sort;
      }
    } catch (_err) {
      favoriteGameIDs = new Set();
      gameRecent = {};
      gameSort = "recent";
    }
  }

  function saveGamePreferences() {
    try {
      localStorage.setItem(gamePreferencesKey, JSON.stringify({
        favorites: Array.from(favoriteGameIDs),
        recent: gameRecent,
        sort: gameSort
      }));
    } catch (_err) {
      // Browser storage can be unavailable in private contexts; the drawer still works without persistence.
    }
  }

  function setGameSort(nextSort: GameSort) {
    gameSort = nextSort;
    saveGamePreferences();
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
    saveGamePreferences();
  }

  function markGameRecent(appID: string) {
    const next = { ...gameRecent, [appID]: Date.now() };
    gameRecent = Object.fromEntries(Object.entries(next).sort((a, b) => b[1] - a[1]).slice(0, 50));
    saveGamePreferences();
  }

  function openSettings(page: SettingsPage) {
    activeSettingsPage = page;
    surface = "settings";
    drawer = null;
  }

  function openRequests() {
    surface = "requests";
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
    await loadProfiles(selectedGame);
    await loadInstalledMods(selectedGame);
    await applyCurrentProfileChanges();
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
    const updated = await response.json();
    installedMods = installedMods.map((item) => (item.id === updated.id ? updated : item));
    await applyCurrentProfileChanges();
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
    const updated = await response.json();
    installedMods = installedMods
      .map((item) => (item.id === updated.id ? updated : item))
      .sort((a, b) => a.priority - b.priority || a.name.localeCompare(b.name));
    await applyCurrentProfileChanges();
  }

  async function removeInstalledMod(mod: InstalledMod) {
    if (!selectedGame) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/mods/${mod.id}`, { method: "DELETE" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    installedMods = installedMods.filter((item) => item.id !== mod.id);
    await refreshSelectedGame({ refreshPreview: true });
    await applyCurrentProfileChanges();
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
    return candidateSelections[candidate.id] ?? defaultCandidateSelections(installer);
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
      candidateSelections = Object.fromEntries(Object.entries(candidateSelections).filter(([id]) => Number(id) !== candidate.id));
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
    try {
      const response = await fetch(`/api/jobs/${job.id}/cancel`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
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
    try {
      const response = await fetch(`/api/imports/pending/${request.id}/approve`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
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
    try {
      const response = await fetch(`/api/imports/pending/${request.id}/retry`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
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

  async function deployStagedMods() {
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

  async function applyCurrentProfileChanges() {
    if (!selectedGame) return false;
    error = "";
    const previewResponse = await fetch(`/api/games/${selectedGame.app_id}/deploy/preview`);
    if (!previewResponse.ok) {
      error = await previewResponse.text();
      return false;
    }
    const nextPlan: DeployPlan = await previewResponse.json();
    deployPlan = nextPlan;
    const actions = getDeployableActions(nextPlan);
    if (nextPlan.conflicts.length > 0 || actions.length === 0) {
      await refreshSelectedGame();
      return false;
    }
    const response = await fetch(`/api/games/${selectedGame.app_id}/deploy`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return false;
    }
    const result = await response.json();
    upsertJob(result.job);
    deployPlan = result.plan;
    await refreshSelectedGame({ refreshPreview: true });
    return true;
  }

  async function askDeployStagedMods() {
    if (!selectedGame) return;
    const plan = await ensureDeployPlan();
    if (!plan) return;
    const actions = getDeployableActions(plan);
    if (plan.conflicts.length > 0 || actions.length === 0) return;
    const adds = actions.filter((action) => action.operation === "add").length;
    const replaces = actions.filter((action) => action.operation === "replace").length;
    const removes = actions.filter((action) => action.operation === "remove").length;
    confirmation = {
      title: "Apply profile changes",
      message: `DMM will update ${selectedGame.name}'s game folder to match the selected profile.`,
      detail: `${adds} add, ${replaces} replace, ${removes} remove. Advanced file details remain available before or after applying.`,
      confirmLabel: "Apply Changes",
      run: deployStagedMods
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
    if (!changed || !selectedGame || !jobMatchesGame(job, selectedGame)) return;
    if (["pending-import", "installer-choice", "deploy", "purge", "repair", "recover-downloads"].includes(job.type)) {
      scheduleSelectedGameRefresh(job.status === "completed" || deployPlan !== null);
    }
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
    if (mod.status === "needs_recovery") return "Needs recovery before it can be applied";
    return mod.status;
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
    if (request.status === "waiting") {
      return "Approve install to add this downloaded mod to the selected profile.";
    }
    if (request.status === "running" || request.status === "queued") {
      return "DMM is downloading or installing this mod. It will appear in the profile when ready.";
    }
    if (request.status === "failed") {
      return "The mod was not added. Retry the request if the download link is still valid, or clear it after saving anything useful.";
    }
    return "This request is retained in job history for diagnostics.";
  }

  function requestStatusLabel(request: Job) {
    if (request.status === "waiting") return "Needs approval";
    if (request.status === "running") return "Processing";
    if (request.status === "queued") return "Queued";
    if (request.status === "failed") return "Failed";
    return request.status;
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
    loadGamePreferences();
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
          <button type="button" on:click={openRequests}>Install Requests</button>
          <button type="button" class:active={activeSettingsPage === "jobs"} on:click={() => openSettings("jobs")}>Jobs</button>
          <button type="button" class:active={activeSettingsPage === "install"} on:click={() => openSettings("install")}>Install</button>
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
  {:else if surface === "requests"}
    <section class="settings-screen">
      <article class="workspace-panel">
        <div class="panel-heading">
          <h2>Install Requests</h2>
          <span>{installRequests.length} pending</span>
        </div>
        {#if installRequests.length > 0}
          <button type="button" class="secondary-action" on:click={clearInstallRequests}>Clear Requests</button>
        {/if}
        {#if installRequests.length === 0}
          <div class="request-home">
            <div class="empty-state inline-empty">
              <h2>No Install Requests</h2>
              <p class="hint">Open a game to paste a Nexus URL, or capture an nxm:// link from the Decky plugin browser flow.</p>
            </div>
            <button type="button" on:click={() => (drawer = "games")}>Choose Game</button>
          </div>
        {:else}
          <div class="request-list">
            {#each installRequests as request}
              <article class:failed-request={request.status === "failed"}>
                <div>
                  <strong>{request.title}</strong>
	                    {#if request.message}<p>{request.message}</p>{/if}
	                    <p class="request-next-step">{requestNextStep(request)}</p>
	                    <small>{new Date(request.updated_at).toLocaleString()}</small>
	                  </div>
	                  <div class="request-actions">
	                    <span>{requestStatusLabel(request)}</span>
	                    {#if request.status === "waiting"}
	                      <button type="button" on:click={() => approveInstallRequest(request)} disabled={isJobBusy(request)}>{isJobBusy(request) ? "Working..." : "Approve Install"}</button>
	                    {/if}
	                    {#if request.status === "failed"}
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
            <div><dt>Auto install</dt><dd>{status?.install.auto_install_captured_downloads ? "Enabled" : "Approval required"}</dd></div>
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
          <h2>Install</h2>
          <dl class="settings-list">
            <div><dt>Auto install captured downloads</dt><dd>{status?.install.auto_install_captured_downloads ? "Enabled" : "Approval required"}</dd></div>
            <div><dt>Auto enable installed mods</dt><dd>{status?.install.auto_enable_installed_mods ? "Enabled" : "Disabled"}</dd></div>
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
        <button type="button" class:active={activeGameModule === "plugins"} on:click={() => (activeGameModule = "plugins")}>Plugins</button>
        <button type="button" class:active={activeGameModule === "requests"} on:click={() => (activeGameModule = "requests")}>Requests</button>
        <button type="button" class:active={activeGameModule === "profiles"} on:click={() => (activeGameModule = "profiles")}>Profiles</button>
        <button type="button" class:active={activeGameModule === "review"} on:click={() => (activeGameModule = "review")}>Review</button>
        <button type="button" class:active={activeGameModule === "paths"} on:click={() => (activeGameModule = "paths")}>Paths</button>
      </nav>

      {#if activeGameModule === "plugins"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Mod Management</h2>
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
                <div><strong>{selectedGameRequests.length}</strong><span>Requests</span></div>
              </div>
              {#if deployPlan && (deployableActions.length > 0 || deployPlan.conflicts.length > 0)}
                <div class="profile-change-summary" class:has-conflicts={deployPlan.conflicts.length > 0}>
                  <div><strong>{deployAdds}</strong><span>Add</span></div>
                  <div><strong>{deployReplaces}</strong><span>Update</span></div>
                  <div><strong>{deployRemoves}</strong><span>Remove</span></div>
                  <div><strong>{deployPlan.conflicts.length}</strong><span>Conflict</span></div>
                </div>
              {/if}
              {#if deploymentStatus?.deployed && deployPlan && deployPlan.conflicts.length === 0 && deployableActions.length === 0}
                <p class="deploy-message success">
                  Profile is applied to the game with {deploymentStatus.file_count} managed file{deploymentStatus.file_count === 1 ? "" : "s"}.
                </p>
              {:else if deploymentStatus?.deployed && !deployPlan}
                <p class="deploy-message">A profile is deployed. Refresh status to check for pending profile changes.</p>
              {:else}
                <p class="deploy-message">Profile changes have not been applied to the game yet.</p>
              {/if}
	              {#if deployPlan}
	                {#if deployPlan.conflicts.length > 0}
	                  <p class="deploy-message danger">Conflicts need attention before profile changes can be applied.</p>
	                {:else if deployableActions.length === 0}
	                  <p class="deploy-message">This profile is already applied.</p>
	                {:else}
	                  <p class="deploy-message">{deployAdds + deployReplaces + deployRemoves} pending profile change{deployAdds + deployReplaces + deployRemoves === 1 ? "" : "s"} ready to apply.</p>
	                {/if}
	              {:else}
	                <p class="deploy-message">Enable or disable mods to apply the selected profile to the game.</p>
	              {/if}
	              <div class="deploy-actions primary-actions profile-actions">
	                {#if deployPlan && deployableActions.length > 0 && !hasDeployConflicts}
	                  <button type="button" on:click={askDeployStagedMods}>Apply Changes</button>
	                {/if}
	                <button type="button" class="secondary-action" on:click={previewDeploy} disabled={installedMods.length === 0 && !deploymentStatus?.deployed}>Check Profile Changes</button>
	              </div>
            </div>

            <div class="management-card import-card">
              <div class="card-heading">
                <h3>Add From Nexus</h3>
                <span>{selectedGameRequests.length} pending</span>
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
            <p class="hint">No profile mods yet. Approve an install request to add a supported Nexus mod to this profile.</p>
          {:else}
            <section class="mod-section">
              <div class="card-heading">
                <h3>Profile Mods</h3>
                <span>{selectedProfile?.name ?? "Default"}</span>
              </div>
              <div class="mod-list">
                {#each installedMods as mod}
                  {@const metadata = primaryModMetadata(mod)}
                  {@const dependencyLabels = modDependencyLabels(mod)}
                  <article>
                    <div>
                      <strong>{mod.name}</strong>
                      <p>{mod.source_game_domain}/mods/{mod.source_mod_id}/files/{mod.source_file_id}</p>
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
                      <small>{mod.enabled ? "Enabled in this profile" : "Disabled in this profile"} · Priority {mod.priority} · {modStatusText(mod)}</small>
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
              <small>{deploymentStatus?.deployed ? `${deploymentStatus.file_count} applied` : "Not applied"}</small>
            </summary>
            <div class="deployment-summary">
              <div><strong>{enabledMods.length}</strong><span>Enabled</span></div>
              <div><strong>{deploymentStatus?.file_count ?? 0}</strong><span>Applied</span></div>
              <div><strong>{deployPlan?.conflicts.length ?? 0}</strong><span>Conflicts</span></div>
	            </div>
	            <div class="deploy-actions utility-actions">
	              <button type="button" class="secondary-action" on:click={askDeployStagedMods} disabled={!deployPlan || deployableActions.length === 0 || hasDeployConflicts}>Apply Pending Changes</button>
	              <button type="button" class="secondary-action" on:click={previewDeploy} disabled={installedMods.length === 0 && !deploymentStatus?.deployed}>Preview Files</button>
	              <button type="button" class="secondary-action" on:click={repairDeployment} disabled={!deploymentStatus?.deployed}>Repair Managed Files</button>
	              <button type="button" class="secondary-action" on:click={askPurgeDeployment} disabled={!deploymentStatus?.deployed}>Purge Managed Files</button>
              <button type="button" class="secondary-action" on:click={recoverDownloads}>Recover Downloads</button>
            </div>
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
      {:else if activeGameModule === "requests"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Install Requests</h2>
            <span>{selectedGameRequests.length} pending · {installCandidates.length} blocked</span>
          </div>
          {#if selectedGameRequests.length > 0}
            <button type="button" class="secondary-action" on:click={clearInstallRequests}>Clear Requests</button>
          {/if}
          {#if selectedGameRequests.length === 0 && installCandidates.length === 0}
            <p class="hint">No install requests or blocked install candidates matched this game.</p>
          {/if}
          {#if selectedGameRequests.length > 0}
            <div class="request-list">
              {#each selectedGameRequests as request}
                <article class:failed-request={request.status === "failed"}>
                  <div>
                    <strong>{request.title}</strong>
                    {#if request.message}<p>{request.message}</p>{/if}
                    <p class="request-next-step">{requestNextStep(request)}</p>
                    <small>{new Date(request.updated_at).toLocaleString()}</small>
                  </div>
                    <div class="request-actions">
                      <span>{requestStatusLabel(request)}</span>
                      {#if request.status === "waiting"}
                        <button type="button" on:click={() => approveInstallRequest(request)} disabled={isJobBusy(request)}>{isJobBusy(request) ? "Working..." : "Approve Install"}</button>
                      {/if}
                      {#if request.status === "failed"}
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
                <h3>Blocked Install Plans</h3>
                <span>{installCandidates.length}</span>
              </div>
              <button type="button" class="secondary-action" on:click={clearBlockedInstallCandidates}>Clear Blocked</button>
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
                      <span>{candidate.status}</span>
                      {#if installer}
                        <button type="button" on:click={() => applyInstallCandidate(candidate)} disabled={isInstallCandidateBusy(candidate)}>
                          {isInstallCandidateBusy(candidate) ? "Applying..." : "Apply Choices"}
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
