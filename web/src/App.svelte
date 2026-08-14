<script lang="ts">
  import { onMount } from "svelte";

  const authStorageKey = "decky-mod-manager:api-token";
  let apiAuthToken = "";

  type UISettings = {
    favorite_game_ids?: string[];
    recent_games?: Record<string, number>;
    game_sort?: GameSort;
  };

  type Status = {
    game_count: number;
    auth?: { enabled: boolean };
    install: { auto_install_captured_downloads: boolean; auto_enable_installed_mods: boolean; auto_show_fomod_installers?: boolean };
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

  type ExtensionSetting = {
    extension_id: string;
    setting_id: string;
    name: string;
    scope?: string;
    value_type?: "json" | "string" | "path" | "bool" | "number" | string;
    placeholder?: string;
    status: string;
    message?: string;
    value: unknown;
    options?: ExtensionSettingOption[];
    options_error?: string;
    updated_at?: string;
  };

  type ExtensionSettingOption = {
    id: string;
    label: string;
    description?: string;
    disabled?: boolean;
  };

  type Game = {
    app_id: string;
    name: string;
    store?: string;
    store_app_id?: string;
    path: string;
    library_path: string;
    state: string;
    markers?: string[];
    steam_workshop?: SteamWorkshop;
    nexus_domains?: string[];
    extension?: GameExtensionInfo;
  };

  type GameExtensionInfo = {
    id: string;
    name: string;
    supported: boolean;
    coverage?: string;
    coverage_label?: string;
    nexus: boolean;
    steam_workshop: boolean;
    installers: boolean;
    installer_choices: boolean;
    runtime_requirements: boolean;
    launch_tools: boolean;
    plugin_activation: boolean;
    load_order: boolean;
    game_versions: boolean;
    catalog_sources?: CatalogSourceHint[];
    sources?: SourceRef[];
  };

  type CatalogSourceHint = {
    catalog: string;
    game_id: string;
    domain?: string;
    url?: string;
  };

  type SourceRef = {
    name?: string;
    url?: string;
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
    mod_count: number;
    enabled_mod_count: number;
  };

  type ProfileFeature = {
    profile_id: number;
    feature_id: string;
    name: string;
    enabled: boolean;
    status: string;
    message?: string;
    extension?: string;
  };

  type ProfileFile = {
    profile_id: number;
    file_id: string;
    name: string;
    game_id: string;
    base?: string;
    path?: string;
    feature_id?: string;
    optional?: boolean;
    sync_on_profile_switch?: boolean;
    resolved_path?: string;
    status: string;
    message?: string;
    extension?: string;
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

  type NexusSearchSort = "downloads" | "unique_downloads" | "popular" | "updated" | "name" | "relevance";
  type NexusTimeWindow = "all" | "one_week" | "three_weeks" | "one_month" | "three_months" | "one_year";

  type CatalogModResult = {
    catalog?: string;
    source_tag?: string;
    mod_id: string | number;
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

  type ModUpdateBrowserPrompt = {
    url: string;
    mod_id: string | number;
    mod_name: string;
    title: string;
  };

  type DeckBrowserPrompt = {
    url: string;
    source: string;
    title: string;
    message: string;
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
    target_profile_id?: number;
  };

  type LocalArchiveFile = {
    path: string;
    name: string;
    extension: string;
    bytes: number;
    root: string;
    modified_at: string;
  };

  type LocalArchiveList = {
    roots: string[];
    files: LocalArchiveFile[];
  };

  type LocalArchiveBrowseEntry = {
    path: string;
    name: string;
    kind: "directory" | "file";
    extension?: string;
    bytes?: number;
    root?: string;
    modified_at?: string;
    supported?: boolean;
  };

  type LocalArchiveBrowseList = {
    roots: string[];
    current_path: string;
    parent_path?: string;
    entries: LocalArchiveBrowseEntry[];
  };

  type ExternalModCandidate = {
    adoption_id: string;
    name: string;
    path: string;
    relative_path: string;
    root_path: string;
    mod_type: string;
    size: number;
    sha256: string;
    delete_original: boolean;
  };

  type ExternalModList = {
    items: ExternalModCandidate[];
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
    description?: string;
    placeholder?: string;
    required?: boolean;
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

  type DeployPreviewSummary = {
    available: boolean;
    add: number;
    replace: number;
    remove: number;
    keep: number;
    skip: number;
    conflicts: number;
    error?: string;
  };

  type DeploymentRestorePreview = {
    deployment_id: number;
    current_file_count: number;
    target_file_count: number;
    summary: DeployPreviewSummary;
    sample_files?: string[];
    plan: DeployPlan;
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
    active: boolean;
    strategy: string;
    file_count: number;
    sources?: DeploymentSourceSummary[];
    created_at: string;
    updated_at: string;
  };

  type DeploymentSourceSummary = {
    catalog: string;
    source_tag?: string;
    file_count: number;
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

  type BulkCapturedInstallItem = {
    index: number;
    url: string;
    ok: boolean;
    error?: string;
    job?: Job;
    resolved?: {
      catalog: string;
      game_domain?: string;
      steam_app_id?: string;
      mod_id?: string;
      file_id?: string;
    };
    browser_required?: boolean;
    duplicate?: boolean;
  };

  type BulkCapturedInstallResult = {
    total: number;
    accepted: number;
    failed: number;
    browser_required: number;
    items: BulkCapturedInstallItem[];
  };

  type PluginLoadOrder = {
    app_id: string;
    profile_id?: number;
    supported: boolean;
    activation_id?: string;
    name?: string;
    target_root?: string;
    plugins_file?: string;
    load_order_file?: string;
    loot?: LOOTStatus;
    plugins: PluginLoadOrderEntry[];
    extension_orders?: ExtensionLoadOrder[];
    warnings?: string[];
  };

  type LOOTStatus = {
    supported: boolean;
    revision?: string;
    profile_id?: number;
    game_id?: string;
    masterlist_game_id?: string;
    sorter_status?: string;
    sorter_message?: string;
    sorter_engine?: string;
    sorter_command?: string;
    sorter_available: boolean;
    masterlist?: LOOTFileStatus;
    userlist?: LOOTFileStatus;
    userlist_rules?: LOOTRuleSummary;
    userlist_warning?: string;
    prelude?: LOOTFileStatus;
    last_refresh_warning?: string;
  };

  type LOOTRuleSummary = {
    plugins: number;
    rules: number;
    groups: number;
    group_rules: number;
  };

  type LOOTUserlist = {
    globals: Record<string, unknown>[];
    plugins: LOOTUserlistPlugin[];
    groups: LOOTUserlistGroup[];
  };

  type LOOTUserlistPlugin = {
    name: string;
    group?: string;
    after?: string[];
    requires?: string[];
    incompatible?: string[];
  };

  type LOOTUserlistGroup = {
    name: string;
    after?: string[];
  };

  type LOOTFileStatus = {
    path?: string;
    url?: string;
    exists: boolean;
    size_bytes?: number;
    updated_at?: string;
  };

  type PluginLoadOrderEntry = {
    name: string;
    source: string;
    catalog?: string;
    installed_mod_id?: number;
    mod_id?: string;
    priority: number;
    active: boolean;
    mutable?: boolean;
  };

  type ExtensionLoadOrder = {
    id: string;
    name?: string;
    target_relative?: string;
    target_root?: string;
    target_root_id?: string;
    status?: string;
    message?: string;
    entry_name_mode?: string;
    toggleable_entries?: boolean;
    usage_instructions?: string;
    mutable?: boolean;
    entries: ExtensionLoadOrderEntry[];
  };

  type ExtensionLoadOrderEntry = {
    id: string;
    name: string;
    installed_mod_id?: number;
    mod_id?: string;
    catalog?: string;
    source_tag?: string;
    mod_type?: string;
    priority: number;
    active: boolean;
    mutable?: boolean;
    targets?: string[];
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
    acquisition?: RuntimeAcquisition;
  };

  type RuntimeAcquisition = {
    id?: string;
    name?: string;
    catalog?: string;
    url?: string;
    archive_name?: string;
    required?: boolean;
    auto_acquire?: boolean;
    source_mod_id?: string;
    source_file_id?: string;
    source_game?: string;
    source_provider?: string;
    message?: string;
  };

  type LauncherRequirement = {
    id: string;
    name: string;
    launcher: string;
    store?: string;
    app_id?: string;
    status: string;
    required: boolean;
    satisfied: boolean;
    message?: string;
    details?: string[];
    parameters?: { name: string; value: string }[];
    source_extension: string;
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
      executable_relative?: string;
      executable_path: string;
      arguments?: string[];
      required_files?: string[];
      shell?: boolean;
      detach?: boolean;
      exclusive?: boolean;
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
    launcher_requirements?: LauncherRequirement[];
    validation_warnings?: string[];
  };

  type GameInfo = {
    app_id: string;
    name: string;
    ran: boolean;
    details: GameInfoDetail[];
  };

  type GameInfoDetail = {
    id: string;
    title: string;
    type?: string;
    value: unknown;
    source?: string;
  };

  type GameExtensionAction = {
    id: string;
    name?: string;
    scope?: string;
    kind?: string;
    status?: string;
    message?: string;
    source_extension?: string;
    action_target?: GameExtensionActionTarget;
  };

  type GameExtensionActionTarget = {
    type?: string;
    scope?: string;
    base?: string;
    target_root_id?: string;
    relative_path?: string;
    fallback_base?: string;
    fallback_root_id?: string;
    fallback_relative?: string;
    tool_id?: string;
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
    cascade?: InstalledMod[];
    cascade_notes?: string[];
    apply: ProfileApplyResult;
  };

  type ProfileModOrderUpdateResult = {
    mods: InstalledMod[];
    apply: ProfileApplyResult;
  };

  type ProfilePluginActivationState = {
    profile_id: number;
    activation_id: string;
    plugin_name: string;
    plugin_key: string;
    enabled: boolean;
    priority: number;
  };

  type ProfilePluginActivationUpdateResult = {
    load_order: PluginLoadOrder;
    state?: ProfilePluginActivationState;
    states?: ProfilePluginActivationState[];
    apply: ProfileApplyResult;
  };

  type LOOTSortResult = {
    load_order: PluginLoadOrder;
    states?: ProfilePluginActivationState[];
    sorted_plugins: string[];
    warnings?: string[];
    engine?: string;
    apply: ProfileApplyResult;
  };

  type SetDefaultProfileResult = {
    profile: Profile;
    apply: ProfileApplyResult;
  };

  type DeleteProfileResult = {
    deleted?: Profile;
    active_profile: Profile;
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
  type SettingsPage = "overview" | "jobs" | "install" | "sources" | "game-stores" | "extensions" | "nexus";
  type GameSort = "recent" | "az" | "za";
  type GameVisibility = "manageable" | "extensions" | "all";
  type ModListSort = "profile" | "source" | "az" | "enabled";

  let status: Status | null = null;
  let catalogs: CatalogStatus[] = [];
  let extensionSettings: ExtensionSetting[] = [];
  let games: Game[] = [];
  let jobs: Job[] = [];
  let selectedGame: Game | null = null;
  let profiles: Profile[] = [];
  let profileFeatures: ProfileFeature[] = [];
  let profileFiles: ProfileFile[] = [];
  let profileExtensionSettings: ExtensionSetting[] = [];
  let installedMods: InstalledMod[] = [];
  let installCandidates: InstallCandidate[] = [];
  let installerChoicePresets: InstallerChoicePreset[] = [];
  let profileName = "";
  let copyProfileFromActive = false;
  let captureURL = "";
  let resolvedCapture = "";
  let bulkCaptureMessage = "";
  let extensionSurfaceMessage = "";
  let extensionSurfaceBusyID = "";
  let captureBrowserPrompt: DeckBrowserPrompt | null = null;
  let captureBrowserOpenBusy = false;
  let nexusSearchQuery = "";
  let nexusSearchSort: NexusSearchSort = "updated";
  let nexusSearchTimeWindow: NexusTimeWindow = "all";
  let nexusSearchVortexOnly = true;
  let nexusSearchResults: CatalogModResult[] = [];
  let nexusSearchTotal = 0;
  let nexusSearchBusy = false;
  let nexusSearchError = "";
  let nexusSearchMessage = "";
  let nexusBrowseDomain = "";
  let busyCatalogOpenModID: string | number | null = null;
  let exploreSourceID = "nexus";
  let localArchiveFile: File | null = null;
  let localArchiveInput: HTMLInputElement | null = null;
  let localArchiveBusy = false;
  let localArchiveMessage = "";
  let deckLocalArchiveRoots: string[] = [];
  let deckLocalArchives: LocalArchiveFile[] = [];
  let deckArchiveBrowserOpen = false;
  let deckArchiveBrowserEntries: LocalArchiveBrowseEntry[] = [];
  let deckArchiveBrowserPath = "";
  let deckArchiveBrowserParentPath = "";
  let deckArchivePathInput = "";
  let deckLocalArchiveBusy = false;
  let deckLocalArchiveMessage = "";
  let busyDeckLocalArchivePath = "";
  let externalModAdoptionOpen = false;
  let externalModCandidates: ExternalModCandidate[] = [];
  let selectedExternalModPaths: Record<string, boolean> = {};
  let externalModAdoptionBusy = false;
  let externalModAdoptionMessage = "";
  let deployPlan: DeployPlan | null = null;
  let deploymentStatus: DeploymentStatus | null = null;
  let deploymentSettings: DeploymentSettings | null = null;
  let deploymentHistory: DeploymentHistoryItem[] = [];
  let restorePointPreview: DeploymentRestorePreview | null = null;
  let restorePointPreviewBusy = 0;
  let pluginLoadOrder: PluginLoadOrder | null = null;
  let lootUserlist: LOOTUserlist | null = null;
  let lootRulePlugin = "";
  let lootRuleReference = "";
  let lootRuleType: "after" | "requires" | "incompatible" = "after";
  let lootGroupName = "";
  let lootGroupReference = "";
  let lootGroupPlugin = "";
  let lootGroupAssignment = "";
  let lootUserlistBusy = false;
  let lootUserlistMessage = "";
  let gameDiagnostics: GameDiagnostics | null = null;
  let gameInfo: GameInfo | null = null;
  let gameExtensionActions: GameExtensionAction[] = [];
  let gameLaunchStatus: GameLaunchStatus | null = null;
  let workshopState: WorkshopState | null = null;
  let workshopItems: WorkshopItem[] = [];
  let globalInstallCandidates: InstallCandidate[] = [];
  let loading = true;
  let error = "";
  let authRejected = false;
  let drawer: Drawer = null;
  let confirmation: Confirmation | null = null;
  let surface: Surface = "actions";
  let activeGameModule: GameModule = "plugins";
  let activeSettingsPage: SettingsPage = "overview";
  let gameQuery = "";
  let gameSort: GameSort = "recent";
  let gameVisibility: GameVisibility = "manageable";
  let manualGameStore = "gog";
  let manualGameStoreAppID = "";
  let manualGameName = "";
  let manualGamePath = "";
  let manualGameBusy = false;
  let manualGameMessage = "";
  let actionSourceFilter = "all";
  let jobSourceFilter = "all";
  let modSourceFilter = "all";
  let modListSort: ModListSort = "profile";
  let favoriteGameIDs = new Set<string>();
  let gameRecent: Record<string, number> = {};
  let busyJobs: Record<string, boolean> = {};
  let busyInstallCandidates: Record<number, boolean> = {};
  let busyWorkshopActions: Record<string, boolean> = {};
  let busyPluginActivationRows: Record<string, boolean> = {};
  let busyProfileFeatures: Record<string, boolean> = {};
  let profileFeatureMessage = "";
  let lootRefreshBusy = false;
  let workshopOrderBusy = false;
  type ModBusyAction = "toggle" | "remove" | "reinstall" | "reconfigure" | "update" | "copy" | "move";

  let busyMods: Record<number, ModBusyAction> = {};
  let modUpdateBusy = false;
  let modUpdateMessage = "";
  let modUpdateBrowserPrompt: ModUpdateBrowserPrompt | null = null;
  let modUpdateBrowserOpenBusy = false;
  let lootSortBusy = false;
  let modIOAPIKey = "";
  let curseForgeAPIKey = "";
  let nexusAPIKey = "";
  let nexusSettingsBusy = false;
  let nexusSettingsMessage = "";
  let catalogSettingsBusy = "";
  let catalogSettingsMessage = "";
  let extensionSettingDrafts: Record<string, string> = {};
  let extensionSettingBusy = "";
  let extensionSettingsMessage = "";
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
  let installTargetProfileID = "";
  let actionProfileTargets: Record<string, string> = {};
  let candidateProfileTargets: Record<number, string> = {};
  let profileTransferTargets: Record<number, string> = {};
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
  let selectedGameLoadSequence = 0;
  let fullRefreshSequence = 0;

  function setBusyMod(modID: number, action: ModBusyAction) {
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
  $: enabledProfileFeatureIDs = new Set(profileFeatures.filter((feature) => feature.enabled).map((feature) => feature.feature_id.toLowerCase()));
  $: visibleProfileFiles = profileFiles.filter((file) => file.feature_id && enabledProfileFeatureIDs.has(file.feature_id.toLowerCase()));
  $: {
    const options = exploreSourceOptions();
    if (options.length > 0 && !options.some((catalog) => catalog.id === exploreSourceID)) {
      exploreSourceID = options.find((catalog) => catalog.id === "nexus")?.id ?? options[0].id;
    }
  }
  $: if (profiles.length > 0 && !profiles.some((profile) => String(profile.id) === installTargetProfileID)) {
    installTargetProfileID = String(selectedProfile?.id ?? profiles[0].id);
  }
  $: capturedInstallActions = jobs.filter((job) => job.type === "captured-install" && !["completed", "canceled"].includes(job.status));
  $: actionItems = jobs.filter((job) =>
    ["captured-install", "installer-choice", "steam-workshop-action", "extension-notice", "extension-tool-action"].includes(job.type) &&
    !["completed", "canceled"].includes(job.status) &&
    !jobHasInstallCandidateReview(job) &&
    !installerChoiceJobHasCandidateReview(job)
  );
  $: actionCenterCandidates = globalInstallCandidates;
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
        return ["installer-choice", "steam-workshop-action", "extension-notice", "extension-tool-action", "deploy", "purge", "repair", "recover-downloads", "rollback"].includes(job.type) && jobMatchesGame(job, selectedGame) && !["completed", "canceled"].includes(job.status);
      })
    : [];
  $: manageableGameCount = games.filter(gameManageReady).length;
  $: extensionGameCount = games.filter(gameHasExtension).length;
  $: filteredGames = sortDrawerGames(games.filter((game) => {
    if (gameVisibility === "manageable" && !gameManageReady(game)) return false;
    if (gameVisibility === "extensions" && !gameHasExtension(game)) return false;
    const query = gameQuery.trim().toLowerCase();
    if (!query) return true;
    return game.name.toLowerCase().includes(query) || game.app_id.includes(query);
  }));
  $: homeQuickGames = sortDrawerGames(games.filter(gameManageReady)).slice(0, 6);
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

  function initializeAPIAuth() {
    const hashParams = new URLSearchParams(window.location.hash.replace(/^#/, ""));
    const queryParams = new URLSearchParams(window.location.search);
    const token = (hashParams.get("token") || hashParams.get("dmm_token") || queryParams.get("token") || queryParams.get("dmm_token") || "").trim();
    if (token) {
      apiAuthToken = token;
      try {
        window.localStorage.setItem(authStorageKey, token);
      } catch (_err) {
        // The in-memory token still carries the current session if browser storage is unavailable.
      }
      hashParams.delete("token");
      hashParams.delete("dmm_token");
      queryParams.delete("token");
      queryParams.delete("dmm_token");
      const search = queryParams.toString();
      const hash = hashParams.toString();
      window.history.replaceState(null, "", `${window.location.pathname}${search ? `?${search}` : ""}${hash ? `#${hash}` : ""}`);
      return;
    }
    try {
      apiAuthToken = window.localStorage.getItem(authStorageKey) ?? "";
    } catch (_err) {
      apiAuthToken = "";
    }
  }

  function apiHeaders(headers?: HeadersInit): Headers {
    const next = new Headers(headers ?? {});
    if (apiAuthToken && !next.has("X-DMM-Token")) {
      next.set("X-DMM-Token", apiAuthToken);
    }
    return next;
  }

  async function apiFetch(input: RequestInfo | URL, init: RequestInit = {}): Promise<Response> {
    const response = await fetch(input, { ...init, headers: apiHeaders(init.headers) });
    if (response.status === 401) {
      authRejected = true;
      loading = false;
      closeEventSocket();
    } else if (response.ok && authRejected) {
      authRejected = false;
    }
    return response;
  }

  async function getJSON<T>(url: string): Promise<T> {
    const response = await apiFetch(url);
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  function logClientEvent(message: string, detail: ClientEventDetail = {}) {
    const body = JSON.stringify({ message, detail });
    apiFetch("/api/client-events", {
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
      detail.job_app_id = jobAppID(event.payload);
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
      const [nextStatus, nextGames, nextCatalogs, nextExtensionSettings] = await Promise.all([
        getJSON<Status>("/api/status"),
        getJSON<Game[]>("/api/games"),
        getJSON<CatalogStatus[]>("/api/catalogs"),
        getJSON<ExtensionSetting[]>("/api/extensions/settings")
      ]);
      if (sequence !== fullRefreshSequence) {
        logClientEvent("full refresh discarded stale response", { sequence, latest_sequence: fullRefreshSequence, reason });
        return;
      }
      status = nextStatus;
      applyUIPreferences(nextStatus);
      games = nextGames;
      catalogs = nextCatalogs;
      setExtensionSettings(nextExtensionSettings);
      const previousSelection = selectedGame?.app_id;
      selectedGame = nextGames.find((game) => game.app_id === previousSelection) ?? null;
      if (selectedGame) await loadGameState(selectedGame);
      logClientEvent("full refresh completed", {
        sequence,
        reason,
        games: nextGames.length,
        catalogs: nextCatalogs.length,
        extension_settings: nextExtensionSettings.length,
        selected_app_id: selectedGame?.app_id ?? ""
      });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      logClientEvent("full refresh failed", { sequence, reason, error: compactLogValue(error) });
    } finally {
      if (sequence === fullRefreshSequence) {
        loading = false;
        initialRefreshComplete = true;
      }
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
    if (sequence !== actionStateRefreshSequence) {
      logClientEvent("action state refresh discarded stale response", { sequence, latest_sequence: actionStateRefreshSequence, reason });
      return;
    }
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
    const params = new URLSearchParams();
    if (lastEventID > 0) params.set("after", String(lastEventID));
    if (apiAuthToken) params.set("token", apiAuthToken);
    const query = params.toString();
    return `${protocol}//${window.location.host}/api/events/ws${query ? `?${query}` : ""}`;
  }

  function connectEvents() {
    if (authRejected) return;
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
      detail.job_app_id = jobAppID(event.payload);
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

  function clearLocalPairingToken() {
    apiAuthToken = "";
    authRejected = true;
    error = "";
    try {
      window.localStorage.removeItem(authStorageKey);
    } catch (_err) {
      // Clearing in-memory state is still enough for this browser session.
    }
    closeEventSocket();
  }

  function retryPairing() {
    authRejected = false;
    error = "";
    loading = true;
    void refresh();
    connectEvents();
  }

  function handleDomainEvent(event: DomainEvent) {
    if (event.id > lastEventID) lastEventID = event.id;
    if (event.type === "ui.changed") {
      if (isUISettings(event.payload)) applyUIPreferencesFromUI(event.payload);
      return;
    }
    if (event.type === "extension_settings.changed") {
      void loadExtensionSettings("extension_settings.changed");
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
    return ["captured-install", "installer-choice", "steam-workshop-action", "extension-notice", "extension-tool-action"].includes(job.type);
  }

  function eventMatchesSelectedGame(event: DomainEvent) {
    const appID = eventAppID(event);
    return Boolean(selectedGame && (!appID || appID === selectedGame.app_id));
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

  function eventAppID(event: DomainEvent) {
    const direct = (event.app_id ?? "").trim();
    if (direct) return direct;
    if (isJob(event.payload)) return jobAppID(event.payload);
    if (event.payload && typeof event.payload === "object") {
      const payloadAppID = (event.payload as { app_id?: unknown }).app_id;
      if (typeof payloadAppID === "string" && payloadAppID.trim()) return payloadAppID.trim();
    }
    return "";
  }

  function jobAppID(job: Job) {
    return (job.app_id || job.payload?.app_id || "").trim();
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
    logClientEvent("game selection started", { app_id: game.app_id, game_name: game.name });
    markGameRecent(game.app_id);
    selectedGame = game;
    surface = "game";
    activeGameModule = "plugins";
    drawer = null;
    exploreSourceID = "nexus";
    resolvedCapture = "";
    bulkCaptureMessage = "";
    extensionSurfaceMessage = "";
    extensionSurfaceBusyID = "";
    captureBrowserPrompt = null;
    captureBrowserOpenBusy = false;
    nexusSearchResults = [];
    nexusSearchTotal = 0;
    nexusSearchError = "";
    nexusSearchMessage = "";
    nexusBrowseDomain = game.nexus_domains?.[0]?.trim().toLowerCase() ?? "";
    busyCatalogOpenModID = null;
    deckLocalArchiveRoots = [];
    deckLocalArchives = [];
    deckArchiveBrowserOpen = false;
    deckArchiveBrowserEntries = [];
    deckArchiveBrowserPath = "";
    deckArchiveBrowserParentPath = "";
    deckArchivePathInput = "";
    deckLocalArchiveMessage = "";
    busyDeckLocalArchivePath = "";
    deployPlan = null;
    deploymentStatus = null;
    deploymentSettings = null;
    deploymentHistory = [];
    restorePointPreview = null;
    restorePointPreviewBusy = 0;
    gameDiagnostics = null;
    gameInfo = null;
    gameExtensionActions = [];
    gameLaunchStatus = null;
    pluginLoadOrder = null;
    lootUserlist = null;
    lootUserlistMessage = "";
    installCandidates = [];
    installerChoicePresets = [];
    try {
      await loadGameState(game);
      await previewDeploy();
      logClientEvent("game selection completed", {
        app_id: game.app_id,
        game_name: game.name,
        mods: installedMods.length,
        profiles: profiles.length
      });
    } catch (err) {
      error = `Unable to load all ${game.name} details: ${compactLogValue(err)}`;
      logClientEvent("game selection failed", { app_id: game.app_id, game_name: game.name, error: compactLogValue(err) });
    }
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
    const response = await apiFetch("/api/settings/ui", {
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

  async function registerManualStoreGame() {
    manualGameMessage = "";
    const payload = {
      store: manualGameStore.trim(),
      store_app_id: manualGameStoreAppID.trim(),
      name: manualGameName.trim(),
      path: manualGamePath.trim()
    };
    if (!payload.store || !payload.store_app_id || !payload.name || !payload.path) {
      manualGameMessage = "Store, store app ID, game name, and install path are required.";
      return;
    }
    manualGameBusy = true;
    try {
      const response = await apiFetch("/api/games/manual", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });
      if (!response.ok) throw new Error(await response.text());
      const result = await response.json() as { ok: boolean; game: Game };
      if (!result.ok) throw new Error("Manual game registration failed.");
      manualGameMessage = `${result.game.name} is registered for ${sourceLabel(result.game.store ?? payload.store)} management.`;
      manualGameStoreAppID = "";
      manualGameName = "";
      manualGamePath = "";
      await refresh("manual-store-game");
      const registered = games.find((game) => game.app_id === result.game.app_id) ?? result.game;
      await selectGame(registered);
      drawer = null;
    } catch (err) {
      manualGameMessage = `Unable to register game: ${compactLogValue(err)}`;
      logClientEvent("manual store game registration failed", { error: compactLogValue(err), store: payload.store, store_app_id: payload.store_app_id });
    } finally {
      manualGameBusy = false;
    }
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
    const activeProfile = profiles.find((profile) => profile.is_default) ?? profiles[0] ?? null;
    await loadProfileCapabilities(activeProfile?.id ?? 0);
  }

  async function loadInstalledMods(game: Game) {
    installedMods = await getJSON<InstalledMod[]>(`/api/games/${game.app_id}/mods`);
  }

  async function getProfileCapabilities(profileID: number) {
    if (profileID <= 0) {
      return { features: [] as ProfileFeature[], files: [] as ProfileFile[], settings: [] as ExtensionSetting[] };
    }
    const [features, files, settings] = await Promise.all([
      getJSON<ProfileFeature[]>(`/api/profiles/${profileID}/features`),
      getJSON<ProfileFile[]>(`/api/profiles/${profileID}/files`),
      getJSON<ExtensionSetting[]>(`/api/profiles/${profileID}/extension-settings`)
    ]);
    return { features, files, settings };
  }

  async function loadProfileCapabilities(profileID = defaultProfileID()) {
    const { features, files, settings } = await getProfileCapabilities(profileID);
    profileFeatures = features;
    profileFiles = files;
    profileExtensionSettings = settings;
  }

  async function loadGameState(game: Game) {
    const sequence = ++selectedGameLoadSequence;
    const requestAppID = game.app_id;
    const [nextProfiles, nextMods, nextCandidates, nextPresets, nextLocalArchives, nextExternalMods, nextDeploymentStatus, nextDeploymentSettings, nextDeploymentHistory, nextPluginLoadOrder, nextDiagnostics, nextGameInfo, nextExtensionActions, nextLaunchStatus, nextWorkshopState] = await Promise.all([
      getJSON<Profile[]>(`/api/games/${game.app_id}/profiles`),
      getJSON<InstalledMod[]>(`/api/games/${game.app_id}/mods`),
      getJSON<InstallCandidate[]>(`/api/games/${game.app_id}/install-candidates`),
      getJSON<InstallerChoicePreset[]>(`/api/games/${game.app_id}/installer-choice-presets`),
      getJSON<LocalArchiveList>(`/api/games/${game.app_id}/local-archives`),
      getJSON<ExternalModList>(`/api/games/${game.app_id}/external-mods`),
      getJSON<DeploymentStatus>(`/api/games/${game.app_id}/deploy/status`),
      getJSON<DeploymentSettings>(`/api/games/${game.app_id}/deploy/settings`),
      getJSON<{ deployments: DeploymentHistoryItem[] }>(`/api/games/${game.app_id}/deploy/history?limit=5`),
      getJSON<PluginLoadOrder>(`/api/games/${game.app_id}/load-order`),
      getJSON<GameDiagnostics>(`/api/games/${game.app_id}/diagnostics`),
      getJSON<GameInfo>(`/api/games/${game.app_id}/info`),
      getJSON<{ actions: GameExtensionAction[] }>(`/api/games/${game.app_id}/extension-actions`),
      getJSON<GameLaunchStatus>(`/api/games/${game.app_id}/launch`),
      getJSON<WorkshopState>(`/api/games/${game.app_id}/workshop`)
    ]);
    if (!selectedGame || selectedGame.app_id !== requestAppID || sequence !== selectedGameLoadSequence) {
      logClientEvent("selected game state discarded stale response", {
        sequence,
        latest_sequence: selectedGameLoadSequence,
        request_app_id: requestAppID,
        selected_app_id: selectedGame?.app_id ?? ""
      });
      return;
    }
    const nextProfile = nextProfiles.find((profile) => profile.is_default) ?? nextProfiles[0] ?? null;
    const nextProfileCapabilities = await getProfileCapabilities(nextProfile?.id ?? 0);
    if (!selectedGame || selectedGame.app_id !== requestAppID || sequence !== selectedGameLoadSequence) {
      logClientEvent("selected game capabilities discarded stale response", {
        sequence,
        latest_sequence: selectedGameLoadSequence,
        request_app_id: requestAppID,
        selected_app_id: selectedGame?.app_id ?? ""
      });
      return;
    }
	profiles = nextProfiles;
	profileFeatures = nextProfileCapabilities.features;
	profileFiles = nextProfileCapabilities.files;
	profileExtensionSettings = nextProfileCapabilities.settings;
	installedMods = nextMods;
    installCandidates = nextCandidates;
    installerChoicePresets = nextPresets;
    deckLocalArchiveRoots = nextLocalArchives.roots ?? [];
    deckLocalArchives = nextLocalArchives.files ?? [];
    externalModCandidates = nextExternalMods.items ?? [];
    selectedExternalModPaths = {};
    externalModAdoptionMessage = "";
    deploymentStatus = nextDeploymentStatus;
    deploymentSettings = nextDeploymentSettings;
    deploymentHistory = nextDeploymentHistory.deployments ?? [];
    if (restorePointPreview && !deploymentHistory.some((deployment) => deployment.id === restorePointPreview?.deployment_id)) {
      restorePointPreview = null;
    }
    pluginLoadOrder = nextPluginLoadOrder;
    await loadLOOTUserlist(game, nextPluginLoadOrder);
    gameDiagnostics = nextDiagnostics;
    gameInfo = nextGameInfo;
    gameExtensionActions = nextExtensionActions.actions ?? [];
    gameLaunchStatus = nextLaunchStatus;
    workshopState = nextWorkshopState;
    workshopItems = nextWorkshopState.items ?? [];
    globalInstallCandidates = mergeInstallCandidatesForGame(globalInstallCandidates, game.app_id, nextCandidates);
    reconcileBusyState();
  }

  function defaultProfileID() {
    return selectedProfile?.id ?? profiles[0]?.id ?? 0;
  }

  function normalizedProfileID(value: string | number | undefined, defaultValue = defaultProfileID()) {
    const parsed = Number(value ?? 0);
    if (parsed > 0 && profiles.some((profile) => profile.id === parsed)) return parsed;
    return defaultValue;
  }

  function selectedInstallProfileID() {
    return normalizedProfileID(installTargetProfileID);
  }

  function actionInstallProfileID(action: Job) {
    return normalizedProfileID(actionProfileTargets[action.id], normalizedProfileID(action.payload?.target_profile_id, selectedInstallProfileID()));
  }

  function candidateInstallProfileID(candidate: InstallCandidate) {
    return normalizedProfileID(candidateProfileTargets[candidate.id], normalizedProfileID(candidate.target_profile_id, selectedInstallProfileID()));
  }

  function transferProfiles() {
    if (!selectedProfile) return [];
    return profiles.filter((profile) => profile.id !== selectedProfile.id);
  }

  function transferTargetProfileID(mod: InstalledMod) {
    const defaultTarget = transferProfiles()[0]?.id ?? 0;
    const target = normalizedProfileID(profileTransferTargets[mod.id], defaultTarget);
    return target === selectedProfile?.id ? defaultTarget : target;
  }

  function profileOptionLabel(profile: Profile, defaultLabel = "default") {
    const state = `${profile.enabled_mod_count} on / ${profile.mod_count} total`;
    return `${profile.name}${profile.is_default ? ` - ${defaultLabel}` : ""} (${state})`;
  }

  async function refreshSelectedGame(options: { refreshPreview?: boolean; refreshJobs?: boolean; reason?: string } = {}) {
    if (!selectedGame) return;
    const game = selectedGame;
    const sequence = ++selectedGameRefreshSequence;
    const appID = game.app_id;
    const reason = options.reason ?? "manual";
    logClientEvent("selected game refresh started", {
      sequence,
      reason,
      selected_app_id: appID,
      refresh_preview: Boolean(options.refreshPreview),
      refresh_jobs: Boolean(options.refreshJobs)
    });
    if (options.refreshJobs) await refreshActionState(`${reason}:selected-game-jobs`);
    if (!selectedGame || selectedGame.app_id !== game.app_id) {
      logClientEvent("selected game refresh discarded after selection changed", {
        sequence,
        reason,
        request_app_id: appID,
        selected_app_id: selectedGame?.app_id ?? ""
      });
      return;
    }
    await loadGameState(game);
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/deploy/settings`, {
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

  async function updateInstallSettings(patch: Partial<Status["install"]>) {
    if (!status) return;
    error = "";
    const nextInstall = { ...status.install, ...patch };
    const response = await apiFetch("/api/settings/install", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        auto_install_captured_downloads: Boolean(nextInstall.auto_install_captured_downloads),
        auto_enable_installed_mods: Boolean(nextInstall.auto_enable_installed_mods),
        auto_show_fomod_installers: Boolean(nextInstall.auto_show_fomod_installers ?? true)
      })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const nextStatus = await response.json() as Status;
    status = nextStatus;
    applyUIPreferences(nextStatus);
  }

  function updateAutoInstall(value: boolean) {
    void updateInstallSettings({ auto_install_captured_downloads: value });
  }

  function updateAutoEnable(value: boolean) {
    void updateInstallSettings({ auto_enable_installed_mods: value });
  }

  function updateAutoShowFOMOD(value: boolean) {
    void updateInstallSettings({ auto_show_fomod_installers: value });
  }

  async function updateDownloadConcurrency(maxDownloads: number) {
    error = "";
    const response = await apiFetch("/api/settings/downloads", {
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
    const response = await apiFetch("/api/settings/downloads", {
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

  async function updateNexusAPIKey() {
    const key = nexusAPIKey.trim();
    if (!key) {
      error = "Enter a Nexus API key before saving.";
      return;
    }
    nexusSettingsBusy = true;
    nexusSettingsMessage = "";
    error = "";
    try {
      const response = await apiFetch("/api/settings/nexus", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ api_key: key })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const nextStatus = await response.json() as Status;
      status = nextStatus;
      applyUIPreferences(nextStatus);
      nexusAPIKey = "";
      nexusSettingsMessage = "Nexus API key saved.";
      catalogs = await getJSON<CatalogStatus[]>("/api/catalogs");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      nexusSettingsBusy = false;
    }
  }

  async function updateCatalogCredential(provider: "modio" | "curseforge", apiKey: string) {
    catalogSettingsBusy = provider;
    catalogSettingsMessage = "";
    error = "";
    const body = provider === "modio"
      ? { modio: { api_key: apiKey } }
      : { curseforge: { api_key: apiKey } };
    try {
      const response = await apiFetch("/api/settings/catalogs", {
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

  function extensionSettingKey(setting: ExtensionSetting) {
    return `${setting.extension_id}\u0000${setting.setting_id}`;
  }

  function extensionSettingValueType(setting: ExtensionSetting) {
    const valueType = (setting.value_type || "json").trim().toLowerCase();
    if (["string", "path", "bool", "number"].includes(valueType)) return valueType;
    return "json";
  }

  function extensionSettingReady(setting: ExtensionSetting) {
    return (setting.status || "ready").trim() === "ready";
  }

  function extensionSettingDisplayValue(setting: ExtensionSetting) {
    const value = setting.value;
    if (value === null || value === undefined) return "";
    const valueType = extensionSettingValueType(setting);
    if (valueType === "bool") return value === true ? "true" : "false";
    if (valueType === "number") return typeof value === "number" ? String(value) : "";
    if (valueType === "string" || valueType === "path") return typeof value === "string" ? value : "";
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return String(value);
    }
  }

  function setExtensionSettings(nextSettings: ExtensionSetting[]) {
    extensionSettings = nextSettings;
    const nextKeys = new Set(nextSettings.map(extensionSettingKey));
    const nextDrafts: Record<string, string> = {};
    for (const setting of nextSettings) {
      const key = extensionSettingKey(setting);
      nextDrafts[key] = extensionSettingDrafts[key] ?? extensionSettingDisplayValue(setting);
    }
    for (const [key, value] of Object.entries(extensionSettingDrafts)) {
      if (nextKeys.has(key)) nextDrafts[key] = value;
    }
    extensionSettingDrafts = nextDrafts;
  }

  async function loadExtensionSettings(reason = "manual") {
    try {
      const nextSettings = await getJSON<ExtensionSetting[]>("/api/extensions/settings");
      setExtensionSettings(nextSettings);
      logClientEvent("extension settings refreshed", { reason, settings: nextSettings.length });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
      logClientEvent("extension settings refresh failed", { reason, error: compactLogValue(error) });
    }
  }

  function extensionSettingDraft(setting: ExtensionSetting) {
    return extensionSettingDrafts[extensionSettingKey(setting)] ?? extensionSettingDisplayValue(setting);
  }

  function updateExtensionSettingDraft(setting: ExtensionSetting, value: string) {
    extensionSettingDrafts = { ...extensionSettingDrafts, [extensionSettingKey(setting)]: value };
  }

  function extensionSettingGroups() {
    const groups = new Map<string, ExtensionSetting[]>();
    for (const setting of extensionSettings) {
      const key = setting.extension_id || "extension";
      groups.set(key, [...(groups.get(key) ?? []), setting]);
    }
    return Array.from(groups.entries()).map(([extensionID, settings]) => ({
      extensionID,
      settings: settings.sort((a, b) => a.name.localeCompare(b.name) || a.setting_id.localeCompare(b.setting_id))
    })).sort((a, b) => a.extensionID.localeCompare(b.extensionID));
  }

  function readyExtensionSettingCount() {
    return extensionSettings.filter(extensionSettingReady).length;
  }

  function extensionSettingPayload(setting: ExtensionSetting) {
    const valueType = extensionSettingValueType(setting);
    const draft = extensionSettingDraft(setting);
    if (valueType === "path") return draft.trim() === "" ? null : draft.trim();
    if (valueType === "string") return draft;
    if (valueType === "bool") return draft === "true";
    if (valueType === "number") {
      if (draft.trim() === "") return null;
      const parsed = Number(draft);
      if (!Number.isFinite(parsed)) throw new Error(`${setting.name} must be a number.`);
      return parsed;
    }
    if (draft.trim() === "") return null;
    return JSON.parse(draft);
  }

  async function saveExtensionSetting(setting: ExtensionSetting) {
    const key = extensionSettingKey(setting);
    extensionSettingBusy = key;
    extensionSettingsMessage = "";
    error = "";
    try {
      const value = extensionSettingPayload(setting);
      const response = await apiFetch(`/api/extensions/${encodeURIComponent(setting.extension_id)}/settings/${encodeURIComponent(setting.setting_id)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ value })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const updated = await response.json() as ExtensionSetting;
      setExtensionSettings(extensionSettings.map((item) => extensionSettingKey(item) === key ? updated : item));
      extensionSettingDrafts = { ...extensionSettingDrafts, [key]: extensionSettingDisplayValue(updated) };
      extensionSettingsMessage = `${updated.name} saved.`;
      if (selectedGame) await refreshSelectedGame({ refreshPreview: true, reason: "extension-setting" });
    } catch (err) {
      error = err instanceof SyntaxError ? "Setting value must be valid JSON." : err instanceof Error ? err.message : String(err);
    } finally {
      extensionSettingBusy = "";
    }
  }

  async function saveProfileExtensionSetting(setting: ExtensionSetting) {
    if (!selectedProfile) return;
    const key = extensionSettingKey(setting);
    extensionSettingBusy = key;
    extensionSettingsMessage = "";
    error = "";
    try {
      const value = extensionSettingPayload(setting);
      const response = await apiFetch(`/api/profiles/${selectedProfile.id}/extensions/${encodeURIComponent(setting.extension_id)}/settings/${encodeURIComponent(setting.setting_id)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ value })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const updated = await response.json() as ExtensionSetting;
      profileExtensionSettings = profileExtensionSettings.map((item) => extensionSettingKey(item) === key ? { ...updated, options: setting.options, options_error: setting.options_error } : item);
      extensionSettingDrafts = { ...extensionSettingDrafts, [key]: extensionSettingDisplayValue(updated) };
      extensionSettingsMessage = `${updated.name} saved for ${selectedProfile.name}.`;
      if (selectedGame) await refreshSelectedGame({ refreshPreview: true, reason: "profile-extension-setting" });
    } catch (err) {
      error = err instanceof SyntaxError ? "Setting value must be valid JSON." : err instanceof Error ? err.message : String(err);
    } finally {
      extensionSettingBusy = "";
    }
  }

  function extensionSettingSelectedIDs(setting: ExtensionSetting) {
    const draft = extensionSettingDraft(setting);
    try {
      const parsed = JSON.parse(draft || "[]");
      return Array.isArray(parsed) ? parsed.map((value) => String(value)) : [];
    } catch {
      return [];
    }
  }

  function toggleExtensionSettingOption(setting: ExtensionSetting, optionID: string, checked: boolean) {
    const selected = new Set(extensionSettingSelectedIDs(setting));
    if (checked) selected.add(optionID);
    else selected.delete(optionID);
    updateExtensionSettingDraft(setting, JSON.stringify(Array.from(selected), null, 2));
  }

  async function createProfile() {
    if (!selectedGame || !profileName.trim()) return;
    error = "";
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/profiles`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        name: profileName,
        source_profile_id: copyProfileFromActive ? selectedProfile?.id ?? 0 : 0
      })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    profileName = "";
    copyProfileFromActive = false;
    await loadProfiles(selectedGame);
  }

  async function selectProfileByID(profileID: string) {
    const profile = profiles.find((item) => String(item.id) === profileID);
    if (!profile) return;
    await setDefaultProfile(profile);
  }

  async function setDefaultProfile(profile: Profile) {
    if (!selectedGame) return;
    installTargetProfileID = String(profile.id);
    if (profile.is_default) return;
    error = "";
    const response = await apiFetch(`/api/profiles/${profile.id}/default`, { method: "PUT" });
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

  async function deleteProfile(profile: Profile) {
    if (!selectedGame || profiles.length <= 1) return;
    error = "";
    const response = await apiFetch(`/api/profiles/${profile.id}`, { method: "DELETE" });
    const raw = await response.text();
    let result: DeleteProfileResult | null = null;
    if (raw.trim()) {
      try {
        result = JSON.parse(raw) as DeleteProfileResult;
      } catch {
        result = null;
      }
    }
    if (!response.ok) {
      if (result?.apply) handleProfileApplyResult(result.apply);
      error = result?.apply?.message || raw || "Unable to remove profile.";
      await refreshSelectedGame({ refreshPreview: true });
      return;
    }
    if (result?.apply) handleProfileApplyResult(result.apply);
    if (result?.active_profile) installTargetProfileID = String(result.active_profile.id);
    await refreshSelectedGame({ refreshPreview: true, refreshJobs: true });
  }

  function askDeleteProfile(profile: Profile) {
    if (!selectedGame || profiles.length <= 1) return;
    const replacement = profiles.find((item) => item.id !== profile.id);
    confirmation = {
      title: "Remove profile",
      message: `Remove ${profile.name} from ${selectedGame.name}.`,
      detail: profile.is_default
        ? `DMM will switch to ${replacement?.name ?? "another profile"} first, apply that profile, then remove ${profile.name}. Installed mod files and cached downloads are kept.`
        : "Installed mod files and cached downloads are kept. Only this profile's membership, conflict choices, and history are removed.",
      confirmLabel: "Remove Profile",
      danger: true,
      run: () => deleteProfile(profile)
    };
  }

  function profileFeatureKey(feature: ProfileFeature) {
    return `${feature.profile_id}:${feature.feature_id}`;
  }

  function profileFeatureReady(feature: ProfileFeature) {
    return !feature.status || feature.status === "ready";
  }

  async function setProfileFeatureEnabled(feature: ProfileFeature, enabled: boolean) {
    if (!selectedProfile || feature.profile_id !== selectedProfile.id || !profileFeatureReady(feature)) return;
    const key = profileFeatureKey(feature);
    busyProfileFeatures = { ...busyProfileFeatures, [key]: true };
    profileFeatureMessage = "";
    error = "";
    try {
      const response = await apiFetch(`/api/profiles/${selectedProfile.id}/features/${encodeURIComponent(feature.feature_id)}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const updated = await response.json() as ProfileFeature;
      profileFeatures = profileFeatures.map((item) => item.feature_id === updated.feature_id ? updated : item);
      profileFeatureMessage = `${updated.name || updated.feature_id} ${updated.enabled ? "enabled" : "disabled"} for ${selectedProfile.name}.`;
      await refreshSelectedGame({ refreshPreview: true, reason: "profile-feature-toggle" });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      const { [key]: _removed, ...rest } = busyProfileFeatures;
      busyProfileFeatures = rest;
    }
  }

  async function setModEnabled(mod: InstalledMod, enabled: boolean) {
    if (!selectedProfile) return;
    error = "";
    setBusyMod(mod.id, "toggle");
    try {
      const response = await apiFetch(`/api/profiles/${selectedProfile.id}/mods/${mod.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          enabled,
          cascade_dependencies: true,
          include_recommended_dependencies: false
        })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: ProfileModUpdateResult = await response.json();
      const updatedMods = new Map<number, InstalledMod>([[result.mod.id, result.mod]]);
      for (const cascadeMod of result.cascade ?? []) updatedMods.set(cascadeMod.id, cascadeMod);
      installedMods = installedMods.map((item) => updatedMods.get(item.id) ?? item);
      if ((result.cascade?.length ?? 0) > 0) {
        actionMessage = `${enabled ? "Enabled" : "Disabled"} ${mod.name} and ${result.cascade?.length} required dependenc${result.cascade?.length === 1 ? "y" : "ies"}.`;
      }
      if ((result.cascade_notes?.length ?? 0) > 0) {
        error = result.cascade_notes?.join(" ") ?? "";
      }
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
    const response = await apiFetch(`/api/profiles/${selectedProfile.id}/mods/order`, {
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

  async function moveExtensionLoadOrderEntry(order: ExtensionLoadOrder, entry: ExtensionLoadOrderEntry, direction: -1 | 1) {
    if (!selectedProfile || !entry.mutable || !entry.installed_mod_id) return;
    const orderedEntries = order.entries.filter((item) => item.mutable && item.installed_mod_id);
    const from = orderedEntries.findIndex((item) => item.installed_mod_id === entry.installed_mod_id);
    const to = from + direction;
    if (from < 0 || to < 0 || to >= orderedEntries.length) return;
    const targetID = orderedEntries[to].installed_mod_id;
    const current = [...installedMods].sort((a, b) => a.priority - b.priority || a.name.localeCompare(b.name));
    const movingIndex = current.findIndex((item) => item.id === entry.installed_mod_id);
    const targetIndex = current.findIndex((item) => item.id === targetID);
    if (movingIndex < 0 || targetIndex < 0) return;
    const [moving] = current.splice(movingIndex, 1);
    const adjustedTarget = current.findIndex((item) => item.id === targetID);
    if (adjustedTarget < 0) return;
    current.splice(direction < 0 ? adjustedTarget : adjustedTarget + 1, 0, moving);
    error = "";
    setBusyMod(entry.installed_mod_id, "order");
    try {
      const response = await apiFetch(`/api/profiles/${selectedProfile.id}/mods/order`, {
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
    } finally {
      clearBusyMod(entry.installed_mod_id);
    }
  }

  function pluginActivationRowKey(plugin: PluginLoadOrderEntry) {
    return plugin.name.trim().toLowerCase();
  }

  function mutablePluginRows() {
    return pluginLoadOrder?.plugins.filter((plugin) => plugin.mutable) ?? [];
  }

  function mutablePluginIndex(plugin: PluginLoadOrderEntry) {
    return mutablePluginRows().findIndex((item) => pluginActivationRowKey(item) === pluginActivationRowKey(plugin));
  }

  async function setPluginActivationEnabled(plugin: PluginLoadOrderEntry, enabled: boolean) {
    if (!selectedProfile || !pluginLoadOrder?.activation_id || !plugin.mutable) return;
    const key = pluginActivationRowKey(plugin);
    busyPluginActivationRows = { ...busyPluginActivationRows, [key]: true };
    error = "";
    try {
      const response = await apiFetch(`/api/profiles/${selectedProfile.id}/plugin-activation/${encodeURIComponent(pluginLoadOrder.activation_id)}/plugins`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name: plugin.name, enabled })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: ProfilePluginActivationUpdateResult = await response.json();
      pluginLoadOrder = result.load_order;
      handleProfileApplyResult(result.apply);
      await refreshSelectedGame({ refreshPreview: true });
    } finally {
      const { [key]: _removed, ...rest } = busyPluginActivationRows;
      busyPluginActivationRows = rest;
    }
  }

  async function movePluginActivation(plugin: PluginLoadOrderEntry, direction: -1 | 1) {
    if (!selectedProfile || !pluginLoadOrder?.activation_id || !plugin.mutable) return;
    const mutablePlugins = pluginLoadOrder.plugins.filter((item) => item.mutable);
    const from = mutablePlugins.findIndex((item) => pluginActivationRowKey(item) === pluginActivationRowKey(plugin));
    const to = from + direction;
    if (from < 0 || to < 0 || to >= mutablePlugins.length) return;
    [mutablePlugins[from], mutablePlugins[to]] = [mutablePlugins[to], mutablePlugins[from]];
    const key = pluginActivationRowKey(plugin);
    busyPluginActivationRows = { ...busyPluginActivationRows, [key]: true };
    error = "";
    try {
      const response = await apiFetch(`/api/profiles/${selectedProfile.id}/plugin-activation/${encodeURIComponent(pluginLoadOrder.activation_id)}/order`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ plugins: mutablePlugins.map((item) => item.name) })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: ProfilePluginActivationUpdateResult = await response.json();
      pluginLoadOrder = result.load_order;
      handleProfileApplyResult(result.apply);
      await refreshSelectedGame({ refreshPreview: true });
    } finally {
      const { [key]: _removed, ...rest } = busyPluginActivationRows;
      busyPluginActivationRows = rest;
    }
  }

  function lootFileStatusLabel(file?: LOOTFileStatus) {
    if (!file?.exists) return "Missing";
    const updated = file.updated_at ? new Date(file.updated_at).toLocaleString() : "unknown time";
    const size = file.size_bytes ? `${Math.round(file.size_bytes / 1024)} KB` : "0 KB";
    return `${size} · ${updated}`;
  }

  function emptyLOOTUserlist(): LOOTUserlist {
    return { globals: [], plugins: [], groups: [] };
  }

  function lootSummaryFor(userlist: LOOTUserlist): LOOTRuleSummary {
    return {
      plugins: userlist.plugins.length,
      groups: userlist.groups.length,
      rules: userlist.plugins.reduce((sum, plugin) => sum + (plugin.after?.length ?? 0) + (plugin.requires?.length ?? 0) + (plugin.incompatible?.length ?? 0), 0),
      group_rules: userlist.groups.reduce((sum, group) => sum + (group.after?.length ?? 0), 0)
    };
  }

  function lootUserlistURL(game: Game, profileID = selectedProfile?.id ?? profiles.find((profile) => profile.is_default)?.id ?? profiles[0]?.id ?? 0) {
    const params = profileID > 0 ? `?profile_id=${encodeURIComponent(String(profileID))}` : "";
    return `/api/games/${game.app_id}/load-order/loot/userlist${params}`;
  }

  function lootMetadataRefreshURL(game: Game, profileID = selectedProfile?.id ?? profiles.find((profile) => profile.is_default)?.id ?? profiles[0]?.id ?? 0) {
    const params = profileID > 0 ? `?profile_id=${encodeURIComponent(String(profileID))}` : "";
    return `/api/games/${game.app_id}/load-order/loot/refresh${params}`;
  }

  async function loadLOOTUserlist(game: Game, loadOrder: PluginLoadOrder) {
    lootUserlist = null;
    lootUserlistMessage = "";
    if (!loadOrder.loot?.supported) return;
    try {
      lootUserlist = await getJSON<LOOTUserlist>(lootUserlistURL(game, loadOrder.profile_id));
    } catch (err) {
      lootUserlistMessage = err instanceof Error ? err.message : String(err);
    }
  }

  async function saveLOOTUserlist(next: LOOTUserlist, message: string) {
    if (!selectedGame || lootUserlistBusy) return;
    lootUserlistBusy = true;
    lootUserlistMessage = "";
    error = "";
    try {
      const response = await apiFetch(lootUserlistURL(selectedGame), {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(next)
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const saved: LOOTUserlist = await response.json();
      lootUserlist = saved;
      lootUserlistMessage = message;
      if (pluginLoadOrder?.loot) {
        pluginLoadOrder = {
          ...pluginLoadOrder,
          loot: {
            ...pluginLoadOrder.loot,
            userlist_rules: lootSummaryFor(saved),
            userlist_warning: ""
          }
        };
      }
    } finally {
      lootUserlistBusy = false;
    }
  }

  function cleanRuleText(value: string) {
    return value.trim().replace(/\s+/g, " ");
  }

  function ruleListFor(plugin: LOOTUserlistPlugin, ruleType: "after" | "requires" | "incompatible") {
    if (ruleType === "requires") return plugin.requires ?? [];
    if (ruleType === "incompatible") return plugin.incompatible ?? [];
    return plugin.after ?? [];
  }

  function withRuleList(plugin: LOOTUserlistPlugin, ruleType: "after" | "requires" | "incompatible", values: string[]) {
    if (ruleType === "requires") return { ...plugin, requires: values };
    if (ruleType === "incompatible") return { ...plugin, incompatible: values };
    return { ...plugin, after: values };
  }

  function appendUnique(values: string[] | undefined, value: string) {
    const cleaned = cleanRuleText(value);
    if (!cleaned) return values ?? [];
    const next = [...(values ?? [])];
    if (!next.some((item) => item.toLowerCase() === cleaned.toLowerCase())) {
      next.push(cleaned);
    }
    return next;
  }

  function removeRuleValue(values: string[] | undefined, value: string) {
    const key = cleanRuleText(value).toLowerCase();
    return (values ?? []).filter((item) => cleanRuleText(item).toLowerCase() !== key);
  }

  async function addLOOTPluginRule() {
    const pluginName = cleanRuleText(lootRulePlugin);
    const reference = cleanRuleText(lootRuleReference);
    if (!pluginName || !reference) {
      lootUserlistMessage = "Plugin and reference are required.";
      return;
    }
    const current = lootUserlist ?? emptyLOOTUserlist();
    const index = current.plugins.findIndex((plugin) => plugin.name.toLowerCase() === pluginName.toLowerCase());
    const plugins = [...current.plugins];
    if (index >= 0) {
      plugins[index] = withRuleList(plugins[index], lootRuleType, appendUnique(ruleListFor(plugins[index], lootRuleType), reference));
    } else {
      plugins.push(withRuleList({ name: pluginName }, lootRuleType, [reference]));
    }
    await saveLOOTUserlist({ ...current, plugins }, "LOOT plugin rule saved.");
    lootRuleReference = "";
  }

  async function removeLOOTPluginRule(plugin: LOOTUserlistPlugin, ruleType: "after" | "requires" | "incompatible", reference: string) {
    const current = lootUserlist ?? emptyLOOTUserlist();
    const plugins = current.plugins.map((item) => {
      if (item.name.toLowerCase() !== plugin.name.toLowerCase()) return item;
      return withRuleList(item, ruleType, removeRuleValue(ruleListFor(item, ruleType), reference));
    }).filter((item) => item.group || (item.after?.length ?? 0) > 0 || (item.requires?.length ?? 0) > 0 || (item.incompatible?.length ?? 0) > 0);
    await saveLOOTUserlist({ ...current, plugins }, "LOOT plugin rule removed.");
  }

  async function assignLOOTPluginGroup() {
    const pluginName = cleanRuleText(lootGroupPlugin);
    const group = cleanRuleText(lootGroupAssignment);
    if (!pluginName || !group) {
      lootUserlistMessage = "Plugin and group are required.";
      return;
    }
    const current = lootUserlist ?? emptyLOOTUserlist();
    const index = current.plugins.findIndex((plugin) => plugin.name.toLowerCase() === pluginName.toLowerCase());
    const plugins = [...current.plugins];
    if (index >= 0) {
      plugins[index] = { ...plugins[index], group };
    } else {
      plugins.push({ name: pluginName, group });
    }
    const groups = current.groups.some((item) => item.name.toLowerCase() === group.toLowerCase())
      ? current.groups
      : [...current.groups, { name: group, after: [] }];
    await saveLOOTUserlist({ ...current, plugins, groups }, "LOOT plugin group saved.");
  }

  async function clearLOOTPluginGroup(plugin: LOOTUserlistPlugin) {
    const current = lootUserlist ?? emptyLOOTUserlist();
    const plugins = current.plugins.map((item) => (
      item.name.toLowerCase() === plugin.name.toLowerCase() ? { ...item, group: "" } : item
    )).filter((item) => item.group || (item.after?.length ?? 0) > 0 || (item.requires?.length ?? 0) > 0 || (item.incompatible?.length ?? 0) > 0);
    await saveLOOTUserlist({ ...current, plugins }, "LOOT plugin group cleared.");
  }

  async function addLOOTGroupRule() {
    const group = cleanRuleText(lootGroupName);
    const reference = cleanRuleText(lootGroupReference);
    if (!group) {
      lootUserlistMessage = "Group name is required.";
      return;
    }
    const current = lootUserlist ?? emptyLOOTUserlist();
    const index = current.groups.findIndex((item) => item.name.toLowerCase() === group.toLowerCase());
    const groups = [...current.groups];
    if (index >= 0) {
      groups[index] = { ...groups[index], after: reference ? appendUnique(groups[index].after, reference) : groups[index].after ?? [] };
    } else {
      groups.push({ name: group, after: reference ? [reference] : [] });
    }
    await saveLOOTUserlist({ ...current, groups }, "LOOT group rule saved.");
    lootGroupReference = "";
  }

  async function removeLOOTGroupRule(group: LOOTUserlistGroup, reference: string) {
    const current = lootUserlist ?? emptyLOOTUserlist();
    const groups = current.groups.map((item) => (
      item.name.toLowerCase() === group.name.toLowerCase()
        ? { ...item, after: removeRuleValue(item.after, reference) }
        : item
    ));
    await saveLOOTUserlist({ ...current, groups }, "LOOT group rule removed.");
  }

  async function removeLOOTGroup(group: LOOTUserlistGroup) {
    const current = lootUserlist ?? emptyLOOTUserlist();
    const key = group.name.toLowerCase();
    const groups = current.groups
      .filter((item) => item.name.toLowerCase() !== key)
      .map((item) => ({ ...item, after: removeRuleValue(item.after, group.name) }));
    const plugins = current.plugins.map((plugin) => (
      plugin.group?.toLowerCase() === key ? { ...plugin, group: "" } : plugin
    )).filter((item) => item.group || (item.after?.length ?? 0) > 0 || (item.requires?.length ?? 0) > 0 || (item.incompatible?.length ?? 0) > 0);
    await saveLOOTUserlist({ ...current, plugins, groups }, "LOOT group removed.");
  }

  function askClearLOOTUserlist() {
    confirmation = {
      title: "Clear LOOT rules",
      message: "Remove all DMM-managed LOOT plugin rules and group assignments for this game.",
      detail: "This updates DMM's userlist.yaml. It does not remove mods or game files.",
      confirmLabel: "Clear Rules",
      danger: true,
      run: () => saveLOOTUserlist(emptyLOOTUserlist(), "LOOT userlist cleared.")
    };
  }

  async function refreshLOOTMetadata() {
    if (!selectedGame || lootRefreshBusy) return;
    lootRefreshBusy = true;
    error = "";
    try {
      const response = await apiFetch(lootMetadataRefreshURL(selectedGame), { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const status: LOOTStatus = await response.json();
      if (pluginLoadOrder) {
        pluginLoadOrder = { ...pluginLoadOrder, loot: status };
      }
    } finally {
      lootRefreshBusy = false;
    }
  }

  async function sortWithLOOT() {
    if (!selectedGame || !pluginLoadOrder?.loot?.supported || lootSortBusy) return;
    lootSortBusy = true;
    lootUserlistMessage = "";
    error = "";
    try {
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/load-order/loot/sort`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ profile_id: pluginLoadOrder.profile_id ?? selectedProfile?.id })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: LOOTSortResult = await response.json();
      pluginLoadOrder = result.load_order;
      lootUserlistMessage = result.warnings?.length
        ? `LOOT sorted with ${result.engine ?? "configured helper"} and returned ${result.warnings.length} warning${result.warnings.length === 1 ? "" : "s"}.`
        : `LOOT sorted ${result.sorted_plugins.length} plugin${result.sorted_plugins.length === 1 ? "" : "s"} with ${result.engine ?? "configured helper"}.`;
      handleProfileApplyResult(result.apply);
      await refreshSelectedGame({ refreshPreview: true });
    } finally {
      lootSortBusy = false;
    }
  }

  async function removeInstalledMod(mod: InstalledMod) {
    if (!selectedGame) return;
    error = "";
    setBusyMod(mod.id, "remove");
    try {
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/mods/${mod.id}`, { method: "DELETE" });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: ProfileModUpdateResult = await response.json();
      installedMods = installedMods.filter((item) => item.id !== mod.id);
      handleProfileApplyResult(result.apply);
      await refreshSelectedGame({ refreshPreview: true });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      clearBusyMod(mod.id);
    }
  }

  async function transferInstalledMod(mod: InstalledMod, move: boolean) {
    if (!selectedProfile) return;
    const targetProfileID = transferTargetProfileID(mod);
    if (targetProfileID <= 0 || targetProfileID === selectedProfile.id) {
      error = "Choose a different target profile first.";
      return;
    }
    const action = move ? "move" : "copy";
    error = "";
    setBusyMod(mod.id, action);
    try {
      const response = await apiFetch(`/api/profiles/${selectedProfile.id}/mods/${mod.id}/${action}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ target_profile_id: targetProfileID, enabled: mod.enabled })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: ProfileModUpdateResult = await response.json();
      if (move) {
        installedMods = installedMods.filter((item) => item.id !== mod.id);
      }
      handleProfileApplyResult(result.apply);
      await refreshSelectedGame({ refreshPreview: true });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      clearBusyMod(mod.id);
    }
  }

  async function reinstallInstalledMod(mod: InstalledMod, promptInstallerChoices = false) {
    if (!selectedGame) return;
    error = "";
    setBusyMod(mod.id, promptInstallerChoices ? "reconfigure" : "reinstall");
    try {
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/mods/${mod.id}/reinstall`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ prompt_installer_choices: promptInstallerChoices })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: { job?: Job; mod?: InstalledMod; candidate?: InstallCandidate } = await response.json();
      if (result.job) upsertJob(result.job);
      if (result.candidate) replaceInstallCandidate(result.candidate);
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
    modUpdateBrowserPrompt = null;
    setBusyMod(mod.id, "update");
    try {
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/mods/${mod.id}/update`, { method: "POST" });
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
        const browserURL = result.file_url ?? result.resolved?.source_url ?? "";
        modUpdateMessage = "Open this update on the Steam Deck, then click Nexus Mod Manager Download to capture it.";
        if (browserURL) {
          modUpdateBrowserPrompt = {
            url: browserURL,
            mod_id: mod.id,
            mod_name: mod.name,
            title: `Update ${mod.name} - Nexus Mods`
          };
        } else {
          error = modUpdateMessage;
        }
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
    modUpdateBrowserPrompt = null;
    modUpdateBusy = true;
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/mods/check-updates`, { method: "POST" });
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

  async function openModUpdateOnDeck(prompt: ModUpdateBrowserPrompt | null) {
    if (!selectedGame || !prompt || modUpdateBrowserOpenBusy) return;
    error = "";
    modUpdateBrowserOpenBusy = true;
    try {
      const response = await apiFetch("/api/decky/browser/open", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url: prompt.url,
          steam_app_id: selectedGame.app_id,
          profile_id: selectedInstallProfileID(),
          source: "web-mod-update",
          title: prompt.title
        })
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      modUpdateMessage = `Opening ${prompt.mod_name} on the Steam Deck. Click Nexus Mod Manager Download there to capture the update.`;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      modUpdateBrowserOpenBusy = false;
    }
  }

  async function openProviderPageOnDeck(url: string, source: string, title: string) {
    if (!selectedGame) return;
    logClientEvent("deck browser handoff requested", { app_id: selectedGame.app_id, source, url });
    const response = await apiFetch("/api/decky/browser/open", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        url,
        steam_app_id: selectedGame.app_id,
        profile_id: selectedInstallProfileID(),
        source,
        title
      })
    });
    if (!response.ok) {
      logClientEvent("deck browser handoff failed", { app_id: selectedGame.app_id, source, status: response.status });
      throw new Error(await response.text());
    }
    const result = await response.json();
    logClientEvent("deck browser handoff queued", { app_id: selectedGame.app_id, source });
    return result;
  }

  async function openCapturePromptOnDeck(prompt: DeckBrowserPrompt | null) {
    if (!prompt || captureBrowserOpenBusy) return;
    error = "";
    captureBrowserOpenBusy = true;
    try {
      await openProviderPageOnDeck(prompt.url, prompt.source, prompt.title);
      bulkCaptureMessage = prompt.message;
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      captureBrowserOpenBusy = false;
    }
  }

  function isHTTPProviderPage(value: string) {
    try {
      const parsed = new URL(value);
      return parsed.protocol === "http:" || parsed.protocol === "https:";
    } catch {
      return false;
    }
  }

  function isNexusHTTPPage(value: string) {
    try {
      const parsed = new URL(value);
      const host = parsed.hostname.toLowerCase().replace(/^www\./, "");
      return (parsed.protocol === "http:" || parsed.protocol === "https:") && host === "nexusmods.com" && /^\/[^/]+\/mods\/\d+/.test(parsed.pathname);
    } catch {
      return false;
    }
  }

  function handleProfileApplyResult(result: ProfileApplyResult | null | undefined) {
    if (!result) return;
    if (result.job) upsertJob(result.job);
    if (result.plan) deployPlan = result.plan;
    if (result.status === "blocked" || result.status === "failed") {
      error = result.message;
    }
  }

  function captureInputURLs(value: string) {
    const seen = new Set<string>();
    const urls: string[] = [];
    const add = (candidate: string) => {
      const url = candidate.trim();
      if (!url || seen.has(url)) return;
      seen.add(url);
      urls.push(url);
    };
    for (const line of value.split(/\n/)) {
      for (const segment of line.split(",")) {
        for (const field of segment.trim().split(/\s+/)) {
          add(field);
        }
      }
    }
    return urls;
  }

  function askRemoveInstalledMod(mod: InstalledMod) {
    confirmation = {
      title: "Uninstall mod",
      message: `Remove ${mod.name} from every profile and delete its installed files.`,
      detail: `${mod.source_game_domain}/mods/${mod.source_mod_id}/files/${mod.source_file_id}`,
      confirmLabel: "Uninstall Mod",
      danger: true,
      run: () => removeInstalledMod(mod)
    };
  }

  async function captureInstallURL(url: string, profileID: number, reason = "captured-install") {
    if (!selectedGame) return null;
    bulkCaptureMessage = "";
    const response = await apiFetch("/api/captured-installs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url, steam_app_id: selectedGame.app_id, profile_id: profileID })
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    const result = await response.json();
    if (result.job) upsertJob(result.job);
    if (result.resolved) {
      resolvedCapture = `${result.resolved.catalog}:${result.resolved.game_domain || result.resolved.steam_app_id}/mods/${result.resolved.mod_id}${result.resolved.file_id ? `/files/${result.resolved.file_id}` : ""}`;
    }
    await refreshJobsAndSelectedGame(reason, true);
    return result;
  }

  async function captureBulkInstallURLs(urls: string[], profileID: number) {
    if (!selectedGame || urls.length === 0) return;
    bulkCaptureMessage = "";
    captureBrowserPrompt = null;
    const response = await apiFetch("/api/captured-installs/bulk", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        urls,
        steam_app_id: selectedGame.app_id,
        profile_id: profileID,
        source: "web-bulk-capture"
      })
    });
    if (!response.ok) {
      throw new Error(await response.text());
    }
    const result: BulkCapturedInstallResult = await response.json();
    for (const item of result.items ?? []) {
      if (item.job) upsertJob(item.job);
    }
    const failed = (result.items ?? []).filter((item) => !item.ok);
    const browserRequired = result.browser_required ?? 0;
    const duplicateCount = (result.items ?? []).filter((item) => item.duplicate).length;
    const parts = [`${result.accepted} of ${result.total} links added`];
    if (duplicateCount > 0) parts.push(`${duplicateCount} duplicate${duplicateCount === 1 ? "" : "s"} reused`);
    if (browserRequired > 0) parts.push(`${browserRequired} need the Deck browser`);
    if (result.failed > 0) parts.push(`${result.failed} failed`);
    bulkCaptureMessage = parts.join(" · ");
    if (failed.length > 0) {
      captureURL = failed.map((item) => item.url).join("\n");
      error = failed.slice(0, 3).map((item) => item.error || `${item.url} failed`).join("\n");
    } else {
      captureURL = "";
      if (browserRequired > 0) {
        const browserItem = (result.items ?? []).find((item) => item.browser_required && isHTTPProviderPage(item.url));
        if (browserRequired === 1 && browserItem) {
          captureBrowserPrompt = {
            url: browserItem.url,
            source: "web-bulk-browser-required",
            title: "DMM Nexus Download",
            message: "This Nexus page needs the Deck browser. Click Nexus Mod Manager Download there to capture the generated link."
          };
          error = captureBrowserPrompt.message;
        } else {
          error = "Some Nexus links require the Deck browser flow. Open those mod pages from DMM one at a time, then click Nexus Mod Manager Download on each Nexus page.";
        }
      }
    }
    await refreshJobsAndSelectedGame("captured-install-bulk", true);
  }

  async function resolveCapturedInstall() {
    if (!selectedGame || !captureURL.trim()) return;
    error = "";
    bulkCaptureMessage = "";
    captureBrowserPrompt = null;
    const requestedURLs = captureInputURLs(captureURL);
    if (requestedURLs.length > 1) {
      try {
        await captureBulkInstallURLs(requestedURLs, selectedInstallProfileID());
      } catch (err) {
        error = err instanceof Error ? err.message : String(err);
      }
      return;
    }
    const requestedURL = requestedURLs[0] ?? captureURL;
    const targetProfileID = selectedInstallProfileID();
    if (isNexusHTTPPage(requestedURL)) {
      captureBrowserPrompt = {
        url: requestedURL,
        source: "web-paste-nexus-page",
        title: "DMM Nexus Download",
        message: "Opening this Nexus page on the Steam Deck. Click Nexus Mod Manager Download there to add it to DMM."
      };
      await openCapturePromptOnDeck(captureBrowserPrompt);
      return;
    }
    try {
      const result = await captureInstallURL(requestedURL, targetProfileID, "captured-install-url");
      if (!result) return;
      if (result.job?.status === "failed") {
        error = result.job.message || "Unable to add this mod link.";
        return;
      }
      if (result.browser_required && isHTTPProviderPage(requestedURL)) {
        captureBrowserPrompt = {
          url: result.resolved?.source_url || requestedURL,
          source: "web-paste-browser-required",
          title: "DMM Nexus Download",
          message: "Opening this page on the Steam Deck. Click Nexus Mod Manager Download there to add it to DMM."
        };
        await openCapturePromptOnDeck(captureBrowserPrompt);
      } else if (result.resolved?.catalog === "nexus" && !result.resolved?.nxm_key) {
        captureBrowserPrompt = {
          url: result.resolved?.source_url || requestedURL,
          source: "web-paste-nexus-page",
          title: "DMM Nexus Download",
          message: "Nexus page links need the Deck browser. Click Nexus Mod Manager Download there to capture the generated link."
        };
        error = captureBrowserPrompt.message;
      } else {
        captureURL = "";
        captureBrowserPrompt = null;
        bulkCaptureMessage = result.download_started ? "Mod link added; downloading archive now." : result.job?.message || "Mod link added.";
      }
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    }
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
      form.append("profile_id", String(selectedInstallProfileID()));
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/local-archives`, {
        method: "POST",
        body: form
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      localArchiveMessage = result.install_started ? "Upload received; installing archive." : "Upload received; finish the install from Action Center.";
      localArchiveFile = null;
      if (localArchiveInput) localArchiveInput.value = "";
      await refreshJobsAndSelectedGame("local-archive");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      localArchiveBusy = false;
    }
  }

  async function refreshDeckLocalArchives() {
    if (!selectedGame) return;
    error = "";
    deckLocalArchiveMessage = "";
    deckLocalArchiveBusy = true;
    try {
      const result = await getJSON<LocalArchiveList>(`/api/games/${selectedGame.app_id}/local-archives`);
      deckLocalArchiveRoots = result.roots ?? [];
      deckLocalArchives = result.files ?? [];
      deckLocalArchiveMessage = deckLocalArchives.length > 0
        ? `Found ${deckLocalArchives.length} archive file${deckLocalArchives.length === 1 ? "" : "s"} on the Deck.`
        : "No supported archive files found in Deck Downloads.";
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      deckLocalArchiveBusy = false;
    }
  }

  async function browseDeckArchiveFolder(path = deckArchiveBrowserPath) {
    if (!selectedGame) return;
    error = "";
    deckLocalArchiveMessage = "";
    deckLocalArchiveBusy = true;
    try {
      const params = new URLSearchParams();
      if (path.trim()) params.set("path", path.trim());
      const result = await getJSON<LocalArchiveBrowseList>(`/api/games/${selectedGame.app_id}/local-archives/browse${params.toString() ? `?${params.toString()}` : ""}`);
      deckLocalArchiveRoots = result.roots ?? [];
      deckArchiveBrowserEntries = result.entries ?? [];
      deckArchiveBrowserPath = result.current_path ?? "";
      deckArchiveBrowserParentPath = result.parent_path ?? "";
      deckArchivePathInput = result.current_path ?? "";
      deckLocalArchives = deckArchiveBrowserEntries.filter((entry) => entry.kind === "file").map((entry) => ({
        path: entry.path,
        name: entry.name,
        extension: entry.extension ?? "",
        bytes: entry.bytes ?? 0,
        root: entry.root ?? "",
        modified_at: entry.modified_at ?? ""
      }));
      deckLocalArchiveMessage = deckArchiveBrowserEntries.length > 0
        ? `Opened ${result.current_path || "Deck Downloads"}.`
        : "No folders or supported archives found here.";
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      deckLocalArchiveBusy = false;
    }
  }

  async function toggleDeckArchiveBrowser() {
    deckArchiveBrowserOpen = !deckArchiveBrowserOpen;
    if (deckArchiveBrowserOpen) {
      await browseDeckArchiveFolder("");
    }
  }

  async function importDeckLocalArchive(file: { path: string; name: string }) {
    if (!selectedGame || !file.path || busyDeckLocalArchivePath) return;
    error = "";
    deckLocalArchiveMessage = "";
    busyDeckLocalArchivePath = file.path;
    try {
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/local-archives/import`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          path: file.path,
          profile_id: selectedInstallProfileID(),
          source: "web-deck-local-file"
        })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      deckLocalArchiveMessage = result.install_started ? "Deck archive imported; installing archive." : "Deck archive imported; finish the install from Action Center.";
      await refreshJobsAndSelectedGame("deck-local-archive");
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      busyDeckLocalArchivePath = "";
    }
  }

  function externalModCandidateKey(candidate: ExternalModCandidate) {
    return `${candidate.adoption_id}\n${candidate.path}`;
  }

  function externalModAdoptionGroups() {
    const grouped = new Map<string, ExternalModCandidate[]>();
    for (const candidate of externalModCandidates) {
      const key = candidate.adoption_id;
      grouped.set(key, [...(grouped.get(key) ?? []), candidate]);
    }
    return Array.from(grouped.entries()).map(([adoptionID, candidates]) => ({
      adoptionID,
      candidates,
      modType: candidates[0]?.mod_type ?? "",
      deleteOriginal: Boolean(candidates[0]?.delete_original),
      rootPath: candidates[0]?.root_path ?? ""
    }));
  }

  function selectedExternalCandidates(adoptionID: string) {
    return externalModCandidates.filter((candidate) => candidate.adoption_id === adoptionID && selectedExternalModPaths[externalModCandidateKey(candidate)]);
  }

  function toggleExternalModCandidate(candidate: ExternalModCandidate, checked: boolean) {
    const key = externalModCandidateKey(candidate);
    selectedExternalModPaths = { ...selectedExternalModPaths, [key]: checked };
  }

  async function refreshExternalModCandidates() {
    if (!selectedGame) return;
    error = "";
    externalModAdoptionMessage = "";
    externalModAdoptionBusy = true;
    try {
      const result = await getJSON<ExternalModList>(`/api/games/${selectedGame.app_id}/external-mods`);
      externalModCandidates = result.items ?? [];
      selectedExternalModPaths = {};
      externalModAdoptionMessage = externalModCandidates.length > 0
        ? `Found ${externalModCandidates.length} unmanaged mod candidate${externalModCandidates.length === 1 ? "" : "s"}.`
        : "No extension-recognized unmanaged mods found.";
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      externalModAdoptionBusy = false;
    }
  }

  async function toggleExternalModAdoption() {
    externalModAdoptionOpen = !externalModAdoptionOpen;
    if (externalModAdoptionOpen) {
      await refreshExternalModCandidates();
    }
  }

  async function adoptExternalMods(adoptionID: string) {
    if (!selectedGame || externalModAdoptionBusy) return;
    const selected = selectedExternalCandidates(adoptionID);
    if (selected.length === 0) {
      externalModAdoptionMessage = "Select at least one unmanaged mod first.";
      return;
    }
    error = "";
    externalModAdoptionMessage = "";
    externalModAdoptionBusy = true;
    try {
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/external-mods/adopt`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          adoption_id: adoptionID,
          paths: selected.map((candidate) => candidate.path),
          profile_id: selectedInstallProfileID()
        })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: { imported?: InstalledMod[] } = await response.json();
      externalModAdoptionMessage = `Imported ${result.imported?.length ?? selected.length} unmanaged mod${(result.imported?.length ?? selected.length) === 1 ? "" : "s"} into DMM staging.`;
      await refreshSelectedGame({ refreshPreview: true, reason: "external-mod-adoption" });
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      externalModAdoptionBusy = false;
    }
  }

  function selectedNexusDomain() {
    const domains = selectedNexusDomains();
    if (domains.includes(nexusBrowseDomain)) return nexusBrowseDomain;
    return domains[0] ?? "";
  }

  function selectedNexusDomains() {
    return (selectedGame?.nexus_domains ?? []).map((domain) => domain.trim().toLowerCase()).filter(Boolean);
  }

  function nexusDomainLabel(domain: string) {
    const cleaned = String(domain || "")
      .trim()
      .replace(/([a-z])([0-9])/gi, "$1 $2")
      .replace(/([0-9])([a-z])/gi, "$1 $2")
      .replace(/[-_]+/g, " ");
    if (!cleaned) return "Nexus";
    return cleaned.replace(/\b\w/g, (value) => value.toUpperCase());
  }

  function selectedGameMetadataOnly() {
    return selectedGame?.extension?.coverage === "metadata_only";
  }

  function exploreSourceOptions() {
    if (selectedGameMetadataOnly()) return [];
    return catalogs.filter((catalog) => {
      if (catalog.status !== "ready" || (!catalog.search && !catalog.browse)) return false;
      if (catalog.id === "nexus") return Boolean(selectedNexusDomain());
      return catalog.download || catalog.url_import;
    });
  }

  function selectedExploreSource() {
    const options = exploreSourceOptions();
    return options.find((catalog) => catalog.id === exploreSourceID) ?? options.find((catalog) => catalog.id === "nexus") ?? options[0] ?? null;
  }

  function selectedExploreSourceReady() {
    const source = selectedExploreSource();
    return Boolean(source && source.status === "ready");
  }

  function selectedExploreSourceBrowseReady() {
    const source = selectedExploreSource();
    if (!source || !source.search || source.status !== "ready") return false;
    if (source.id === "nexus") return Boolean(selectedNexusDomain());
    return true;
  }

  function selectedCatalogGameID(sourceID: string) {
    sourceID = sourceID.trim().toLowerCase().replaceAll("-", "_");
    const hint = selectedGame?.extension?.catalog_sources?.find((item) => item.catalog?.trim().toLowerCase().replaceAll("-", "_") === sourceID);
    return hint?.game_id?.trim() || "";
  }

  function selectedExtensionSourceNote() {
    const source = selectedGame?.extension?.sources?.find((item) => (item.name || item.url));
    if (!source) return "";
    if (source.name && source.url) return `${source.name}: ${source.url}`;
    return source.name || source.url || "";
  }

  function selectedExtensionSources() {
    if (selectedGame?.extension?.coverage !== "metadata_only") return [];
    return selectedGame.extension.sources?.filter((item) => (item.name || item.url)) ?? [];
  }

  function nexusModURL(mod: CatalogModResult) {
    if (mod.url) return mod.url;
    const domain = selectedNexusDomain();
    return `https://www.nexusmods.com/${encodeURIComponent(domain)}/mods/${mod.mod_id}`;
  }

  function catalogResultURL(mod: CatalogModResult) {
    if (mod.url) return mod.url;
    const source = selectedExploreSource();
    if (source?.id === "nexus") return nexusModURL(mod);
    return "";
  }

  function catalogResultSource(mod: CatalogModResult) {
    return mod.source_tag || mod.catalog || selectedExploreSource()?.source_tag || selectedExploreSource()?.id || "unknown";
  }

  function catalogResultActionLabel(mod: CatalogModResult) {
    if (catalogResultSource(mod) === "nexus") return "Open on Deck";
    return "Add Mod";
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

  function jobDownloadProgress(job: Job) {
    const payload = job.payload ?? {};
    const written = Number(payload.download_bytes_written ?? 0);
    const total = Number(payload.download_total_bytes ?? 0);
    const rate = Number(payload.download_rate_bytes_per_second ?? 0);
    const status = String(payload.download_status ?? "");
    if (!status && (!Number.isFinite(written) || written <= 0) && (!Number.isFinite(total) || total <= 0)) return null;
    const boundedTotal = Number.isFinite(total) && total > 0 ? total : 0;
    const boundedWritten = Number.isFinite(written) && written > 0 ? written : 0;
    const percent = boundedTotal > 0 ? Math.min(100, Math.max(0, (boundedWritten / boundedTotal) * 100)) : 0;
    const rateLabel = Number.isFinite(rate) && rate > 0 ? ` · ${formatBytes(rate)}/s` : "";
    const indeterminate = boundedTotal <= 0 && !["downloaded", "completed"].includes(status);
    const label =
      boundedTotal > 0
        ? `${formatBytes(boundedWritten)} / ${formatBytes(boundedTotal)}${rateLabel}`
        : boundedWritten > 0
          ? `${formatBytes(boundedWritten)} downloaded${rateLabel}`
          : "Starting download...";
    return {
      written: boundedWritten,
      total: boundedTotal,
      percent,
      indeterminate,
      barWidth: indeterminate ? 36 : boundedTotal > 0 ? percent : 100,
      label
    };
  }

  function jobIssueTitle(job: Job) {
    return String(job.payload?.issue_title ?? "").trim();
  }

  function jobIssueMessage(job: Job) {
    return String(job.payload?.issue_message ?? "").trim();
  }

  function jobIssueDetails(job: Job) {
    return parseJobStringList(job.payload?.issue_details_json);
  }

  function jobIssueActions(job: Job) {
    return parseJobStringList(job.payload?.issue_actions_json);
  }

  function parseJobStringList(value: string | undefined) {
    const raw = String(value ?? "").trim();
    if (!raw) return [];
    try {
      const parsed = JSON.parse(raw);
      if (!Array.isArray(parsed)) return [];
      return parsed.map((item) => String(item ?? "").trim()).filter(Boolean);
    } catch {
      return [];
    }
  }

  async function searchNexusMods(nextSort = nexusSearchSort, nextWindow = nexusSearchTimeWindow) {
    if (!selectedGame) return;
    const source = selectedExploreSource();
    if (!source) return;
    nexusSearchBusy = true;
    nexusSearchError = "";
    nexusSearchMessage = "";
    try {
      const params = new URLSearchParams({
        q: nexusSearchQuery,
        domain: source.id === "nexus" ? selectedNexusDomain() : selectedCatalogGameID(source.id),
        sort: nextSort,
        time_window: nextWindow,
        count: "20",
        offset: "0",
        vortex_only: nexusSearchVortexOnly ? "true" : "false"
      });
      const endpoint = source.id === "nexus"
        ? `/api/games/${selectedGame.app_id}/nexus/mods?${params.toString()}`
        : `/api/games/${selectedGame.app_id}/catalogs/${encodeURIComponent(source.id)}/mods?${params.toString()}`;
      const result = await getJSON<{ mods: CatalogModResult[]; total_count: number }>(endpoint);
      nexusSearchResults = result.mods ?? [];
      nexusSearchTotal = result.total_count ?? nexusSearchResults.length;
      if (nexusSearchResults.length === 0) {
        nexusSearchError = `No ${source.name} mods matched this search.`;
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

  function resetNexusSearchResults() {
    nexusSearchResults = [];
    nexusSearchTotal = 0;
    nexusSearchError = "";
    nexusSearchMessage = "";
  }

  async function openCatalogMod(mod: CatalogModResult) {
    if (!selectedGame) return;
    const source = selectedExploreSource();
    const url = catalogResultURL(mod);
    if (!source || !url) {
      nexusSearchError = "This result does not include a provider URL.";
      return;
    }
    nexusSearchError = "";
    nexusSearchMessage = "";
    busyCatalogOpenModID = mod.mod_id;
    try {
      if (source.id !== "nexus") {
        const result = await captureInstallURL(url, selectedInstallProfileID(), `catalog-search-${source.id}`);
        if (result?.job?.status === "failed") {
          nexusSearchError = result.job.message || "Unable to add this provider mod.";
          return;
        }
        nexusSearchMessage = result?.download_started ? `${source.name} mod added; downloading archive now.` : result?.job?.message || `${source.name} mod added.`;
        return;
      }
      const response = await apiFetch("/api/decky/browser/open", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          url,
          steam_app_id: selectedGame.app_id,
          profile_id: selectedInstallProfileID(),
          source: "web-catalog-search:nexus",
          title: `${mod.name} - Nexus Mods`
        })
      });
      if (!response.ok) {
        throw new Error(await response.text());
      }
      nexusSearchMessage = "Opening this Nexus page on the Steam Deck. Click Nexus Mod Manager Download there to add it to DMM.";
    } catch (err) {
      nexusSearchError = err instanceof Error ? err.message : String(err);
    } finally {
      busyCatalogOpenModID = null;
    }
  }

  async function clearCapturedInstallActions() {
    error = "";
    const response = await apiFetch("/api/captured-installs", { method: "DELETE" });
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/install-candidates`, { method: "DELETE" });
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/installer-choice-presets/${preset.id}`, { method: "DELETE" });
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

  function fomodGroupIsText(group: FomodGroup) {
    const type = fomodGroupType(group);
    return type === "text" || type === "textinput";
  }

  function visibleFomodSteps(installer: FomodInstaller) {
    return (installer.steps ?? []).filter((step) => step.visible !== false);
  }

  function installerRequiresSelections(installer: FomodInstaller) {
    return visibleFomodSteps(installer).some((step) => (step.groups ?? []).some((group) => fomodGroupIsText(group) || (group.plugins ?? []).length > 0));
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

  function candidateGroupTextValue(candidate: InstallCandidate, group: FomodGroup) {
    return candidateCurrentSelections(candidate)[group.id]?.[0] ?? "";
  }

  function setCandidateTextSelection(candidate: InstallCandidate, group: FomodGroup, value: string, save: boolean) {
    const current = candidateCurrentSelections(candidate);
    const next = { ...current, [group.id]: value.trim() === "" ? [] : [value] };
    candidateSelections = { ...candidateSelections, [candidate.id]: next };
    if (save) void saveCandidateSelections(candidate, next);
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
    if (fomodGroupIsText(group)) {
      return !group.required || candidateGroupTextValue(candidate, group).trim() !== "";
    }
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
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/install-candidates/${candidate.id}/choices`, {
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
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/install-candidates/${candidate.id}/apply`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ selections, profile_id: candidateInstallProfileID(candidate) })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      if (result.candidate) replaceInstallCandidate(result.candidate);
      if (result.job?.status === "failed") {
        error = result.job.message || "Unable to apply installer choices.";
        await refreshJobsAndSelectedGame("installer-choice-apply-failed", true);
        return;
      }
      if (result.mod) {
        installedMods = [result.mod, ...installedMods.filter((mod) => mod.id !== result.mod.id)];
        installCandidates = installCandidates.filter((item) => item.id !== candidate.id);
        globalInstallCandidates = globalInstallCandidates.filter((item) => item.id !== candidate.id);
        candidateSelections = Object.fromEntries(Object.entries(candidateSelections).filter(([id]) => Number(id) !== candidate.id));
        candidateStepIndices = Object.fromEntries(Object.entries(candidateStepIndices).filter(([id]) => Number(id) !== candidate.id));
      }
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
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/install-candidates/${candidate.id}/retry`, { method: "POST" });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result = await response.json();
      if (result.job) upsertJob(result.job);
      if (result.candidate) replaceInstallCandidate(result.candidate);
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
      const response = await apiFetch(`/api/jobs/${job.id}/cancel`, { method: "POST" });
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
      const response = await apiFetch(`/api/captured-installs/${action.id}/install`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ profile_id: actionInstallProfileID(action) })
      });
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
      const response = await apiFetch(`/api/captured-installs/${action.id}/retry`, { method: "POST" });
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
      const response = await apiFetch(`/api/workshop/actions/${action.id}/retry`, { method: "POST" });
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/deploy/preview`);
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/deploy/preview`);
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/deploy`, { method: "POST" });
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
    const response = await apiFetch(`/api/profiles/${selectedProfile.id}/conflicts/winner`, {
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
    const response = await apiFetch(`/api/profiles/${selectedProfile.id}/conflicts/winner?target_path=${encodeURIComponent(target.target_path)}`, {
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/deploy`, { method: "DELETE" });
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/reset`, { method: "POST" });
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
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/deploy/repair`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    if (deployPlan) await previewDeploy();
  }

  async function previewRestoreDeploymentPoint(deployment: DeploymentHistoryItem) {
    if (!selectedGame) return;
    error = "";
    restorePointPreviewBusy = deployment.id;
    try {
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/deploy/history/${deployment.id}/preview`);
      if (!response.ok) {
        error = await response.text();
        return;
      }
      restorePointPreview = await response.json();
    } finally {
      restorePointPreviewBusy = 0;
    }
  }

  async function restoreDeploymentPoint(deployment: DeploymentHistoryItem) {
    if (!selectedGame || deployment.active) return;
    error = "";
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/deploy/history/${deployment.id}/restore`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    restorePointPreview = null;
    await refreshSelectedGame({ refreshPreview: deployPlan !== null });
  }

  function askRestoreDeploymentPoint(deployment: DeploymentHistoryItem) {
    if (!selectedGame || deployment.active) return;
    const preview = restorePointPreview?.deployment_id === deployment.id ? restorePointPreview : null;
    const summary = preview?.summary;
    const detail = summary
      ? `${new Date(deployment.created_at).toLocaleString()} · ${deployment.profile_name}. Restore delta: ${summary.add} add, ${summary.replace} update, ${summary.remove} remove, ${summary.conflicts} conflict${summary.conflicts === 1 ? "" : "s"}. DMM touches only files from its deployment manifests.`
      : `${new Date(deployment.created_at).toLocaleString()} · ${deployment.profile_name} · ${deployment.file_count} file${deployment.file_count === 1 ? "" : "s"} · ${deploymentPointDelta(deployment)}. DMM removes newer managed files that are not part of this point.`;
    confirmation = {
      title: "Restore deployment point",
      message: `DMM will restore ${selectedGame.name} to the selected deployment point.`,
      detail,
      confirmLabel: "Restore Point",
      run: () => restoreDeploymentPoint(deployment)
    };
  }

  async function applyLaunchSetup() {
    if (!selectedGame || !launchSetupAvailable) return;
    error = "";
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/launch/apply`, { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    if (result.job) upsertJob(result.job);
    gameLaunchStatus = result.status;
    await refreshSelectedGame({ refreshPreview: deployPlan !== null });
  }

  async function acquireRuntimeRequirement(requirement: RuntimeRequirement) {
    if (!selectedGame || !runtimeRequirementCanAcquire(requirement)) return;
    error = "";
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/requirements/${encodeURIComponent(requirement.id)}/acquire`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ profile_id: selectedInstallProfileID() })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result: { job?: Job; result?: { job?: Job } } = await response.json();
    if (result.job) upsertJob(result.job);
    else if (result.result?.job) upsertJob(result.result.job);
    await refreshJobsAndSelectedGame("runtime-requirement-acquire", true);
  }

  async function recoverDownloads() {
    if (!selectedGame) return;
    error = "";
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/mods/recover-downloads`, { method: "POST" });
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
      const response = await apiFetch(
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
      const response = await apiFetch(`/api/games/${encodeURIComponent(selectedGame.app_id)}/workshop/order`, {
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
    const activeCandidateIDs = new Set([...installCandidates, ...globalInstallCandidates].map((candidate) => candidate.id));
    busyInstallCandidates = Object.fromEntries(Object.entries(busyInstallCandidates).filter(([candidateID]) => activeCandidateIDs.has(Number(candidateID))));
    candidateSelections = Object.fromEntries(Object.entries(candidateSelections).filter(([candidateID]) => activeCandidateIDs.has(Number(candidateID))));
    candidateStepIndices = Object.fromEntries(Object.entries(candidateStepIndices).filter(([candidateID]) => activeCandidateIDs.has(Number(candidateID))));
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
    if (job.type === "extension-tool-action" && job.status === "failed") return true;
    if (job.type === "captured-install" && job.status === "failed") return true;
    return !["completed", "failed", "canceled"].includes(job.status);
  }

  function jobCancelLabel(job: Job) {
    return job.type === "extension-notice" ? "Dismiss" : "Cancel";
  }

  function getDeployableActions(plan: DeployPlan | null) {
    return plan?.actions.filter((action) => action.operation !== "keep" && action.operation !== "skip") ?? [];
  }

  function sortDrawerGames(items: Game[]) {
    return [...items].sort((a, b) => {
      const favoriteDelta = Number(favoriteGameIDs.has(b.app_id)) - Number(favoriteGameIDs.has(a.app_id));
      if (favoriteDelta !== 0) return favoriteDelta;
      const readyDelta = Number(gameManageReady(b)) - Number(gameManageReady(a));
      if (readyDelta !== 0) return readyDelta;
      const supportedDelta = Number(gameHasExtension(b)) - Number(gameHasExtension(a));
      if (supportedDelta !== 0) return supportedDelta;
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

  function gameHasExtension(game: Game) {
    return Boolean(game.extension?.supported);
  }

  function gameManageReady(game: Game) {
    const coverage = game.extension?.coverage;
    return coverage === "installer" || coverage === "workshop_only";
  }

  function gameCapabilityBadges(game: Game) {
    const extension = game.extension;
    if (!extension?.supported) return [{ label: "Unsupported", className: "unsupported" }];
    const coverage = extension.coverage ?? "metadata_only";
    const badges = [{ label: extension.coverage_label ?? "DMM", className: `coverage-${coverage}` }];
    if (extension.nexus) badges.push({ label: "Nexus", className: "nexus" });
    if (extension.steam_workshop) badges.push({ label: "Workshop", className: "workshop" });
    if (coverage === "installer" && (extension.installers || extension.installer_choices)) badges.push({ label: "Installers", className: "installers" });
    if (extension.load_order || extension.plugin_activation) badges.push({ label: "Load Order", className: "load-order" });
    if (extension.launch_tools) badges.push({ label: "Launch", className: "launch" });
    return badges;
  }

  function gameImage(appID: string) {
    return `https://cdn.cloudflare.steamstatic.com/steam/apps/${appID}/header.jpg`;
  }

  function settingsTitle(page: SettingsPage) {
    if (page === "jobs") return "Jobs";
    if (page === "install") return "Install";
    if (page === "sources") return "Sources";
    if (page === "game-stores") return "Game Stores";
    if (page === "extensions") return "Extension Settings";
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
    const appID = jobAppID(job);
    if (appID) {
      const exact = games.find((game) => game.app_id === appID);
      if (exact) return exact;
    }
    return games.find((game) => actionMatchesGame(job, game)) ?? null;
  }

  function gameForInstallCandidate(candidate: InstallCandidate) {
    return games.find((game) => game.app_id === candidate.steam_app_id) ?? null;
  }

  function jobHasInstallCandidateReview(job: Job) {
    if (["queued", "waiting", "running"].includes(job.status)) return false;
    return globalInstallCandidates.some((candidate) => jobMatchesInstallCandidate(job, candidate));
  }

  function jobMatchesInstallCandidate(job: Job, candidate: InstallCandidate) {
    const payload = job.payload ?? {};
    const candidateID = payload.candidate_id;
    if (candidateID && String(candidate.id) === candidateID) return true;
    return (
      (job.app_id || payload.app_id || "") === candidate.steam_app_id &&
      (job.catalog || payload.catalog || "") === candidate.catalog &&
      (payload.game_domain || "") === candidate.source_game_domain &&
      (payload.mod_id || "") === candidate.source_mod_id &&
      (payload.file_id || "") === candidate.source_file_id
    );
  }

  function installerChoiceJobHasCandidateReview(job: Job) {
    if (job.type !== "installer-choice") return false;
    const candidateID = job.payload?.candidate_id;
    if (!candidateID) return false;
    return globalInstallCandidates.some((candidate) => String(candidate.id) === candidateID);
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
    if (job.type === "installer-choice") {
      const candidateID = job.payload?.candidate_id;
      const candidate = candidateID ? globalInstallCandidates.find((item) => String(item.id) === candidateID) : null;
      if (candidate) {
        await openInstallCandidate(candidate);
        return;
      }
      error = "This installer-choice action no longer has stored choices. DMM canceled it; download the mod again.";
      await cancelJob(job);
      return;
    }
    const game = gameForJob(job);
    if (!game) return;
    await selectGame(game);
    openGameModule("actions");
  }

  function jobMatchesGame(job: Job, game: Game) {
    const appID = jobAppID(job);
    if (appID && appID === game.app_id) return true;
    const domain = job.payload?.game_domain?.toLowerCase();
    if (!domain) return false;
    return nexusDomainsForGame(game).includes(domain);
  }

  function nexusDomainsForGame(game: Game) {
    return (game.nexus_domains ?? []).map((domain) => domain.toLowerCase()).filter(Boolean);
  }

  function modStatusText(mod: InstalledMod) {
    if (mod.status === "installed") return mod.enabled ? "Enabled" : "Installed";
    return mod.status;
  }

  function modProfileStateText(mod: InstalledMod) {
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
    if (source === "extension") return "Extension";
    if (source === "nexus") return "Nexus";
    if (source === "steam_workshop" || source === "steam-workshop" || source === "workshop") return "Steam Workshop";
    if (source === "thunderstore") return "Thunderstore";
    if (source === "modrinth") return "Modrinth";
    if (source === "gamebanana") return "GameBanana";
    if (source === "modio" || source === "mod.io") return "mod.io";
    if (source === "curseforge") return "CurseForge";
    if (source === "moddb") return "ModDB";
    if (source === "itchio" || source === "itch.io") return "itch.io";
    if (source === "github" || source === "github_releases") return "GitHub";
    if (source === "direct") return "Direct";
    if (source === "local") return "Local";
    if (source === "native") return "Native";
    return source ? source.replace(/[_-]+/g, " ") : "Unknown";
  }

  function sourceClass(catalog: string | undefined) {
    const source = (catalog ?? "").trim().toLowerCase().replace(/_/g, "-");
    if (source === "extension") return "source-extension";
    if (source === "nexus") return "source-nexus";
    if (source === "steam-workshop" || source === "workshop") return "source-workshop";
    if (source === "thunderstore") return "source-thunderstore";
    if (source === "modrinth") return "source-modrinth";
    if (source === "gamebanana") return "source-gamebanana";
    if (source === "modio" || source === "mod.io") return "source-modio";
    if (source === "curseforge") return "source-curseforge";
    if (source === "moddb") return "source-moddb";
    if (source === "itchio" || source === "itch.io") return "source-itchio";
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

  function showPrimaryModUpdate(update: ModUpdate | undefined) {
    return update?.status === "available" || update?.status === "error";
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

  function extensionNoticeTool(action: Job) {
    return action.payload?.tool_name || action.payload?.tool_id || "";
  }

  function extensionNoticeActionLabel(action: Job) {
    return action.payload?.action_label || (extensionNoticeTool(action) ? `Open ${extensionNoticeTool(action)}` : "");
  }

  function extensionNoticeHelpURL(action: Job) {
    const value = action.payload?.help_url ?? "";
    return value.startsWith("http://") || value.startsWith("https://") ? value : "";
  }

  function openExtensionNoticeHelp(action: Job) {
    const url = extensionNoticeHelpURL(action);
    if (url) window.open(url, "_blank", "noopener,noreferrer");
  }

  function actionNextStep(action: Job) {
    if (action.type === "extension-tool-action") {
      const tool = extensionNoticeTool(action);
      if (action.status === "waiting" || action.status === "queued") return `Waiting for Decky to launch ${tool || "the extension tool"} through Steam.`;
      if (action.status === "running") return `Decky is launching ${tool || "the extension tool"} through Steam.`;
      if (action.status === "failed") return `Decky could not launch ${tool || "the extension tool"}. Make sure Steam is running in Gaming Mode, then retry from Decky or cancel this action.`;
      return "This extension tool action is retained in job history for diagnostics.";
    }
    if (action.type === "extension-notice") {
      const tool = extensionNoticeTool(action);
      if (tool) return `${tool} is required for this extension note. Review the linked tool page or dismiss this action when the manual step is handled or not needed.`;
      return "Review this extension note before launching the game. Dismiss it when the manual step is handled or not needed.";
    }
    if (action.type === "steam-workshop-action") {
      if (action.status === "waiting" || action.status === "queued") return "Waiting for Decky to apply this Steam Workshop change through Steam.";
      if (action.status === "running") return "Decky is applying this Steam Workshop change through Steam.";
      if (action.status === "failed") return "Decky could not apply this Steam Workshop change. Make sure the Deck is online with Steam running, then retry or cancel.";
      return "This Workshop action is retained in job history for diagnostics.";
    }
    if (action.type === "installer-choice") {
      return "Choose installer options to finish adding this mod to the selected profile.";
    }
    if (isModUpdateAction(action)) {
      if (action.status === "waiting") return "Install this downloaded update to replace the current cached version for this profile. The mod keeps its current on/off state.";
      if (action.status === "running" || action.status === "queued") return "DMM is downloading or installing this update through the normal safe profile pipeline.";
      if (action.status === "failed") return "The update was not installed. Retry from the cached action when available, or clear it if this update is no longer needed.";
      return "This update action is retained in job history for diagnostics.";
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
    if (action.type === "extension-notice" && action.status === "waiting") return "Needs review";
    if (action.type === "extension-tool-action" && (action.status === "waiting" || action.status === "queued")) return "Waiting for Decky";
    if (action.type === "extension-tool-action" && action.status === "running") return "Launching";
    if (action.type === "steam-workshop-action" && (action.status === "waiting" || action.status === "queued")) return "Waiting for Decky";
    if (action.type === "installer-choice" && action.status === "waiting") return "Needs choices";
    if (isModUpdateAction(action) && action.status === "waiting") return "Ready to update";
    if (isModUpdateAction(action) && (action.status === "queued" || action.status === "running")) return "Updating";
    if (action.status === "waiting") return "Ready to install";
    if (action.status === "running") return "Processing";
    if (action.status === "queued") return "Queued";
    if (action.status === "failed") return "Failed";
    return action.status;
  }

  function isModUpdateAction(action: Job) {
    return action.type === "captured-install" && Boolean(action.payload?.installed_mod_id && action.payload?.update_to_file_id);
  }

  function capturedInstallPrimaryLabel(action: Job) {
    return isModUpdateAction(action) ? "Install Update" : "Install";
  }

  function modUpdateActionDetail(action: Job) {
    if (!isModUpdateAction(action)) return "";
    const from = action.payload?.update_from_file_id || "current";
    const to = action.payload?.update_to_file_id || "latest";
    return `Update ${from} -> ${to}`;
  }

  function activeDeploymentPoint() {
    return deploymentHistory.find((deployment) => deployment.active) ?? null;
  }

  function deploymentPointLabel(deployment: DeploymentHistoryItem) {
    if (deployment.active) return "Active deployment";
    if (deployment.status === "purged") return "Purged point";
    return "Restore point";
  }

  function deploymentPointDelta(deployment: DeploymentHistoryItem) {
    const active = activeDeploymentPoint();
    if (!active || active.id === deployment.id) return "current point";
    const delta = deployment.file_count - active.file_count;
    if (delta === 0) return "same file count as current";
    return `${delta > 0 ? "+" : ""}${delta} file${Math.abs(delta) === 1 ? "" : "s"} versus current`;
  }

  function actionSource(action: Job) {
    if (action.source_tag) return action.source_tag;
    if (action.catalog) return action.catalog;
    if (action.type === "extension-notice" || action.type === "extension-tool-action") return "extension";
    if (action.type === "steam-workshop-action") return "steam_workshop";
    return action.payload?.catalog;
  }

  function installerPresetScopeLabel(preset: InstallerChoicePreset) {
    if (preset.reuse_scope === "exact_file") return "Exact file only";
    return "Manual review";
  }

  function candidateStatusLabel(candidate: InstallCandidate) {
    if (candidate.status === "needs_choices") return "Needs choices";
    if (candidate.status === "blocked") return "Needs review";
    return candidate.status;
  }

  function gameInfoValue(value: unknown) {
    if (value === null || value === undefined) return "";
    if (typeof value === "string") return value;
    if (typeof value === "number" || typeof value === "boolean") return String(value);
    try {
      return JSON.stringify(value);
    } catch {
      return String(value);
    }
  }

  function executableGameExtensionActions() {
    return gameExtensionActions.filter((action) => {
      const status = (action.status || "ready").trim();
      const kind = (action.kind || "").trim();
      return status === "ready" && (kind === "open-directory" || kind === "open-path" || kind === "acquire-tool" || kind === "set-extension-setting");
    });
  }

  function extensionSurfaceActions() {
    return gameExtensionActions.filter((action) => {
      const status = (action.status || "ready").trim();
      const kind = (action.kind || "").trim();
      return status === "ready" && (kind === "page" || kind === "dialog" || kind === "api" || kind === "report");
    });
  }

  function extensionSurfaceLabel(action: GameExtensionAction) {
    const kind = (action.kind || action.action_target?.type || "surface").trim();
    if (kind === "page") return "Page";
    if (kind === "dialog") return "Dialog";
    if (kind === "api") return "API";
    if (kind === "report") return "Report";
    return kind;
  }

  function extensionSurfaceDescription(action: GameExtensionAction) {
    const scope = action.action_target?.scope || action.scope || "";
    if (scope) return `Source-backed ${extensionSurfaceLabel(action).toLowerCase()} surface: ${scope}.`;
    return `Source-backed ${extensionSurfaceLabel(action).toLowerCase()} surface.`;
  }

  async function resolveGameExtensionSurface(action: GameExtensionAction) {
    if (!selectedGame) return;
    error = "";
    extensionSurfaceMessage = "";
    extensionSurfaceBusyID = action.id;
    try {
      const response = await apiFetch(`/api/games/${selectedGame.app_id}/extension-actions/${encodeURIComponent(action.id)}/run`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ profile_id: selectedInstallProfileID() })
      });
      if (!response.ok) {
        error = await response.text();
        return;
      }
      const result: { surface?: { type?: string; scope?: string; message?: string } } = await response.json();
      const surface = result.surface;
      extensionSurfaceMessage = surface?.message || `${action.name || action.id} is available as ${surface?.scope || action.action_target?.scope || action.scope || "an extension surface"}.`;
    } finally {
      extensionSurfaceBusyID = "";
    }
  }

  async function runGameExtensionAction(action: GameExtensionAction) {
    if (!selectedGame) return;
    error = "";
    const response = await apiFetch(`/api/games/${selectedGame.app_id}/extension-actions/${encodeURIComponent(action.id)}/run`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ profile_id: selectedInstallProfileID() })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result: { job?: Job; result?: { job?: Job }; duplicate?: boolean } = await response.json();
    if (result.job) upsertJob(result.job);
    else if (result.result?.job) upsertJob(result.result.job);
    await refreshJobsAndSelectedGame("extension-action", true);
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

  function runtimeRequirementCanAcquire(requirement: RuntimeRequirement) {
    const acquisition = requirement.acquisition;
    if (!acquisition) return false;
    return Boolean(
      acquisition.url ||
      acquisition.source_mod_id ||
      acquisition.source_file_id ||
      acquisition.source_game ||
      acquisition.source_provider ||
      acquisition.catalog
    );
  }

  async function confirmCurrentAction() {
    if (!confirmation) return;
    const action = confirmation;
    confirmation = null;
    await action.run();
  }

  onMount(() => {
    initializeAPIAuth();
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

<main class="phone-shell" class:drawer-open={drawer !== null}>
  <header class="phone-topbar">
    <button type="button" class="plain-icon" aria-label="Open menu" disabled={authRejected} on:click={() => (drawer = "settings")}>☰</button>
    <div class="phone-title">
      <p>{selectedGame ? (surface === "game" ? selectedGame.name : title) : surface === "actions" ? "Running Now" : title}</p>
      <h1>{surface === "game" && selectedProfile ? selectedProfile.name : selectedGame?.name ?? "Decky Mod Manager"}</h1>
    </div>
    <button type="button" class="plain-icon" aria-label="Settings" disabled={authRejected} on:click={() => openSettings("overview")}>⚙</button>
  </header>

  {#if drawer === "settings" && !authRejected}
    <button type="button" class="phone-scrim" aria-label="Close menu" on:click={() => (drawer = null)}></button>
    <aside class="phone-drawer">
      <div class="drawer-brand">
        <div class="drawer-mark">D</div>
        <div>
          <p>Decky Mod Manager</p>
          <h2>Menu</h2>
        </div>
      </div>
      <p class="drawer-group">Work</p>
      <button type="button" class:active={surface === "actions"} on:click={openActionCenter}>Action Queue {#if globalActionCount > 0}<span>{globalActionCount}</span>{/if}</button>
      <button type="button" on:click={() => (drawer = "games")}>Games</button>
      <button type="button" class:active={surface === "settings" && activeSettingsPage === "jobs"} on:click={() => openSettings("jobs")}>Jobs</button>
      <p class="drawer-group">Manage</p>
      {#if selectedGame}
        <button type="button" class:active={surface === "game" && activeGameModule === "plugins"} on:click={() => { surface = "game"; activeGameModule = "plugins"; drawer = null; }}>Profile Mods</button>
        <button type="button" class:active={surface === "game" && activeGameModule === "profiles"} on:click={() => { surface = "game"; activeGameModule = "profiles"; drawer = null; }}>Profiles</button>
        <button type="button" class:active={surface === "game" && activeGameModule === "advanced"} on:click={() => { surface = "game"; activeGameModule = "advanced"; drawer = null; }}>Rollback</button>
        <button type="button" class:active={surface === "game" && activeGameModule === "actions"} on:click={() => { surface = "game"; activeGameModule = "actions"; drawer = null; }}>Game Actions</button>
        <button type="button" class:active={surface === "game" && activeGameModule === "review"} on:click={() => { surface = "game"; activeGameModule = "review"; drawer = null; }}>Review</button>
      {/if}
      <button type="button" class:active={surface === "settings" && activeSettingsPage === "sources"} on:click={() => openSettings("sources")}>Sources</button>
      <button type="button" class:active={surface === "settings" && activeSettingsPage === "extensions"} on:click={() => openSettings("extensions")}>Extensions</button>
      <p class="drawer-group">System</p>
      <button type="button" class:active={surface === "settings" && activeSettingsPage === "install"} on:click={() => openSettings("install")}>Install Behavior</button>
      <button type="button" class:active={surface === "settings" && activeSettingsPage === "game-stores"} on:click={() => openSettings("game-stores")}>Game Stores</button>
      <button type="button" class:active={surface === "settings" && activeSettingsPage === "nexus"} on:click={() => openSettings("nexus")}>Nexus</button>
    </aside>
  {/if}

  {#if drawer === "games" && !authRejected}
    <button type="button" class="phone-scrim" aria-label="Close games" on:click={() => (drawer = null)}></button>
    <aside class="phone-drawer game-menu">
      <div class="drawer-brand">
        <div class="drawer-mark">G</div>
        <div><p>Library</p><h2>Games</h2></div>
      </div>
      <input class="phone-search" bind:value={gameQuery} aria-label="Search games" placeholder="Search games" />
      <div class="phone-segments compact">
        <button type="button" class:active={gameSort === "recent"} on:click={() => setGameSort("recent")}>Recent</button>
        <button type="button" class:active={gameSort === "az"} on:click={() => setGameSort("az")}>A-Z</button>
        <button type="button" class:active={gameSort === "za"} on:click={() => setGameSort("za")}>Z-A</button>
      </div>
      <label class="inline-select"><span>Show</span><select bind:value={gameVisibility}><option value="manageable">Manage Ready</option><option value="extensions">DMM Extensions</option><option value="all">All Installed</option></select></label>
      <div class="drawer-scroll">
        {#each filteredGames as game}
          <button type="button" class="drawer-game-row" class:selected={selectedGame?.app_id === game.app_id} on:click={() => selectGame(game)}>
            <img src={gameImage(game.app_id)} alt="" loading="lazy" />
            <span><strong>{game.name}</strong><small>{game.extension?.coverage_label ?? stateLabel(game.state)}</small></span>
            <em>{isFavoriteGame(game.app_id) ? "★" : "›"}</em>
          </button>
        {/each}
      </div>
    </aside>
  {/if}

  {#if error && !authRejected}<section class="phone-alert">{error}</section>{/if}

  {#if authRejected}
    <section class="phone-empty"><p>Pairing Required</p><h2>Open the current Phone URL from Decky Mod Manager.</h2><button type="button" on:click={retryPairing}>Retry</button><button type="button" class="secondary" on:click={clearLocalPairingToken}>Clear Stored Pairing</button></section>
  {:else if loading}
    <section class="phone-empty"><h2>Loading...</h2></section>
  {:else if surface === "actions"}
    <section class="phone-content">
      {#if selectedGame}
        <button type="button" class="context-card" on:click={() => { surface = "game"; activeGameModule = "plugins"; }}>
          <img src={gameImage(selectedGame.app_id)} alt="" />
          <span><strong>{selectedProfile?.name ?? "Default Profile"}</strong><small>{installedMods.filter((mod) => mod.enabled).length} enabled, {installedMods.length} installed</small><span class="pills"><em class="ok">{deploymentStatus?.deployed ? "Deployed" : "Not deployed"}</em>{#if selectedGame.nexus_domains?.length}<em class="source">Nexus</em>{/if}</span></span>
          <b>›</b>
        </button>
      {/if}
      <section class="phone-card">
        <header><h2>Action Queue</h2><span>{globalActionCount} open</span></header>
        {#if visibleActionItems.length === 0 && visibleActionCenterCandidates.length === 0}
          <button type="button" class="phone-row" on:click={() => (drawer = "games")}><span><strong>No actions needed</strong><small>Choose a game or capture a mod link to get started.</small></span><b>›</b></button>
        {/if}
        {#each visibleActionItems as action}
          <button type="button" class="phone-row" class:failed={action.status === "failed"} on:click={() => openActionItem(action)}>
            <span><strong>{action.title}</strong><small>{action.message || actionNextStep(action)}</small><span class="pills"><em class="source">{sourceLabel(actionSource(action))}</em><em>{actionStatusLabel(action)}</em></span></span><b>›</b>
          </button>
        {/each}
        {#each visibleActionCenterCandidates as candidate}
          <button type="button" class="phone-row" on:click={() => openInstallCandidate(candidate)}>
            <span><strong>{candidate.name}</strong><small>{candidate.reason}</small><span class="pills"><em class="source">{sourceLabel(sourceForCandidate(candidate))}</em><em>{candidateStatusLabel(candidate)}</em></span></span><b>›</b>
          </button>
        {/each}
      </section>
      <input class="phone-search" bind:value={gameQuery} aria-label="Search games" placeholder="Search games, extensions, providers" />
      <div class="phone-segments"><button type="button" class:active={gameSort === "recent"} on:click={() => setGameSort("recent")}>Recent</button><button type="button" class:active={gameSort === "az"} on:click={() => setGameSort("az")}>A-Z</button><button type="button" class:active={gameSort === "za"} on:click={() => setGameSort("za")}>Z-A</button><button type="button" on:click={() => (gameVisibility = gameVisibility === "manageable" ? "all" : "manageable")}>{gameVisibility === "manageable" ? "Ready" : "All"}</button></div>
      <section class="phone-card">
        <header><h2>Recent Games</h2><span>{homeQuickGames.length} shown</span></header>
        {#each homeQuickGames as game}
          <button type="button" class="phone-row game" on:click={() => selectGame(game)}><img src={gameImage(game.app_id)} alt="" loading="lazy" /><span><strong>{game.name}</strong><small>{game.extension?.coverage_label ?? stateLabel(game.state)}</small></span><b>›</b></button>
        {/each}
      </section>
    </section>
  {:else if surface === "settings"}
    <section class="phone-content">
      <section class="phone-card"><header><h2>{settingsTitle(activeSettingsPage)}</h2><span>Settings</span></header>
        {#if activeSettingsPage === "overview"}
          <div class="metric-grid"><div><strong>{games.length}</strong><span>Games</span></div><div><strong>{globalActionCount}</strong><span>Actions</span></div><div><strong>{readySourceCatalogCount}/{sourceCatalogCount}</strong><span>Sources</span></div><div><strong>{readyExtensionSettingCount()}/{extensionSettings.length}</strong><span>Ext Settings</span></div></div>
          <button type="button" class="phone-row" on:click={() => openSettings("sources")}><span><strong>Sources</strong><small>{readySourceCatalogCount} ready out of {sourceCatalogCount}; configure provider keys here.</small></span><b>›</b></button>
          <button type="button" class="phone-row" on:click={() => openSettings("install")}><span><strong>Install Behavior</strong><small>Control automatic installs, profile enabling, and installer choice prompts.</small></span><b>›</b></button>
          <button type="button" class="phone-row" on:click={() => openSettings("extensions")}><span><strong>Extension Settings</strong><small>{extensionSettings.length} global extension setting{extensionSettings.length === 1 ? "" : "s"} available.</small></span><b>›</b></button>
        {:else if activeSettingsPage === "jobs"}
          {#if visibleJobs.length === 0}<article class="phone-static-row"><strong>No jobs</strong><small>Downloads, installs, deployment, and provider work will appear here.</small></article>{/if}
          {#each visibleJobs as job}<button type="button" class="phone-row" on:click={() => openActionItem(job)}><span><strong>{job.title}</strong><small>{job.status} · {job.message}</small><span class="pills"><em>{job.type}</em><em>{jobSourceLabel(job)}</em></span></span><b>›</b></button>{/each}
        {:else if activeSettingsPage === "install"}
          <label class="setting-row"><span><strong>Auto-install captured downloads</strong><small>Download links are cached immediately.</small></span><input type="checkbox" checked={status?.install.auto_install_captured_downloads} on:change={(event) => updateAutoInstall(event.currentTarget.checked)} /></label>
          <label class="setting-row"><span><strong>Auto-enable installed mods</strong><small>Leave off unless you trust the current profile.</small></span><input type="checkbox" checked={status?.install.auto_enable_installed_mods} on:change={(event) => updateAutoEnable(event.currentTarget.checked)} /></label>
          <label class="setting-row"><span><strong>Auto-display installer choices</strong><small>Show FOMOD and multi-choice installers as soon as they are ready.</small></span><input type="checkbox" checked={status?.install.auto_show_fomod_installers ?? true} on:change={(event) => updateAutoShowFOMOD(event.currentTarget.checked)} /></label>
          <label class="setting-row"><span><strong>Global download slots</strong><small>Limit simultaneous captured downloads across all games.</small></span><select value={status?.download?.max_concurrent_captured_downloads ?? 1} on:change={(event) => updateDownloadConcurrency(Number(event.currentTarget.value))}>{#each [1, 2, 3, 4] as count}<option value={count}>{count}</option>{/each}</select></label>
          <label class="setting-row"><span><strong>Per-game download slots</strong><small>Keep one large game from starving the rest of the queue.</small></span><select value={status?.download?.max_concurrent_captured_downloads_per_game ?? 1} on:change={(event) => updatePerGameDownloadConcurrency(Number(event.currentTarget.value))}>{#each [1, 2, 3, 4] as count}<option value={count}>{count}</option>{/each}</select></label>
        {:else if activeSettingsPage === "sources"}
          {#if catalogSettingsMessage}<p class="phone-hint success">{catalogSettingsMessage}</p>{/if}
          {#each catalogs as catalog}
            <article class="phone-static-row"><strong>{catalog.name}</strong><small>{catalog.configured ? "Configured" : catalog.credentials_required ? "Needs credentials" : catalog.status} · {catalogCapabilities(catalog).join(", ") || "No active capabilities"}</small><span class="pills"><em class:ok={catalog.status === "ready"}>{catalog.status}</em>{#if catalog.source_tag}<em class="source">{sourceLabel(catalog.source_tag)}</em>{/if}</span></article>
          {/each}
          <form class="inline-form-clean" on:submit|preventDefault={() => updateCatalogCredential("modio", modIOAPIKey)}><input bind:value={modIOAPIKey} placeholder="mod.io API key" /><button type="submit" disabled={catalogSettingsBusy === "modio" || !modIOAPIKey.trim()}>Save mod.io</button></form>
          <form class="inline-form-clean" on:submit|preventDefault={() => updateCatalogCredential("curseforge", curseForgeAPIKey)}><input bind:value={curseForgeAPIKey} placeholder="CurseForge API key" /><button type="submit" disabled={catalogSettingsBusy === "curseforge" || !curseForgeAPIKey.trim()}>Save CurseForge</button></form>
        {:else if activeSettingsPage === "nexus"}
          {#if nexusSettingsMessage}<p class="phone-hint success">{nexusSettingsMessage}</p>{/if}
          <article class="phone-static-row"><strong>Nexus Mods API</strong><small>{status?.nexus.api_key_configured ? "Configured. Paste a new key to replace it." : "Missing. Nexus browsing and API lookup need a key."}</small><span class="pills"><em class:ok={status?.nexus.api_key_configured}>{status?.nexus.api_key_configured ? "Ready" : "Required"}</em></span></article>
          <form class="inline-form-clean" on:submit|preventDefault={updateNexusAPIKey}><input bind:value={nexusAPIKey} placeholder="Paste Nexus API key" /><button type="submit" disabled={nexusSettingsBusy || !nexusAPIKey.trim()}>{status?.nexus.api_key_configured ? "Replace Key" : "Save Key"}</button></form>
        {:else if activeSettingsPage === "extensions"}
          {#if extensionSettingsMessage}<p class="phone-hint success">{extensionSettingsMessage}</p>{/if}
          {#if extensionSettings.length === 0}<article class="phone-static-row"><strong>No extension settings</strong><small>Installed game extensions do not expose global settings yet.</small></article>{/if}
          {#each extensionSettingGroups() as group}
            <article class="phone-static-row extension-group"><strong>{group.extensionID}</strong><small>{group.settings.length} setting{group.settings.length === 1 ? "" : "s"}</small>
              {#each group.settings as setting}
                <div class="setting-editor">
                  <label><span><strong>{setting.name}</strong><small>{setting.message || setting.scope || setting.setting_id}</small></span>
                    {#if extensionSettingValueType(setting) === "bool"}
                      <input type="checkbox" checked={extensionSettingDraft(setting) === "true"} disabled={!extensionSettingReady(setting) || extensionSettingBusy === extensionSettingKey(setting)} on:change={(event) => updateExtensionSettingDraft(setting, event.currentTarget.checked ? "true" : "false")} />
                    {:else if setting.options?.length}
                      <select value={extensionSettingDraft(setting)} disabled={!extensionSettingReady(setting) || extensionSettingBusy === extensionSettingKey(setting)} on:change={(event) => updateExtensionSettingDraft(setting, event.currentTarget.value)}>{#each setting.options as option}<option value={option.id} disabled={option.disabled}>{option.label}</option>{/each}</select>
                    {:else}
                      <input value={extensionSettingDraft(setting)} disabled={!extensionSettingReady(setting) || extensionSettingBusy === extensionSettingKey(setting)} placeholder={setting.placeholder || setting.setting_id} on:input={(event) => updateExtensionSettingDraft(setting, event.currentTarget.value)} />
                    {/if}
                  </label>
                  <button type="button" disabled={!extensionSettingReady(setting) || extensionSettingBusy === extensionSettingKey(setting)} on:click={() => saveExtensionSetting(setting)}>Save</button>
                </div>
              {/each}
            </article>
          {/each}
        {:else if activeSettingsPage === "game-stores"}
          <div class="metric-grid"><div><strong>{games.filter((game) => game.store === "steam").length}</strong><span>Steam</span></div><div><strong>{games.filter((game) => game.store && game.store !== "steam").length}</strong><span>Other</span></div><div><strong>{games.filter((game) => game.state === "managed").length}</strong><span>Managed</span></div><div><strong>{games.filter((game) => game.downloaded).length}</strong><span>Installed</span></div></div>
          <label class="inline-select"><span>Game list</span><select bind:value={gameVisibility}><option value="manageable">Manage Ready</option><option value="extensions">DMM Extensions</option><option value="all">All Installed</option></select></label>
        {:else}
          <article class="phone-static-row"><strong>Settings page unavailable</strong><small>This section is not registered in the phone shell.</small></article>
        {/if}
      </section>
    </section>
  {:else if selectedGame}
    <section class="phone-content">
      <section class="phone-card game-hero">
        <div><img src={gameImage(selectedGame.app_id)} alt="" /><span><strong>{selectedGame.name}</strong><small>{selectedProfile?.name ?? "Default Profile"} · {installedMods.filter((mod) => mod.enabled).length} enabled / {installedMods.length} installed</small></span></div>
        <div class="action-grid"><button type="button" on:click={applyLaunchSetup}>Launch Game</button><button type="button" on:click={() => openGameModule("profiles")}>Change Profile</button><button type="button" on:click={toggleDeckArchiveBrowser}>Import Archive</button><button type="button" on:click={() => void searchNexusMods()}>Explore Mods</button></div>
      </section>
      {#if deckArchiveBrowserOpen}
        <section class="phone-card"><header><h2>Import Archive</h2><span>{deckArchiveBrowserEntries.length} entries</span></header>
          {#if deckLocalArchiveMessage}<p class="phone-hint">{deckLocalArchiveMessage}</p>{/if}
          <div class="inline-form-clean"><input bind:value={deckArchivePathInput} placeholder="Deck path" /><button type="button" disabled={deckLocalArchiveBusy} on:click={() => browseDeckArchiveFolder(deckArchivePathInput)}>Open</button></div>
          {#if deckArchiveBrowserParentPath}<button type="button" class="phone-row" on:click={() => browseDeckArchiveFolder(deckArchiveBrowserParentPath)}><span><strong>Up one folder</strong><small>{deckArchiveBrowserParentPath}</small></span><b>↥</b></button>{/if}
          {#each deckArchiveBrowserEntries as entry}
            {#if entry.kind === "directory"}
              <button type="button" class="phone-row" on:click={() => browseDeckArchiveFolder(entry.path)}><span><strong>{entry.name}</strong><small>{entry.path}</small></span><b>›</b></button>
            {:else}
              <button type="button" class="phone-row" disabled={busyDeckLocalArchivePath === entry.path} on:click={() => importDeckLocalArchive(entry)}><span><strong>{entry.name}</strong><small>{formatBytes(entry.bytes ?? 0)} · {entry.path}</small></span><b>{busyDeckLocalArchivePath === entry.path ? "…" : "+"}</b></button>
            {/if}
          {/each}
        </section>
      {/if}
      {#if activeGameModule === "profiles"}
        <section class="phone-card"><header><h2>Profiles</h2><span>{profiles.length}</span></header>{#each profiles as profile}<button type="button" class="phone-row" class:selected={profile.is_default} on:click={() => setDefaultProfile(profile)}><span><strong>{profile.name}</strong><small>{profile.enabled_mod_count} enabled / {profile.mod_count} total</small><span class="pills">{#if profile.is_default}<em class="ok">Active</em>{/if}</span></span><b>›</b></button>{/each}</section>
        <section class="phone-card"><header><h2>Profile Tools</h2></header>
          <form class="inline-form-clean" on:submit|preventDefault={createProfile}><input bind:value={profileName} placeholder="New profile name" /><button type="submit" disabled={!profileName.trim()}>Create</button></form>
          <label class="setting-row"><span><strong>Clone active profile</strong><small>New profiles can start with the active profile's enabled mods.</small></span><input type="checkbox" bind:checked={copyProfileFromActive} /></label>
          {#if selectedProfile && profiles.length > 1 && installedMods.length > 0}
            {#each installedMods as mod}
              <article class="phone-static-row"><strong>{mod.name}</strong><small>{mod.enabled ? "Enabled" : "Disabled"} in {selectedProfile.name}</small>
                <div class="inline-form-clean">
                  <select value={String(transferTargetProfileID(mod))} on:change={(event) => (profileTransferTargets[mod.id] = event.currentTarget.value)}>
                    {#each profiles.filter((profile) => profile.id !== selectedProfile.id) as targetProfile}<option value={targetProfile.id}>{targetProfile.name}</option>{/each}
                  </select>
                  <button type="button" disabled={Boolean(busyMods[mod.id])} on:click={() => transferInstalledMod(mod, false)}>Copy</button>
                  <button type="button" disabled={Boolean(busyMods[mod.id])} on:click={() => transferInstalledMod(mod, true)}>Move</button>
                </div>
              </article>
            {/each}
          {:else}
            <article class="phone-static-row"><strong>Move or copy mods</strong><small>Create another profile and install mods before transferring profile membership.</small></article>
          {/if}
        </section>
      {:else if activeGameModule === "advanced"}
        <section class="phone-card"><header><h2>Rollback</h2><span>{deploymentHistory.length} kept</span></header>{#each deploymentHistory as deployment}<button type="button" class="phone-row" on:click={() => previewRestorePoint(deployment)}><span><strong>{deployment.reason || "Restore point"}</strong><small>{new Date(deployment.created_at).toLocaleString()} · {deployment.actions} changes</small></span><b>›</b></button>{/each}</section>
        {#if restorePointPreview}<section class="phone-card"><header><h2>Inspect Delta</h2><span>{restorePointPreview.actions?.length ?? 0} changes</span></header>{#each restorePointPreview.actions ?? [] as action}<article class="phone-static-row"><strong>{deployOperationLabel(action.operation)}</strong><small>{action.target_path}</small></article>{/each}<div class="action-grid"><button type="button" on:click={() => (restorePointPreview = null)}>Back</button><button type="button" on:click={() => restoreDeployment(deploymentHistory.find((item) => item.id === restorePointPreview?.deployment_id)!)}>Restore</button></div></section>{/if}
      {:else if activeGameModule === "actions"}
        <section class="phone-card"><header><h2>Game Actions</h2><span>{selectedGameActionItems.length + installCandidates.length}</span></header>{#each selectedGameActionItems as action}<button type="button" class="phone-row" on:click={() => openActionItem(action)}><span><strong>{action.title}</strong><small>{action.message || actionNextStep(action)}</small></span><b>›</b></button>{/each}{#each installCandidates as candidate}<button type="button" class="phone-row" on:click={() => openInstallCandidate(candidate)}><span><strong>{candidate.name}</strong><small>{candidate.reason}</small></span><b>›</b></button>{/each}</section>
      {:else}
        <div class="phone-segments"><button type="button" class:active={modSourceFilter === "all"} on:click={() => (modSourceFilter = "all")}>All</button><button type="button" on:click={() => (modSourceFilter = "enabled")}>Enabled</button><button type="button" on:click={() => (modSourceFilter = "disabled")}>Disabled</button><button type="button" on:click={() => openGameModule("advanced")}>Rollback</button></div>
        <section class="phone-card"><header><h2>Profile Mods</h2><span>{installedMods.length} total</span></header>{#each installedMods as mod}<article class="phone-mod-row"><span><strong>{mod.name}</strong><small>{sourceLabel(mod.source_tag || mod.catalog)} · {mod.mod_type ?? mod.status}</small><span class="pills"><em class:ok={mod.enabled}>{mod.enabled ? "Enabled" : "Disabled"}</em>{#if hasSourceTag(mod.source_tag || mod.catalog)}<em class="source">{sourceLabel(mod.source_tag || mod.catalog)}</em>{/if}</span></span><label><input type="checkbox" checked={mod.enabled} disabled={Boolean(busyMods[mod.id])} on:change={(event) => setModEnabled(mod, event.currentTarget.checked)} /></label><div class="mod-row-actions"><button type="button" disabled={Boolean(busyMods[mod.id])} on:click={() => reinstallInstalledMod(mod, true)}>Reinstall</button><button type="button" disabled={Boolean(busyMods[mod.id])} on:click={() => updateInstalledMod(mod)}>Update</button><button type="button" disabled={Boolean(busyMods[mod.id])} on:click={() => askRemoveInstalledMod(mod)}>Delete</button></div></article>{/each}</section>
      {/if}
    </section>
  {:else}
    <section class="phone-empty"><h2>Select a Game</h2><p>Open the menu to choose a game.</p><button type="button" on:click={() => (drawer = "games")}>Open Games</button></section>
  {/if}
</main>

{#if confirmation}
  <section class="confirm-layer" aria-label="Confirm action">
    <button type="button" class="confirm-scrim" aria-label="Cancel confirmation" on:click={() => (confirmation = null)}></button>
    <article class:danger-confirm={confirmation.danger} class="confirm-dialog">
      <div><p class="eyebrow">Confirm</p><h2>{confirmation.title}</h2></div>
      <p>{confirmation.message}</p>
      {#if confirmation.detail}<p class="confirm-detail">{confirmation.detail}</p>{/if}
      <div class="confirm-actions"><button type="button" class="secondary-action" on:click={() => (confirmation = null)}>Cancel</button><button type="button" class:danger-action={confirmation.danger} on:click={confirmCurrentAction}>{confirmation.confirmLabel}</button></div>
    </article>
  </section>
{/if}
