import {
  ButtonItem,
  ConfirmModal,
  Focusable,
  GamepadButton,
  ModalRoot,
  Navigation,
  NavEntryPositionPreferences,
  PanelSection,
  PanelSectionRow,
  Router,
  SteamSpinner,
  Tabs,
  TextField,
  ToggleField,
  findModuleChild,
  gamepadTabbedPageClasses,
  mainBrowserClasses,
  showModal,
  sleep,
  staticClasses,
  steamSpinnerClasses,
  type Tab as DeckyUITab,
  type GamepadEvent
} from "@decky/ui";
import { call, definePlugin, routerHook, toaster } from "@decky/api";
import { FaPowerOff } from "react-icons/fa";
import qrcode from "qrcode-generator";
import { ComponentType, CSSProperties, ReactNode, useEffect, useMemo, useRef, useState } from "react";

declare const SteamClient:
  | {
      Apps?: {
        RunGame?: (appId: string, launchOptions: string, param2: number, launchSource: number) => void;
        SetAppLaunchOptions?: (appid: number, launchOptions: string) => void;
        GetSubscribedWorkshopItems?: (appid: number) => Promise<SteamWorkshopClientItem[]>;
        GetSubscribedWorkshopItemDetails?: (appid: number, itemIds: string[]) => Promise<SteamWorkshopClientItemDetails[]>;
        GetDownloadedWorkshopItems?: (appid: number) => Promise<SteamWorkshopClientItem[]>;
        SetWorkshopItemsDisabledLocally?: (appid: number, itemIds: string[], disabled: boolean) => void | Promise<unknown>;
        SetWorkshopItemsLoadOrder?: (appid: number, itemIds: string[]) => void | Promise<unknown>;
        SubscribeWorkshopItem?: (appid: number, itemId: string, subscribed: boolean) => void | Promise<unknown>;
        DownloadWorkshopItem?: (appid: number, itemId: string, highPriority: boolean) => void | Promise<unknown>;
      };
      GameSessions?: {
        RegisterForAppLifetimeNotifications?: (callback: (notification: { unAppID: number; bRunning: boolean }) => void) => { unregister?: () => void; Unregister?: () => void } | (() => void);
      };
      Overlay?: {
        GetOverlayBrowserInfo?: () => Promise<unknown[]>;
        HandleProtocolForOverlayBrowser?: (appId: number, protocol: string) => void;
        RegisterForOverlayBrowserProtocols?: (callback: (payload: { unAppID?: number; strScheme?: string; bAdded?: boolean; [key: string]: unknown }) => void) => { unregister?: () => void; Unregister?: () => void } | (() => void);
      };
      Browser?: {
        StartDownload?: (url: string) => void;
      };
      BrowserView?: {
        Destroy?: (browserView: SteamBrowserViewPopup) => void;
      };
      System?: {
        OpenInSystemBrowser?: (url: string) => void;
      };
      URL?: {
        ExecuteSteamURL?: (url: string) => void;
        RegisterForRunSteamURL?: (section: string, callback: (param0: number, url: string) => void) => { unregister?: () => void; Unregister?: () => void } | (() => void);
      };
      WebChat?: {
        OpenURLInClient?: (url: string, pid: number, forceExternal: boolean) => void;
      };
    }
  | undefined;

type SteamBrowserViewPopup = {
  GoBack?: () => void;
  GoForward?: () => void;
  LoadURL?: (url: string) => void;
  ReplaceURL?: (url: string) => void;
  SetBlockedProtocols?: (protocols: string) => void;
  SetBounds?: (x: number, y: number, width: number, height: number) => void;
  SetName?: (browserName: string) => void;
  SetFocus?: (value: boolean) => void;
  SetVisible?: (value: boolean) => void;
  SetWindowStackingOrder?: (value: number) => void;
  SetSteamURLCallback?: (callback: (url: string) => void) => void;
  on?: (event: string, callback: (...args: unknown[]) => void) => void;
  off?: (event: string, callback: (...args: unknown[]) => void) => void;
};

type DMMNativeBrowser = {
  m_browserView: SteamBrowserViewPopup;
  m_URLRequested?: string;
  LoadURL?: (url: string) => void;
  Destroy?: () => void;
  name?: string;
};

type DMMWindowRouter = {
  CreateBrowserView?: (name: string) => DMMNativeBrowser;
  HeaderStore?: {
    GetCurrentBrowserAndBackstack?: () => { browser?: { name?: string } | null } | null;
    SetCurrentBrowserAndBackstack?: (browser: DMMNativeBrowser | null, includeBackstack: boolean) => void;
  };
};

type DMMBrowserRequest = {
  appID: string;
  initialURL: string;
  profileID: number;
  source: string;
  title: string;
};

type DMMDeckyReturnContext = {
  appID: string;
  tab: FreshDeckyTab;
};

type DMMBrowserContainerProps = {
  browser: DMMNativeBrowser;
  className?: string;
  visible?: boolean;
  hideForModals?: boolean;
  external?: boolean;
  displayURLBar?: boolean;
  autoFocus?: boolean;
};

type DMMFocusedNodeEvent = CustomEvent<{
  focusedNode?: {
    BChildTakeFocus?: () => void;
  };
}>;

type DeckyExploreSource = {
  id: string;
  catalog: string;
  title: string;
  detail: string;
  action: string;
  enabled: boolean;
  behavior: "browse" | "paste" | "info";
  gameDomain?: string;
  informational?: boolean;
};

const DMMBrowserContainer = findModuleChild((mod: unknown) => {
  if (!mod || typeof mod !== "object") return undefined;
  for (const value of Object.values(mod as Record<string, unknown>)) {
    if (typeof value !== "function") continue;
    const source = value.toString();
    if (source.includes("displayURLBar") && source.includes("BExternalTriggeredLoad()")) {
      return value;
    }
  }
  return undefined;
}) as ComponentType<DMMBrowserContainerProps> | undefined;

type SteamWorkshopClientItem = {
  unAppID?: number;
  appid?: number;
  ulPublishedFileID?: string | number;
  publishedfileid?: string | number;
  published_file_id?: string | number;
  title?: string;
  strTitle?: string;
  details?: SteamWorkshopClientItemDetails;
  bDisabledLocally?: boolean | number;
  bDisabled?: boolean | number;
  disabled?: boolean | number;
  disabled_locally?: boolean | number;
  load_order?: number;
  time_subscribed?: number;
  time_updated?: number;
  file_size?: string | number;
};

type SteamWorkshopClientItemDetails = {
  publishedfileid?: string | number;
  published_file_id?: string | number;
  title?: string;
  strTitle?: string;
  file_size?: string | number;
  preview_url?: string;
  short_description?: string;
  consumer_appid?: number;
  creator?: unknown;
  children?: Array<string | number>;
  tags?: string[];
};

type BackendStatus = {
  running: boolean;
  ip?: string;
  port: number;
  url?: string;
  plain_url?: string;
  pid?: number;
  build?: BuildInfo;
  auth?: {
    enabled?: boolean;
    token?: string;
    token_file?: string;
  };
  backend?: {
    lan_only: boolean;
    game_count: number;
    nexus: { api_key_configured: boolean };
    download?: {
      max_concurrent_captured_downloads: number;
      max_concurrent_captured_downloads_per_game: number;
      active_captured_downloads: number;
      active_captured_downloads_by_game?: Record<string, number>;
    };
    install: {
      auto_install_captured_downloads: boolean;
      auto_enable_installed_mods: boolean;
      auto_show_fomod_installers: boolean;
    };
    ui?: UISettings;
  } | null;
  logs?: {
    plugin: string;
    backend: string;
  };
  error?: string;
};

type CatalogStatus = {
  id: string;
  name: string;
  kind: string;
  status: string;
  configured?: boolean;
  credentials_required?: boolean;
  capabilities?: string[];
  url_import?: boolean;
  search?: boolean;
  browse?: boolean;
  download?: boolean;
  archive_upload?: boolean;
  installed_management?: boolean;
  source_tag?: string;
  notes?: string[];
};

type BuildInfo = {
  path?: string;
  commit?: string;
  short_commit?: string;
  built_at?: string;
  channel?: string;
  version?: string;
  error?: string;
};

type UISettings = {
  favorite_game_ids?: string[];
  recent_games?: Record<string, number>;
  game_sort?: GameSort;
};

type Dependency = {
  name: string;
  command: string;
  installed: boolean;
  path?: string;
};

type NXMStatus = {
  desktop_path: string;
  desktop_exists: boolean;
  current_handler: string;
  protocol_handler?: string;
  xdg_handler?: string;
  registered: boolean;
};

type Diagnostics = {
  logs: Record<string, { path: string; tail: string }>;
};

type UpdateResult = {
  ok: boolean;
  error?: string;
  message?: string;
  package?: string;
  url?: string;
  bytes?: number;
  installer_pid?: number;
  log?: string;
};

type Job = {
  id: string;
  type: string;
  title: string;
  status: string;
  message?: string;
  payload?: Record<string, string>;
  app_id?: string;
  catalog?: string;
  source_tag?: string;
  updated_at?: string;
};

type DomainEvent = {
  id: number;
  type: string;
  app_id?: string;
  job_id?: string;
  payload?: unknown;
  created_at?: string;
};

type DeckyBrowserOpenPayload = {
  url: string;
  profile_id?: number;
  source?: string;
  title?: string;
  expires_at?: string;
};

type ManagedGame = {
  app_id: string;
  name: string;
  state: string;
  nexus_domains?: string[];
  extension?: GameExtensionInfo;
  steam_workshop?: {
    detected?: boolean;
    item_count?: number;
    management_supported?: boolean;
  };
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
  sources?: Array<{ name?: string; url?: string }>;
};

type RunningGame = {
  app_id: string;
  name: string;
};

type Profile = {
  id: number;
  name: string;
  is_default: boolean;
  mod_count: number;
  enabled_mod_count: number;
};

type ManagedMod = {
  id: number;
  name: string;
  enabled: boolean;
  priority: number;
  status: string;
  catalog: string;
  source_tag?: string;
  source_url?: string;
  source_game_domain: string;
  source_mod_id: string;
  source_file_id: string;
  mod_type?: string;
  planner_id?: string;
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

type RuntimeRequirement = {
  id: string;
  name: string;
  kind: string;
  required: boolean;
  status: string;
  message: string;
  details?: string[];
  help_url?: string;
  install_hint?: string;
};

type GameDiagnostics = {
  runtime_requirements?: RuntimeRequirement[];
  validation_warnings?: string[];
};

type NexusSearchSort = "downloads" | "unique_downloads" | "popular" | "updated" | "name" | "relevance";
type NexusTimeWindow = "all" | "one_week" | "three_weeks" | "one_month" | "three_months" | "one_year";
type GameVisibility = "manageable" | "extensions" | "all";

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

type ProfileApplyResult = {
  status: string;
  message?: string;
  job?: Job;
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

type DeployAction = {
  target_path?: string;
  target_relative?: string;
  operation?: string;
  strategy?: string;
  installed_mod_id?: number;
  mod_id?: string;
  priority?: number;
  winner_installed_mod_id?: number;
  winner_mod_id?: string;
  winner_priority?: number;
  conflict?: boolean;
  conflict_reason?: string;
};

type DeployPlan = {
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

type InstallCandidate = {
  id: number;
  steam_app_id: string;
  name: string;
  catalog: string;
  source_tag?: string;
  status: string;
  reason: string;
  installer_json?: string;
  choices_json?: string;
  source_game_domain: string;
  source_mod_id: string;
  source_file_id: string;
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

type LocalArchiveBrowseState = {
  roots: string[];
  entries: LocalArchiveBrowseEntry[];
  current_path: string;
  parent_path?: string;
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

type LaunchAction = {
  type: string;
  app_id: string;
  tool_id: string;
  desired_options: string;
  current_options?: string;
  reason: string;
  source_extension: string;
  risk: string;
};

type LaunchStatus = {
  app_id: string;
  required: boolean;
  configured: boolean;
  can_configure: boolean;
  desired_options?: string;
  current_options?: string;
  missing_files?: string[];
  tool?: {
    id: string;
    name: string;
    executable_relative: string;
    executable_path: string;
    arguments?: string[];
    required_files?: string[];
    shell?: boolean;
    detach?: boolean;
    exclusive?: boolean;
    source_extension: string;
  };
  action?: LaunchAction;
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
  raw_json?: string;
};

type WorkshopState = {
  supported: boolean;
  items: WorkshopItem[];
};

type PluginLoadOrder = {
  app_id?: string;
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

type WorkshopActionJob = Job & {
  payload?: {
    app_id?: string;
    item_id?: string;
    kind?: "subscribe" | "unsubscribe" | "enable" | "disable" | string;
    action_name?: string;
  };
};

type GameSort = "recent" | "az" | "za";
type DeckyModSort = "profile" | "source" | "az" | "za" | "enabled";

const DMM_DECKY_ROUTE = "/decky-mod-manager";
const DMM_BROWSER_ROUTE = "/decky-mod-manager-browser";
const DMM_NATIVE_BROWSER_NAME = "DeckyModManagerBrowser";
const DMM_NATIVE_BROWSER_TAB_ID = "dmm-native-browser-tab";
const EXTENSION_NOTICE_ACTION_RUN_LAUNCH_TOOL = "run-launch-tool";
const STEAM_LAUNCH_SOURCE_DASH_APP_LAUNCH_CMD_LINE = 300;

const deckyQuickAccessFrameStyle: CSSProperties = {
  alignSelf: "stretch",
  boxSizing: "border-box",
  display: "grid",
  flexDirection: "column",
  gap: "8px",
  maxWidth: "100%",
  minWidth: 0,
  overflowX: "hidden",
  overflowY: "visible",
  width: "100%"
};

const deckyRuntimeStyles = `
.dmm-sidebar-surface,
.dmm-sidebar-surface * {
  box-sizing: border-box;
  max-width: 100%;
  min-width: 0;
}
.dmm-sidebar-row {
  box-sizing: border-box;
  max-width: 100%;
  min-width: 0;
  overflow-x: hidden;
  width: 100%;
}
.dmm-sidebar-row-focused,
.dmm-sidebar-row:focus,
.dmm-sidebar-row:focus-visible,
.dmm-sidebar-row:focus-within {
  background: rgba(39, 54, 74, 0.86) !important;
  border-color: #7dd3fc !important;
  box-shadow: inset 0 0 0 1px rgba(125, 211, 252, 0.42) !important;
}
.dmm-focus-card {
  outline: none;
  max-width: 100%;
  min-width: 0;
  transition: background 120ms ease, border-color 120ms ease, box-shadow 120ms ease, opacity 120ms ease;
  width: 100%;
}
.dmm-focus-card > * {
  min-width: 0;
}
.dmm-action-grid,
.dmm-action-grid > *,
.dmm-action-grid .dmm-focus-card {
  max-width: 100%;
  min-width: 0;
  width: 100%;
}
.dmm-action-grid {
  overflow-x: hidden;
}
.dmm-action-grid .dmm-focus-card {
  inline-size: 100%;
}
.dmm-sidebar-row img {
  flex-shrink: 0;
}
.dmm-sidebar-row span,
.dmm-sidebar-row div {
  min-width: 0;
}
  .dmm-focus-card-focused,
  .dmm-decky-tab-focused {
    background: #27364a !important;
    border-color: #7dd3fc !important;
    box-shadow: inset 0 0 0 1px rgba(125, 211, 252, 0.45) !important;
  }
.dmm-focus-card:focus,
.dmm-focus-card:focus-visible,
.dmm-focus-card:focus-within {
  background: #27364a !important;
  border-color: #7dd3fc !important;
  box-shadow: inset 0 0 0 1px rgba(125, 211, 252, 0.45) !important;
}
.dmm-installer-choice-scroll {
  border-right: 1px solid rgba(125, 211, 252, 0.32);
  scrollbar-color: #7dd3fc rgba(15, 23, 42, 0.92);
  scrollbar-gutter: stable;
  scrollbar-width: auto;
}
.dmm-installer-choice-scroll::-webkit-scrollbar {
  width: 12px;
}
.dmm-installer-choice-scroll::-webkit-scrollbar-track {
  background: rgba(15, 23, 42, 0.92);
  border: 1px solid rgba(125, 211, 252, 0.18);
  border-radius: 999px;
}
.dmm-installer-choice-scroll::-webkit-scrollbar-thumb {
  background: #7dd3fc;
  border: 2px solid rgba(15, 23, 42, 0.92);
  border-radius: 999px;
}
.dmm-installer-choice-scroll::-webkit-scrollbar-thumb:hover {
  background: #99f6e4;
}
`;

const deckyFocusableCardBase: CSSProperties = {
  borderRadius: "6px",
  boxSizing: "border-box",
  color: "#f8fafc",
  cursor: "pointer",
  maxWidth: "100%",
  minWidth: 0,
  overflow: "hidden",
  width: "100%"
};

function deckyFocusableCardStyle(focused: boolean, active = false): CSSProperties {
  const highlighted = focused || active;
  return {
    ...deckyFocusableCardBase,
    background: active ? "#0f766e" : focused ? "#27364a" : "#1f2937",
    border: `1px solid ${highlighted ? "#7dd3fc" : "#374151"}`,
    boxShadow: focused ? "inset 0 0 0 1px rgba(125, 211, 252, 0.45)" : "none"
  };
}

const deckyTwoLineTextStyle: CSSProperties = {
  display: "-webkit-box",
  lineHeight: 1.18,
  maxHeight: "2.36em",
  minWidth: 0,
  overflow: "hidden",
  overflowWrap: "anywhere",
  WebkitBoxOrient: "vertical",
  WebkitLineClamp: 2,
  wordBreak: "break-word"
};

const deckySidebarListStyle: CSSProperties = {
  boxSizing: "border-box",
  display: "grid",
  gap: "8px",
  maxWidth: "100%",
  minWidth: 0,
  overflowX: "hidden",
  overflowY: "visible",
  width: "100%"
};

const deckySidebarSurfaceStyle: CSSProperties = {
  boxSizing: "border-box",
  display: "grid",
  gap: "10px",
  maxWidth: "100%",
  minWidth: 0,
  overflowX: "hidden",
  overflowY: "visible",
  width: "100%"
};

function deckyCompositeRowStyle(focused: boolean, active = false): CSSProperties {
  return {
    alignItems: "stretch",
    background: active ? "rgba(15, 118, 110, 0.28)" : focused ? "rgba(39, 54, 74, 0.8)" : "rgba(17, 24, 39, 0.55)",
    border: `1px solid ${focused ? "#7dd3fc" : active ? "#0f766e" : "#303741"}`,
    borderRadius: "6px",
    boxShadow: focused ? "inset 0 0 0 1px rgba(125, 211, 252, 0.35)" : "none",
    boxSizing: "border-box",
    display: "grid",
    gap: "6px",
    gridTemplateColumns: "minmax(0, 1fr)",
    maxWidth: "100%",
    minWidth: 0,
    overflowX: "hidden",
    overflowY: "visible",
    outline: focused ? "2px solid rgba(125, 211, 252, 0.34)" : "none",
    outlineOffset: "-2px",
    padding: "6px",
    scrollMarginBlock: "10px",
    transition: "background 120ms ease, border-color 120ms ease, box-shadow 120ms ease, outline-color 120ms ease",
    width: "100%"
  };
}

function sourceLabel(catalog?: string) {
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

function sourceForManagedMod(mod: ManagedMod) {
  return mod.source_tag || mod.catalog;
}

function sourceForInstallCandidate(candidate: InstallCandidate) {
  return candidate.source_tag || candidate.catalog;
}

function sourceForJob(job: Job) {
  return job.source_tag || job.catalog || job.payload?.source_tag || job.payload?.catalog || (job.type === "steam-workshop-action" ? "steam_workshop" : "extension");
}

function catalogStatusLabel(status?: string) {
  const normalized = (status ?? "").trim().toLowerCase();
  if (normalized === "ready") return "Ready";
  if (normalized === "needs_credentials") return "Needs Key";
  if (normalized === "deferred") return "Deferred";
  if (normalized === "planned") return "Planned";
  return normalized ? normalized.replace(/[_-]+/g, " ") : "Unknown";
}

function deckySourcePillStyle(catalog?: string): CSSProperties {
  const source = (catalog ?? "").trim().toLowerCase().replace(/_/g, "-");
  const colors: Record<string, { border: string; color: string; background: string }> = {
    nexus: { border: "#7c3aed", color: "#ede9fe", background: "#2e1065" },
    "steam-workshop": { border: "#2563eb", color: "#dbeafe", background: "#172554" },
    workshop: { border: "#2563eb", color: "#dbeafe", background: "#172554" },
    thunderstore: { border: "#0891b2", color: "#cffafe", background: "#164e63" },
    modrinth: { border: "#10b981", color: "#d1fae5", background: "#064e3b" },
    gamebanana: { border: "#facc15", color: "#fef9c3", background: "#422006" },
    modio: { border: "#16a34a", color: "#dcfce7", background: "#052e16" },
    "mod.io": { border: "#16a34a", color: "#dcfce7", background: "#052e16" },
    curseforge: { border: "#f97316", color: "#ffedd5", background: "#431407" },
    moddb: { border: "#ca8a04", color: "#fef3c7", background: "#422006" },
    itchio: { border: "#ef4444", color: "#fee2e2", background: "#450a0a" },
    "itch.io": { border: "#ef4444", color: "#fee2e2", background: "#450a0a" },
    github: { border: "#52525b", color: "#f4f4f5", background: "#18181b" },
    "github-releases": { border: "#52525b", color: "#f4f4f5", background: "#18181b" },
    extension: { border: "#0f766e", color: "#ccfbf1", background: "#042f2e" },
    direct: { border: "#475569", color: "#cbd5e1", background: "#1e293b" },
    local: { border: "#475569", color: "#cbd5e1", background: "#1e293b" },
    native: { border: "#475569", color: "#cbd5e1", background: "#1e293b" }
  };
  const palette = colors[source] ?? { border: "#475569", color: "#cbd5e1", background: "#1e293b" };
  return {
    alignSelf: "flex-start",
    background: palette.background,
    border: `1px solid ${palette.border}`,
    borderRadius: "999px",
    color: palette.color,
    flex: "0 0 auto",
    fontSize: "10px",
    fontWeight: 900,
    lineHeight: 1,
    maxWidth: "100%",
    overflow: "hidden",
    padding: "4px 7px",
    textOverflow: "ellipsis",
    textTransform: "uppercase",
    whiteSpace: "nowrap"
  };
}

function deckyActionGridStyle(columns: 1 | 2 | 3): CSSProperties {
  const gridTemplateColumns = columns === 3
    ? "repeat(3, minmax(0, 1fr))"
    : columns === 2
      ? "minmax(0, 1fr) minmax(0, 1fr)"
      : "minmax(0, 1fr)";
  return {
    alignItems: "stretch",
    boxSizing: "border-box",
    display: "grid",
    gap: "6px",
    gridTemplateColumns,
    maxWidth: "100%",
    minWidth: 0,
    overflowX: "hidden",
    overflowY: "visible",
    width: "100%"
  };
}

function deckyCompactActionStyle(kind: "neutral" | "danger" = "neutral", focused = false): CSSProperties {
  const danger = kind === "danger";
  return {
    ...deckyFocusableCardBase,
    alignItems: "center",
    background: danger ? "#3f1d1d" : focused ? "#27364a" : "#1f2937",
    border: `1px solid ${danger ? "#7f1d1d" : focused ? "#7dd3fc" : "#374151"}`,
    color: danger ? "#fecaca" : "#f8fafc",
    display: "flex",
    fontSize: "12px",
    fontWeight: 800,
    justifyContent: "center",
    lineHeight: 1.1,
    minHeight: "38px",
    maxWidth: "100%",
    padding: "8px 6px",
    scrollMarginBlock: "10px",
    textAlign: "center",
    textOverflow: "clip",
    overflowWrap: "anywhere",
    whiteSpace: "normal"
  };
}

const deckyCompactInputStyle: CSSProperties = {
  background: "#171b21",
  border: "1px solid #343b46",
  borderRadius: "6px",
  boxSizing: "border-box",
  color: "#f8fafc",
  fontSize: "12px",
  lineHeight: 1.2,
  minHeight: "38px",
  minWidth: 0,
  outline: "none",
  padding: "8px 10px",
  width: "100%"
};

function jobToastBody(job: Job): string {
  if (isDeckyModUpdateAction(job) && job.status === "waiting") return job.message || "Open Action Center to install this downloaded update.";
  if (isDeckyModUpdateAction(job) && (job.status === "running" || job.status === "queued")) return job.message || "DMM is downloading or installing this mod update.";
  if (job.status === "waiting") return job.message || "Open Action Center on Decky or the phone/tablet UI to continue.";
  if (job.status === "running" || job.status === "queued") return job.message || "DMM is working on this action.";
  if (job.status === "completed") return job.message || "The action completed.";
  if (job.status === "failed") return job.message || "Open DMM to review the error.";
  return job.message || job.title;
}

function jobToastTitle(job: Job): string {
  if (job.status === "failed") {
    if (job.type === "deploy") return "Applying enabled mods failed";
    if (job.type === "rollback") return "DMM rollback failed";
    if (job.type === "purge") return "DMM purge failed";
    if (job.type === "repair") return "DMM repair failed";
    return "DMM action failed";
  }
  if (job.type === "deploy") return "Applying enabled mods";
  if (job.type === "rollback") return "DMM rollback";
  if (job.type === "purge") return "DMM purge";
  if (job.type === "repair") return "DMM repair";
  if (job.type === "recover-downloads") return "DMM recovery";
  if (job.type === "extension-notice") return "DMM extension notice";
  if (isDeckyModUpdateAction(job)) return "DMM mod update";
  return "Decky Mod Manager";
}

function showJobToast(job: Job) {
  toaster.toast({
    title: jobToastTitle(job),
    body: jobToastBody(job),
    subtext: job.title,
    duration: job.status === "failed" ? 9000 : 6000,
    critical: job.status === "failed",
    playSound: true,
    showToast: true
  });
}

function showLaunchToast(title: string, body: string, failed = false) {
  toaster.toast({
    title,
    body,
    duration: failed ? 9000 : 6000,
    critical: failed,
    playSound: true,
    showToast: true
  });
}

const notifiedJobStates = new Map<string, string>();
const shownInstallerChoiceModals = new Set<string>();
const dismissedInstallerChoiceModals = new Set<string>();
const applyingInstallerChoiceCandidates = new Set<string>();
const completedLaunchActions = new Set<string>();
const launchActionAttempts = new Map<string, number>();
const completedWorkshopActions = new Set<string>();
const workshopActionAttempts = new Map<string, number>();
const workshopStateSyncLastAt = new Map<string, number>();
const handledDeckyBrowserOpenEvents = new Set<number>();
const DMM_TOAST_STORAGE_PREFIX = "decky-mod-manager:job-toast:";
const DMM_EVENT_NAME = "dmm-domain-event";
const DMM_BACKEND_WS_URL = "ws://127.0.0.1:17942/api/events/ws";
let backendAuthToken = "";
let eventMonitorSocket: WebSocket | null = null;
let eventMonitorReconnectTimer: number | null = null;
let eventMonitorReconnectDelay = 1000;
let eventMonitorLastID = 0;
let backgroundMonitorsStarted = false;
let steamBrowserNXMProbeRegistration: unknown = null;
let activeDMMNativeBrowser: DMMNativeBrowser | null = null;
let activeDMMBrowserRequest: DMMBrowserRequest | null = null;
let pendingDMMDeckyReturnContext: DMMDeckyReturnContext | null = null;

type LaunchResultSink = (message: string) => void;

function steamHeaderImage(appID: string) {
  return `https://cdn.cloudflare.steamstatic.com/steam/apps/${appID}/header.jpg`;
}

function currentRunningGame(): RunningGame | null {
  try {
    const router = Router as unknown as { MainRunningApp?: { appid?: string; display_name?: string }; RunningApps?: { appid?: string; display_name?: string }[] };
    const app = router?.MainRunningApp ?? router?.RunningApps?.[0];
    const appID = String(app?.appid ?? "").trim();
    if (!appID) return null;
    return { app_id: appID, name: String(app?.display_name ?? appID) };
  } catch (_err) {
    return null;
  }
}

function eventMatchesAppID(event: DomainEvent, appID: string) {
  return !event.app_id || event.app_id === appID;
}

function jobMatchesAppID(job: Job, appID: string) {
  const payloadAppID = String(job.payload?.app_id ?? "").trim();
  return !payloadAppID || payloadAppID === appID;
}

function isDeckyActionCenterJob(job: Job) {
  if (job.status === "completed" || job.status === "canceled") return false;
  return ["captured-install", "steam-workshop-action", "extension-notice", "deploy", "purge", "repair", "recover-downloads", "rollback"].includes(job.type);
}

function deckyJobBelongsToAppID(job: Job, appID: string) {
  const directAppID = String(job.app_id || job.payload?.app_id || "").trim();
  return directAppID !== "" && directAppID === appID;
}

function deckyActionJobsForGame(jobs: Job[], appID: string) {
	if (!appID) return [];
	return jobs.filter((job) => isDeckyActionCenterJob(job) && deckyJobBelongsToAppID(job, appID));
}

function deckyJobHasInstallCandidateReview(job: Job, candidates: InstallCandidate[]) {
  if (job.status === "queued" || job.status === "waiting" || job.status === "running") return false;
  return candidates.some((candidate) => deckyJobMatchesInstallCandidate(job, candidate));
}

function deckyJobMatchesInstallCandidate(job: Job, candidate: InstallCandidate) {
  const payload = job.payload ?? {};
  const candidateID = String(payload.candidate_id ?? "").trim();
  if (candidateID && candidateID === String(candidate.id)) return true;
  return (
    String(job.app_id || payload.app_id || "").trim() === candidate.steam_app_id &&
    String(job.catalog || payload.catalog || "").trim() === candidate.catalog &&
    String(payload.game_domain || "").trim() === candidate.source_game_domain &&
    String(payload.mod_id || "").trim() === candidate.source_mod_id &&
    String(payload.file_id || "").trim() === candidate.source_file_id
  );
}

function deckyJobStatusLabel(job: Job) {
  if (isDeckyModUpdateAction(job) && job.status === "waiting") return "Ready to update";
  if (isDeckyModUpdateAction(job) && (job.status === "queued" || job.status === "running")) return "Updating";
  const status = job.status.replace(/[_-]+/g, " ");
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function isDeckyModUpdateAction(job: Job) {
  return job.type === "captured-install" && Boolean(job.payload?.installed_mod_id && job.payload?.update_to_file_id);
}

function deckyModUpdateActionDetail(job: Job) {
  if (!isDeckyModUpdateAction(job)) return "";
  const from = job.payload?.update_from_file_id || "current";
  const to = job.payload?.update_to_file_id || "latest";
  return `Update ${from} -> ${to}`;
}

function extensionNoticeToolName(job: Job) {
  return String(job.payload?.tool_name || job.payload?.tool_id || "").trim();
}

function extensionNoticeActionKind(job: Job) {
  return String(job.payload?.action_kind || "").trim();
}

function extensionNoticeRunToolAvailable(job: Job) {
  return extensionNoticeActionKind(job) === EXTENSION_NOTICE_ACTION_RUN_LAUNCH_TOOL && String(job.payload?.tool_action_available || "").trim() === "true";
}

function extensionNoticeRunToolError(job: Job) {
  return String(job.payload?.tool_action_error || "").trim();
}

function extensionNoticeActionLabel(job: Job) {
  const label = String(job.payload?.action_label || "").trim();
  if (label) return label;
  const tool = extensionNoticeToolName(job);
  return tool ? `Open ${tool}` : "Review";
}

function extensionNoticeHelpURL(job: Job) {
  return safeHTTPURL(job.payload?.help_url);
}

function runtimeRequirementHelpURL(requirement: RuntimeRequirement) {
  return safeHTTPURL(requirement.help_url);
}

function safeHTTPURL(value: unknown) {
  const url = String(value || "").trim();
  return /^https?:\/\//i.test(url) ? url : "";
}

function deckyJobPrimaryActionLabel(job: Job) {
  if (job.type === "captured-install" && job.status === "waiting") return isDeckyModUpdateAction(job) ? "Install Update" : "Install";
  if (job.type === "captured-install" && job.status === "failed") return "Retry";
  if (job.type === "steam-workshop-action" && job.status === "failed") return "Retry";
  if (job.type === "extension-notice" && extensionNoticeRunToolAvailable(job)) return extensionNoticeActionLabel(job);
  if (job.type === "extension-notice" && extensionNoticeHelpURL(job)) return extensionNoticeActionLabel(job);
  return "";
}

function deckyJobCanCancel(job: Job) {
  if (job.status === "completed" || job.status === "canceled") return false;
  if (job.status === "failed") return job.type === "captured-install" || job.type === "steam-workshop-action";
  return true;
}

function eventShouldSyncLaunchActions(event: DomainEvent) {
  return ["profile_mods.changed", "deployment.changed", "install.changed", "launch.changed"].includes(event.type);
}

function eventShouldSyncWorkshopActions(event: DomainEvent) {
  return event.type === "job.updated" && isJob(event.payload) && event.payload.type === "steam-workshop-action";
}

function deckyModStateLabel(mod: ManagedMod) {
  if (mod.status === "installed") return mod.enabled ? "Enabled" : "Installed";
  return mod.status || (mod.enabled ? "Enabled" : "Installed");
}

function deckyModUpdateLabel(update?: ModUpdate) {
  if (!update) return "Not checked";
  if (update.status === "available") return "Update available";
  if (update.status === "current") return "Current";
  if (update.status === "error") return "Check failed";
  if (update.status === "unsupported") return "Not supported";
  return "Unknown";
}

function deckyModUpdateDetail(update?: ModUpdate) {
  if (!update) return "Use Check Updates to query supported providers.";
  if (update.message) return update.message;
  if (update.status === "available") return `Latest ${update.latest_version || update.latest_file_id || "file"}`;
  if (update.status === "current") return "Installed file is current.";
  return "Review the mod page before updating.";
}

function deckyConflictChoiceTargets(plan: DeployPlan | null, mods: ManagedMod[]): ConflictChoiceTarget[] {
  if (!plan) return [];
  const groups = new Map<string, ConflictChoiceTarget>();
  for (const action of plan.actions ?? []) {
    if (action.operation !== "skip" || !action.target_path || !action.winner_installed_mod_id) continue;
    let group = groups.get(action.target_path);
    if (!group) {
      const winner = mods.find((mod) => mod.id === action.winner_installed_mod_id);
      group = {
        target_path: action.target_path,
        target_relative: action.target_relative || action.target_path,
        current_winner_id: action.winner_installed_mod_id,
        current_winner_name: winner?.name || action.winner_mod_id || "Selected mod",
        reason: action.conflict_reason || "Resolved by profile order",
        candidates: []
      };
      groups.set(action.target_path, group);
    }
    addDeckyConflictCandidate(group, mods, action.installed_mod_id, action.priority, false);
    addDeckyConflictCandidate(group, mods, action.winner_installed_mod_id, action.winner_priority, true);
  }
  return Array.from(groups.values()).map((group) => ({
    ...group,
    candidates: [...group.candidates].sort((a, b) => Number(b.current) - Number(a.current) || (a.priority ?? 0) - (b.priority ?? 0) || a.name.localeCompare(b.name))
  }));
}

function addDeckyConflictCandidate(group: ConflictChoiceTarget, mods: ManagedMod[], installedModID?: number, priority?: number, current = false) {
  if (!installedModID) return;
  const existing = group.candidates.find((candidate) => candidate.id === installedModID);
  if (existing) {
    existing.current = existing.current || current;
    if (existing.priority === undefined) existing.priority = priority;
    return;
  }
  const mod = mods.find((item) => item.id === installedModID);
  group.candidates.push({
    id: installedModID,
    name: mod?.name || `Mod ${installedModID}`,
    catalog: mod ? sourceForManagedMod(mod) : undefined,
    priority,
    current
  });
}

function nextDeckyConflictCandidate(target: ConflictChoiceTarget) {
  return target.candidates.find((candidate) => !candidate.current) ?? null;
}

function nextGameSort(current: GameSort): GameSort {
  if (current === "recent") return "az";
  if (current === "az") return "za";
  return "recent";
}

function gameSortLabel(sort: GameSort) {
  if (sort === "az") return "A-Z";
  if (sort === "za") return "Z-A";
  return "Recent";
}

function gameVisibilityLabel(visibility: GameVisibility) {
  if (visibility === "all") return "All Installed";
  if (visibility === "extensions") return "DMM Extensions";
  return "Manage Ready";
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

function gameHasExtension(game?: ManagedGame | null) {
  return Boolean(game?.extension?.supported);
}

function gameManageReady(game?: ManagedGame | null) {
  const coverage = game?.extension?.coverage;
  return coverage === "installer" || coverage === "workshop_only";
}

function deckyGameCapabilityBadges(game: ManagedGame): Array<{ label: string; kind: string }> {
  const extension = game.extension;
  if (!extension?.supported) return [{ label: "Unsupported", kind: "unsupported" }];
  const coverage = extension.coverage ?? "metadata_only";
  const badges = [{ label: extension.coverage_label ?? "DMM", kind: `coverage-${coverage}` }];
  if (extension.nexus) badges.push({ label: "Nexus", kind: "nexus" });
  if (extension.steam_workshop) badges.push({ label: "Workshop", kind: "workshop" });
  if (coverage === "installer" && (extension.installers || extension.installer_choices)) badges.push({ label: "Install", kind: "installers" });
  if (extension.load_order || extension.plugin_activation) badges.push({ label: "Order", kind: "load-order" });
  if (extension.launch_tools) badges.push({ label: "Launch", kind: "launch" });
  return badges;
}

function firstExtensionSourceNote(game?: ManagedGame | null) {
  const source = game?.extension?.sources?.find((item) => (item.name || item.url));
  if (!source) return "";
  if (source.name && source.url) return `${source.name}: ${source.url}`;
  return source.name || source.url || "";
}

function sourceTagForExtensionURL(url?: string) {
  const normalized = (url || "").toLowerCase();
  if (normalized.includes("moddb.com")) return "moddb";
  if (normalized.includes("nexusmods.com")) return "nexus";
  if (normalized.includes("gamebanana.com")) return "gamebanana";
  if (normalized.includes("modrinth.com")) return "modrinth";
  if (normalized.includes("thunderstore.io")) return "thunderstore";
  if (normalized.includes("github.com")) return "github";
  return "direct";
}

function deckyCapabilityPillStyle(kind: string): CSSProperties {
  const colors: Record<string, { border: string; color: string; background: string }> = {
    dmm: { border: "#0f766e", color: "#ccfbf1", background: "#134e4a" },
    "coverage-installer": { border: "#0f766e", color: "#ccfbf1", background: "#134e4a" },
    "coverage-research_blocked": { border: "#d97706", color: "#ffedd5", background: "#431407" },
    "coverage-browse_only": { border: "#7c3aed", color: "#ede9fe", background: "#2e1065" },
    "coverage-workshop_only": { border: "#2563eb", color: "#dbeafe", background: "#172554" },
    "coverage-metadata_only": { border: "#64748b", color: "#e2e8f0", background: "#1e293b" },
    nexus: { border: "#7c3aed", color: "#ede9fe", background: "#2e1065" },
    workshop: { border: "#2563eb", color: "#dbeafe", background: "#172554" },
    installers: { border: "#ca8a04", color: "#fef3c7", background: "#451a03" },
    "load-order": { border: "#64748b", color: "#e2e8f0", background: "#1e293b" },
    launch: { border: "#64748b", color: "#e2e8f0", background: "#1e293b" },
    unsupported: { border: "#3f3f46", color: "#a1a1aa", background: "#18181b" }
  };
  const palette = colors[kind] ?? colors.unsupported;
  return {
    background: palette.background,
    border: `1px solid ${palette.border}`,
    borderRadius: "999px",
    color: palette.color,
    flex: "0 0 auto",
    fontSize: "9px",
    fontWeight: 900,
    lineHeight: 1,
    maxWidth: "100%",
    overflow: "hidden",
    padding: "3px 5px",
    textOverflow: "ellipsis",
    textTransform: "uppercase",
    whiteSpace: "nowrap"
  };
}

function nextDeckyModSort(current: DeckyModSort): DeckyModSort {
  if (current === "profile") return "source";
  if (current === "source") return "az";
  if (current === "az") return "za";
  if (current === "za") return "enabled";
  return "profile";
}

function profileCountText(profile: Profile) {
  return `${profile.enabled_mod_count} on / ${profile.mod_count} total`;
}

function deckyModSortLabel(sort: DeckyModSort) {
  if (sort === "source") return "Source";
  if (sort === "az") return "A-Z";
  if (sort === "za") return "Z-A";
  if (sort === "enabled") return "Enabled First";
  return "Profile Order";
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
  if (sort === "popular") return "Most Popular";
  if (sort === "updated") return "Updated";
  if (sort === "name") return "A-Z";
  if (sort === "relevance") return "Relevant";
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
  const normalized = Number(value ?? 0);
  if (!Number.isFinite(normalized)) return "0";
  return normalized.toLocaleString(undefined, { maximumFractionDigits: 0, notation: normalized >= 10_000 ? "compact" : "standard" });
}

function formatBytes(value: number | undefined) {
  const bytes = Number(value ?? 0);
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";
  const units = ["B", "KB", "MB", "GB"];
  let size = bytes;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex += 1;
  }
  return `${size.toLocaleString(undefined, { maximumFractionDigits: size >= 10 || unitIndex === 0 ? 0 : 1 })} ${units[unitIndex]}`;
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
    percent,
    indeterminate,
    barWidth: indeterminate ? 36 : boundedTotal > 0 ? percent : 100,
    label
  };
}

function DeckyJobProgress({ job }: { job: Job }) {
  const progress = jobDownloadProgress(job);
  if (!progress) return null;
  return (
    <div style={{ display: "grid", gap: "5px", marginTop: "2px", minWidth: 0, width: "100%" }}>
      <div
        style={{
          background: "#111827",
          border: "1px solid #303741",
          borderRadius: "999px",
          height: "7px",
          overflow: "hidden",
          width: "100%"
        }}
      >
        <div
          style={{
            background: progress.indeterminate ? "linear-gradient(90deg, #38bdf8, #bae6fd, #38bdf8)" : "#7dd3fc",
            height: "100%",
            minWidth: "7px",
            width: `${progress.barWidth}%`
          }}
        />
      </div>
      <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 700, lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
        {progress.label}
      </div>
    </div>
  );
}

function DeckyJobIssueReview({ job }: { job: Job }) {
  const title = String(job.payload?.issue_title ?? "").trim();
  const message = String(job.payload?.issue_message ?? "").trim();
  const details = parseDeckyJobStringList(job.payload?.issue_details_json);
  const actions = parseDeckyJobStringList(job.payload?.issue_actions_json);
  if (!title && details.length === 0 && actions.length === 0) return null;
  return (
    <div style={{ background: "#241a0d", border: "1px solid #92400e", borderRadius: "7px", display: "grid", gap: "5px", marginTop: "2px", minWidth: 0, padding: "8px" }}>
      {title && <div style={{ color: "#fed7aa", fontSize: "12px", fontWeight: 900, lineHeight: 1.2, overflowWrap: "anywhere" }}>{title}</div>}
      {message && message !== job.message && <div style={{ color: "#fde68a", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>{message}</div>}
      {details.slice(0, 3).map((detail) => (
        <div key={detail} style={{ color: "#f8fafc", fontSize: "11px", lineHeight: 1.25, overflowWrap: "anywhere" }}>- {detail}</div>
      ))}
      {details.length > 3 && <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>{details.length - 3} more conflict detail{details.length - 3 === 1 ? "" : "s"}</div>}
      {actions.length > 0 && <div style={{ color: "#fde68a", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>{actions.join(" ")}</div>}
    </div>
  );
}

function parseDeckyJobStringList(value: string | undefined) {
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

function focusDeckyRef(ref: { current: HTMLElement | null }, label: string, logDetail: Record<string, string | number | boolean> = {}) {
  window.setTimeout(() => {
    const target = ref.current;
    if (!target) {
      void logFrontendEvent("decky focus ref missing", { label, ...logDetail });
      return;
    }
    target.focus();
    target.scrollIntoView({ block: "nearest", inline: "nearest" });
    void logFrontendEvent("decky focus ref applied", { label, ...logDetail });
  }, 80);
}

function unregisterSteamCallback(registration: unknown) {
  if (typeof registration === "function") {
    registration();
    return;
  }
  const value = registration as { unregister?: () => void; Unregister?: () => void } | null | undefined;
  if (typeof value?.unregister === "function") value.unregister();
  if (typeof value?.Unregister === "function") value.Unregister();
}

async function logFrontendEvent(message: string, detail: Record<string, string | number | boolean> = {}) {
  try {
    await call<[string, Record<string, string | number | boolean>], { ok: boolean }>("frontend_log", message, detail);
  } catch (_err) {
    // Frontend logging is best-effort and must not block Decky actions.
  }
}

function loggedObjectKeys(value: unknown, limit = 80) {
  return loggedObjectKeyList(value).slice(0, limit).join(",");
}

function loggedObjectKeyList(value: unknown) {
  if (!value || typeof value !== "object") return [];
  try {
    return Object.keys(value as Record<string, unknown>).sort();
  } catch (_err) {
    return [];
  }
}

function compactLogValue(value: string) {
  return value.length > 900 ? `${value.slice(0, 900)}...` : value;
}

function errorLogValue(error: unknown): string {
  if (error instanceof Error) return compactLogValue(error.message);
  if (typeof error === "string") return compactLogValue(error);
  try {
    return compactLogValue(JSON.stringify(error));
  } catch (_err) {
    return String(error);
  }
}

function compactUnknownLogValue(value: unknown): string {
  if (typeof value === "string") return compactLogValue(value);
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    return compactLogValue(JSON.stringify(value));
  } catch (_err) {
    return String(value);
  }
}

async function logSteamClientCapabilities() {
  const root = typeof SteamClient !== "undefined" ? (SteamClient as unknown as Record<string, unknown>) : null;
  if (!root) {
    await logFrontendEvent("steam client capabilities unavailable");
    return;
  }
  const detail: Record<string, string | number | boolean> = {
    top_level: compactLogValue(loggedObjectKeys(root))
  };
  for (const section of ["Apps", "GameSessions", "Workshop", "UGC", "RemoteStorage", "Cloud", "Overlay", "Browser", "BrowserView", "System", "URL", "WebChat"]) {
    detail[section] = compactLogValue(loggedObjectKeys(root[section]));
  }
  const appWorkshopMethods = loggedObjectKeyList(root.Apps).filter((key) => /workshop|subscrib|ugc/i.test(key));
  detail.AppsWorkshop = compactLogValue(appWorkshopMethods.join(","));
  await logFrontendEvent("steam client capabilities", detail);
}

async function startSteamBrowserNXMProbe(probeURL: string, selectedAppID = "") {
  const root = typeof SteamClient !== "undefined" ? SteamClient : undefined;
  const overlay = root?.Overlay;
  const detail: Record<string, string | number | boolean> = {
    probe_url: probeURL,
    selected_app_id: selectedAppID,
    has_overlay_register: typeof overlay?.RegisterForOverlayBrowserProtocols === "function",
    has_overlay_handle: typeof overlay?.HandleProtocolForOverlayBrowser === "function",
    has_navigation_external: typeof Navigation?.NavigateToExternalWeb === "function"
  };
  await logSteamClientCapabilities();
  await logFrontendEvent("steam browser nxm probe starting", detail);

  if (steamBrowserNXMProbeRegistration) {
    unregisterSteamCallback(steamBrowserNXMProbeRegistration);
    steamBrowserNXMProbeRegistration = null;
    await logFrontendEvent("steam browser nxm probe previous listener removed");
  }

  if (typeof overlay?.RegisterForOverlayBrowserProtocols === "function") {
    try {
      steamBrowserNXMProbeRegistration = overlay.RegisterForOverlayBrowserProtocols((payload) => {
        const appID = Number(payload?.unAppID ?? selectedAppID ?? 0);
        const scheme = String(payload?.strScheme ?? "");
        void logFrontendEvent("steam browser nxm protocol callback", {
          app_id: Number.isFinite(appID) ? appID : 0,
          scheme,
          added: Boolean(payload?.bAdded),
          payload: compactUnknownLogValue(payload)
        });
        if (scheme.toLowerCase() === "nxm" && typeof overlay.HandleProtocolForOverlayBrowser === "function") {
          try {
            overlay.HandleProtocolForOverlayBrowser(Number.isFinite(appID) ? appID : 0, scheme);
            void logFrontendEvent("steam browser nxm protocol handle invoked", { app_id: Number.isFinite(appID) ? appID : 0, scheme });
          } catch (err) {
            void logFrontendEvent("steam browser nxm protocol handle failed", { error: errorLogValue(err), scheme });
          }
        }
      });
      await logFrontendEvent("steam browser nxm probe listener registered");
    } catch (err) {
      await logFrontendEvent("steam browser nxm probe listener failed", { error: errorLogValue(err) });
    }
  } else {
    await logFrontendEvent("steam browser nxm probe listener unavailable");
  }

  try {
    Navigation.NavigateToExternalWeb(probeURL);
    await logFrontendEvent("steam browser nxm probe page opened", { probe_url: probeURL });
  } catch (err) {
    await logFrontendEvent("steam browser nxm probe page open failed", { error: errorLogValue(err), probe_url: probeURL });
  }
}

function looksLikeNXMURL(value: unknown): value is string {
  return typeof value === "string" && value.trim().toLowerCase().startsWith("nxm://");
}

function nexusBrowserCredentialRequired(message: string | undefined) {
  const normalized = String(message || "").toLowerCase();
  return normalized.includes("browser-generated") || normalized.includes("mod manager download") || normalized.includes("without visiting nexusmods.com");
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

async function captureNXMFromDMMBrowser(rawURL: string, appID: string, profileID: number, source: string) {
  const url = rawURL.trim();
  await logFrontendEvent("dmm browser nxm capture requested", {
    app_id: appID,
    profile_id: profileID,
    source,
    url
  });
  const result = await call<[string, string, number], { ok: boolean; error?: string; result?: { job?: Job } }>("add_captured_install", url, appID, profileID);
  if (!result.ok) {
    await logFrontendEvent("dmm browser nxm capture failed", {
      app_id: appID,
      profile_id: profileID,
      source,
      error: result.error || ""
    });
    toaster.toast({
      title: "DMM capture failed",
      body: result.error || "DMM could not capture the Nexus download link.",
      duration: 9000,
      critical: true,
      playSound: true,
      showToast: true
    });
    return;
  }
  const job = result.result?.job;
  if (job) markJobToastShown(job);
  await logFrontendEvent("dmm browser nxm captured", {
    app_id: appID,
    profile_id: profileID,
    source,
    job_id: job?.id || "",
    status: job?.status || ""
  });
  toaster.toast({
    title: "DMM captured Nexus download",
    body: job?.message || "The Nexus Mod Manager Download link was sent to DMM.",
    duration: 7000,
    critical: false,
    playSound: true,
    showToast: true
  });
  window.setTimeout(() => closeDMMNativeBrowserAfterCapture(source), 250);
}

function getDMMWindowRouter(): DMMWindowRouter | undefined {
  const typedRouter = Router as unknown as {
    WindowStore?: {
      GamepadUIMainWindowInstance?: DMMWindowRouter;
      SteamUIWindows?: DMMWindowRouter[];
    };
  };
  return typedRouter.WindowStore?.GamepadUIMainWindowInstance ?? typedRouter.WindowStore?.SteamUIWindows?.[0];
}

function rememberDMMDeckyReturnContext(request: DMMBrowserRequest | null) {
  const appID = request?.appID?.trim();
  if (!appID) return;
  pendingDMMDeckyReturnContext = { appID, tab: "games" };
  void logFrontendEvent("decky return context stored", {
    app_id: appID,
    source: request?.source || "",
    tab: "games"
  });
}

function consumeDMMDeckyReturnContext() {
  const context = pendingDMMDeckyReturnContext;
  pendingDMMDeckyReturnContext = null;
  if (context) {
    void logFrontendEvent("decky return context consumed", {
      app_id: context.appID,
      tab: context.tab
    });
  }
  return context;
}

function destroyActiveDMMNativeBrowser(source: string) {
  const browser = activeDMMNativeBrowser;
  if (!browser) return;
  activeDMMNativeBrowser = null;
  try {
    browser.m_browserView?.SetFocus?.(false);
    browser.Destroy?.();
    void logFrontendEvent("dmm native browser destroyed", { source });
  } catch (err) {
    void logFrontendEvent("dmm native browser destroy failed", { source, error: errorLogValue(err) });
  }
}

function navigateDMMRoute(path: string, source: string) {
  Navigation.Navigate(path);
  Navigation.CloseSideMenus();
  void logFrontendEvent("decky navigation requested", { source, path });
  window.setTimeout(() => {
    void logFrontendEvent("decky navigation observed", {
      source,
      path,
      current_path: window.location.pathname
    });
  }, 300);
}

function closeDMMNativeBrowserAfterCapture(source: string) {
  const currentPath = window.location.pathname;
  const onBrowserRoute = currentPath.endsWith(DMM_BROWSER_ROUTE);
  const request = activeDMMBrowserRequest;
  try {
    rememberDMMDeckyReturnContext(request);
    destroyActiveDMMNativeBrowser(`nxm-captured:${source}`);
    activeDMMBrowserRequest = null;
    void logFrontendEvent("dmm native browser closed after nxm capture", {
      app_id: request?.appID || "",
      source,
      path: currentPath,
      navigated_to_dmm: onBrowserRoute
    });
    if (request?.appID || onBrowserRoute) {
      navigateDMMRoute(DMM_DECKY_ROUTE, "nxm-captured");
    }
  } catch (err) {
    void logFrontendEvent("dmm native browser close after nxm capture failed", {
      source,
      path: currentPath,
      error: errorLogValue(err)
    });
  }
}

function registerDMMNativeBrowserEvents(browser: DMMNativeBrowser, request: DMMBrowserRequest) {
  const handledNXMURLs = new Set<string>();
  const captureOnce = (url: unknown, eventName: string) => {
    if (!looksLikeNXMURL(url)) return;
    const normalized = url.trim();
    if (handledNXMURLs.has(normalized)) return;
    handledNXMURLs.add(normalized);
    void captureNXMFromDMMBrowser(normalized, request.appID, request.profileID, `native-${eventName}`);
  };
  const browserView = browser.m_browserView;
  browserView.SetName?.(request.title);
  browserView.SetBlockedProtocols?.("nxm;nxm-protocol");
  browserView.SetSteamURLCallback?.((url) => {
    void logFrontendEvent("dmm native browser steam url", { source: request.source, url });
    captureOnce(url, "steam-url");
  });
  const logEvent = (eventName: string, args: unknown[]) => {
    void logFrontendEvent(`dmm native browser ${eventName}`, {
      source: request.source,
      args: compactUnknownLogValue(args)
    });
    for (const arg of args) captureOnce(arg, eventName);
  };
  for (const eventName of ["start-request", "start-loading", "new-tab", "blocked-request", "load-error", "finished-request", "history-changed", "set-title", "focus-changed", "before-close"]) {
    browserView.on?.(eventName, (...args: unknown[]) => logEvent(eventName, args));
  }
}

async function openDMMBrowserViewCapture(initialURL: string, options: { appID?: string; profileID?: number; source?: string; title?: string } = {}) {
  const appID = options.appID || "";
  const profileID = options.profileID || 0;
  const source = options.source || "dmm-native-browser";
  const title = options.title || "DMM Browser";
  const request: DMMBrowserRequest = { appID, initialURL, profileID, source, title };
  const windowRouter = getDMMWindowRouter();
  activeDMMBrowserRequest = request;
  await logSteamClientCapabilities();
  await logFrontendEvent("dmm native browser opening", {
    app_id: appID,
    profile_id: profileID,
    source,
    url: initialURL,
    has_window_router: Boolean(windowRouter),
    has_create_browser_view: typeof windowRouter?.CreateBrowserView === "function",
    has_header_store: Boolean(windowRouter?.HeaderStore),
    has_browser_container: Boolean(DMMBrowserContainer)
  });
  if (!DMMBrowserContainer) {
    toaster.toast({
      title: "DMM browser unavailable",
      body: "Steam's native BrowserContainer component could not be found.",
      duration: 9000,
      critical: true,
      playSound: true,
      showToast: true
    });
    return false;
  }
  if (typeof windowRouter?.CreateBrowserView !== "function") {
    toaster.toast({
      title: "DMM browser unavailable",
      body: "Steam's Gamepad UI browser factory is unavailable.",
      duration: 9000,
      critical: true,
      playSound: true,
      showToast: true
    });
    return false;
  }

  try {
    destroyActiveDMMNativeBrowser(source);
    const browser = windowRouter.CreateBrowserView(DMM_NATIVE_BROWSER_NAME);
    activeDMMNativeBrowser = browser;
    registerDMMNativeBrowserEvents(browser, request);
    browser.m_browserView?.SetVisible?.(true);
    browser.m_browserView?.SetFocus?.(true);
    windowRouter.HeaderStore?.SetCurrentBrowserAndBackstack?.(browser, true);
    browser.LoadURL?.(initialURL);
    navigateDMMRoute(DMM_BROWSER_ROUTE, source);
    await logFrontendEvent("dmm native browser opened", {
      source,
      route: DMM_BROWSER_ROUTE,
      url: initialURL,
      has_load_url: typeof browser.LoadURL === "function"
    });
    return true;
  } catch (err) {
    await logFrontendEvent("dmm native browser open failed", { source, url: initialURL, error: errorLogValue(err) });
    toaster.toast({
      title: "DMM browser failed",
      body: errorLogValue(err),
      duration: 9000,
      critical: true,
      playSound: true,
      showToast: true
    });
    return false;
  }
}

async function handleDeckyBrowserOpenEvent(event: DomainEvent) {
  if (handledDeckyBrowserOpenEvents.has(event.id)) return;
  handledDeckyBrowserOpenEvents.add(event.id);
  if (!isDeckyBrowserOpenPayload(event.payload)) {
    await logFrontendEvent("decky browser open skipped invalid payload", { event_id: event.id });
    return;
  }
  const payload = event.payload;
  const expiresAt = payload.expires_at ? Date.parse(payload.expires_at) : NaN;
  if (Number.isFinite(expiresAt) && Date.now() > expiresAt) {
    await logFrontendEvent("decky browser open skipped expired event", {
      event_id: event.id,
      app_id: event.app_id || "",
      expires_at: payload.expires_at || ""
    });
    return;
  }
  const appID = event.app_id || "";
  const profileID = typeof payload.profile_id === "number" && Number.isFinite(payload.profile_id) ? payload.profile_id : 0;
  await logFrontendEvent("decky browser open event received", {
    event_id: event.id,
    app_id: appID,
    profile_id: profileID,
    source: payload.source || "event",
    url: payload.url
  });
  const opened = await openDMMBrowserViewCapture(payload.url, {
    appID,
    profileID,
    source: payload.source || "event",
    title: payload.title || "DMM Browser"
  });
  if (!opened) {
    await logFrontendEvent("decky browser open event failed", {
      event_id: event.id,
      app_id: appID,
      source: payload.source || "event"
    });
  }
}

function isNotifiableJob(job: Job) {
  return ["captured-install", "installer-choice", "deploy", "purge", "repair", "recover-downloads", "rollback", "steam-workshop-action", "extension-notice"].includes(job.type);
}

function isJob(value: unknown): value is Job {
  return Boolean(value && typeof value === "object" && typeof (value as Job).id === "string" && typeof (value as Job).type === "string");
}

function jobStatusShouldToast(job: Job) {
  if (job.type === "captured-install") {
    return ["queued", "waiting", "running", "completed", "failed"].includes(job.status);
  }
  if (job.type === "installer-choice" || job.type === "extension-notice") {
    return ["waiting", "completed", "failed"].includes(job.status);
  }
  if (["deploy", "purge", "repair", "recover-downloads", "rollback", "steam-workshop-action"].includes(job.type)) {
    return ["completed", "failed"].includes(job.status);
  }
  return ["completed", "failed"].includes(job.status);
}

function isUISettings(value: unknown): value is UISettings {
  return Boolean(value && typeof value === "object");
}

function isDeckyBrowserOpenPayload(value: unknown): value is DeckyBrowserOpenPayload {
  if (!value || typeof value !== "object") return false;
  const payload = value as DeckyBrowserOpenPayload;
  return typeof payload.url === "string" && payload.url.trim().length > 0;
}

function diagnosticsTerminalText(diagnostics: Diagnostics | null): string {
  if (!diagnostics) return "Loading logs...";
  const entries = Object.entries(diagnostics.logs);
  if (entries.length === 0) return "No log files were reported.";
  return entries
    .map(([name, log]) => {
      const title = `==> ${name} (${log.path || "unknown path"})`;
      const body = log.tail?.trimEnd() || "No log entries.";
      return `${title}\n${body}`;
    })
    .join("\n\n");
}

function updateResultMessage(result: UpdateResult): string {
  const parts = [
    result.message || (result.ok ? "Update installer started." : "Update failed."),
    result.error && result.error !== result.message ? `Error: ${result.error}` : "",
    result.log ? `Log: ${result.log}` : "",
    result.package ? `Package: ${result.package}` : "",
    result.url ? `URL: ${result.url}` : ""
  ].filter((part) => part.trim() !== "");
  return parts.join("\n");
}

function storedJobToastState(jobID: string) {
  try {
    return window.localStorage.getItem(`${DMM_TOAST_STORAGE_PREFIX}${jobID}`);
  } catch (_err) {
    return null;
  }
}

function rememberJobToastState(jobID: string, stateKey: string) {
  try {
    window.localStorage.setItem(`${DMM_TOAST_STORAGE_PREFIX}${jobID}`, stateKey);
  } catch (_err) {
    // Decky can deny localStorage in development contexts; in-memory de-dupe still protects the active instance.
  }
}

function markJobToastShown(job: Job) {
  const stateKey = `${job.status}:${job.message || ""}`;
  notifiedJobStates.set(job.id, stateKey);
  rememberJobToastState(job.id, stateKey);
}

function installerForCandidate(candidate: InstallCandidate): FomodInstaller | null {
  if (!candidate.installer_json) return null;
  try {
    return JSON.parse(candidate.installer_json) as FomodInstaller;
  } catch (_err) {
    return null;
  }
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

function fomodGroupValid(group: FomodGroup, selections: Record<string, string[]>) {
  if (fomodGroupIsText(group)) {
    return !group.required || (selections[group.id]?.[0] ?? "").trim() !== "";
  }
  const selected = (selections[group.id] ?? []).filter((id) => {
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

function fomodStepValid(step: FomodStep | undefined, selections: Record<string, string[]>) {
  if (!step) return true;
  return (step.groups ?? []).every((group) => fomodGroupValid(group, selections));
}

function fomodInstallerValid(installer: FomodInstaller | null, selections: Record<string, string[]>) {
  if (!installer) return false;
  return visibleFomodSteps(installer).every((step) => fomodStepValid(step, selections));
}

function storedFomodSelections(candidate: InstallCandidate): Record<string, string[]> | null {
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

async function maybeShowJobToast(job: Job, { seed = false, source = "event" } = {}) {
  if (!isNotifiableJob(job)) return;
  if (!jobStatusShouldToast(job)) return;
  const stateKey = `${job.status}:${job.message || ""}`;
  const previous = notifiedJobStates.get(job.id) ?? storedJobToastState(job.id);
  notifiedJobStates.set(job.id, stateKey);
  const updatedAt = Date.parse(job.updated_at || "");
  const recent = Number.isFinite(updatedAt) && Date.now() - updatedAt < 120_000;
  const requireRecent = seed || source === "event" || source === "event-snapshot";
  if (previous !== stateKey && (!requireRecent || recent)) {
    rememberJobToastState(job.id, stateKey);
    await logFrontendEvent("job toast shown", { job_id: job.id, status: job.status, seed, recent, type: job.type, source });
    showJobToast(job);
  }
}

async function maybeShowDeckyActionToast(job: Job | null | undefined, source = "decky-action") {
  if (!job) return;
  await maybeShowJobToast(job, { source });
}

async function seedJobNotifications({ seed = false } = {}) {
  try {
    const result = await call<[], { ok: boolean; error?: string; jobs: Job[] }>("jobs");
    if (!result.ok) {
      await logFrontendEvent("job seed returned not ok", { error: result.error || "" });
      return;
    }
    for (const job of result.jobs) {
      await maybeShowJobToast(job, { seed, source: "seed" });
    }
  } catch (_err) {
    await logFrontendEvent("job seed failed", { error: _err instanceof Error ? _err.message : String(_err) });
  }
}

async function syncLaunchActions(options: { force?: boolean; sink?: LaunchResultSink } = {}) {
  try {
    const result = await call<[], { ok: boolean; error?: string; actions: LaunchStatus[] }>("launch_actions");
    if (!result.ok) {
      await logFrontendEvent("launch action sync returned not ok", { error: result.error || "" });
      return;
    }
    if (result.actions.length > 0) {
      await logFrontendEvent("launch action sync found actions", { count: result.actions.length });
    }
    for (const launchStatus of result.actions) {
      const action = launchStatus.action;
      if (!action || action.type !== "set-steam-launch-options") continue;
      if (!launchStatus.can_configure || launchStatus.configured) continue;
      const actionKey = `${action.app_id}:${action.tool_id}:${action.desired_options}`;
      if (completedLaunchActions.has(actionKey)) continue;
      const now = Date.now();
      const previousAttempt = launchActionAttempts.get(actionKey) ?? 0;
      if (!options.force && previousAttempt > 0 && now - previousAttempt < 30_000) continue;
      launchActionAttempts.set(actionKey, now);

      const appid = Number.parseInt(action.app_id, 10);
      const steamApps = typeof SteamClient !== "undefined" ? SteamClient?.Apps : undefined;
      if (!Number.isFinite(appid) || typeof steamApps?.SetAppLaunchOptions !== "function") {
        const message = "Steam launch-option API is unavailable in this Decky context.";
        options.sink?.(message);
        await logFrontendEvent("launch action steam api unavailable", { app_id: action.app_id, tool_id: action.tool_id });
        await call<[string, Record<string, string | boolean>], { ok: boolean }>("record_launch_action", action.app_id, {
          applied: false,
          error: message,
          source: "decky-auto"
        });
        continue;
      }

      try {
        await logFrontendEvent("launch action applying", { app_id: action.app_id, tool_id: action.tool_id });
        steamApps.SetAppLaunchOptions(appid, action.desired_options);
        const report = await call<[string, Record<string, string | boolean>], { ok: boolean; status?: LaunchStatus }>("record_launch_action", action.app_id, {
          applied: true,
          current_options: action.desired_options,
          source: "decky-auto"
        });
        if (report.ok && report.status?.configured) {
          completedLaunchActions.add(actionKey);
          const toolName = launchStatus.tool?.name || action.tool_id;
          const message = `${toolName} launch option configured for ${launchStatus.app_id}.`;
          options.sink?.(message);
          showLaunchToast("DMM launch tool configured", message);
          await logFrontendEvent("launch action applied", { app_id: action.app_id, tool_id: action.tool_id });
          continue;
        }
        await logFrontendEvent("launch action still pending after steam api call", { app_id: action.app_id, tool_id: action.tool_id });
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        options.sink?.(message);
        showLaunchToast("DMM launch tool failed", message, true);
        await logFrontendEvent("launch action failed", { app_id: action.app_id, tool_id: action.tool_id, error: message });
        await call<[string, Record<string, string | boolean>], { ok: boolean }>("record_launch_action", action.app_id, {
          applied: false,
          error: message,
          source: "decky-auto"
        });
      }
    }
  } catch (_err) {
    await logFrontendEvent("launch action sync failed", { error: _err instanceof Error ? _err.message : String(_err) });
  }
}

function workshopIDFromClientItem(item: SteamWorkshopClientItem): string {
  const raw = item.ulPublishedFileID ?? item.publishedfileid ?? item.published_file_id ?? item.details?.publishedfileid ?? item.details?.published_file_id ?? "";
  return String(raw).trim();
}

function workshopIDFromClientDetail(detail: SteamWorkshopClientItemDetails): string {
  const raw = detail.publishedfileid ?? detail.published_file_id ?? "";
  return String(raw).trim();
}

function workshopTitleFromClientItem(item: SteamWorkshopClientItem): string {
  const title = item.title ?? item.strTitle ?? item.details?.title ?? item.details?.strTitle ?? "";
  return String(title).trim();
}

function workshopDisabledState(item: SteamWorkshopClientItem): { known: boolean; value: boolean } {
  for (const key of ["bDisabledLocally", "disabled_locally", "bDisabled", "disabled"] as const) {
    const value = item[key];
    if (typeof value === "boolean") return { known: true, value };
    if (typeof value === "number") return { known: true, value: value !== 0 };
  }
  return { known: false, value: false };
}

function workshopRawJSON(item: SteamWorkshopClientItem): string {
  const raw: Record<string, string | number | boolean> = {};
  for (const key of ["unAppID", "appid", "ulPublishedFileID", "publishedfileid", "published_file_id", "title", "strTitle", "bDisabledLocally", "bDisabled", "disabled", "disabled_locally", "load_order", "time_subscribed", "time_updated", "file_size"] as const) {
    const value = item[key];
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      raw[key] = value;
    }
  }
  if (item.details) {
    const detail: Record<string, string | number | boolean> = {};
    for (const key of ["publishedfileid", "published_file_id", "title", "strTitle", "file_size", "preview_url", "short_description", "consumer_appid"] as const) {
      const value = item.details[key];
      if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
        detail[`details.${key}`] = typeof value === "string" ? compactLogValue(value) : value;
      }
    }
    Object.assign(raw, detail);
  }
  return JSON.stringify(raw);
}

function workshopDetailShape(item: SteamWorkshopClientItem): string {
  const detail = item.details;
  return compactLogValue(JSON.stringify({
    id: workshopIDFromClientItem(item),
    item_keys: loggedObjectKeys(item, 24),
    detail_keys: loggedObjectKeys(detail, 24),
    has_title: workshopTitleFromClientItem(item) !== ""
  }));
}

async function enrichWorkshopItemDetails(appID: string, appid: number, steamApps: NonNullable<typeof SteamClient>["Apps"], items: SteamWorkshopClientItem[]) {
  if (items.length === 0 || typeof steamApps?.GetSubscribedWorkshopItemDetails !== "function") return items;
  const byID = new Map<string, SteamWorkshopClientItem>();
  const missingDetails: string[] = [];
  for (const item of items) {
    const id = workshopIDFromClientItem(item);
    if (!id || byID.has(id)) continue;
    byID.set(id, item);
    if (!item.details || !workshopTitleFromClientItem(item)) missingDetails.push(id);
  }
  if (missingDetails.length === 0) return items;
  try {
    const details = await steamApps.GetSubscribedWorkshopItemDetails(appid, missingDetails);
    for (const detail of details ?? []) {
      const id = workshopIDFromClientDetail(detail);
      const existing = id ? byID.get(id) : undefined;
      if (existing) {
        existing.details = { ...(existing.details ?? {}), ...detail };
      }
    }
    const titled = items.filter((item) => workshopTitleFromClientItem(item) !== "").length;
    const sample = items.find((item) => item.details) ?? items[0];
    await logFrontendEvent("workshop detail sync completed", {
      app_id: appID,
      requested: missingDetails.length,
      returned: details?.length ?? 0,
      titled,
      sample: sample ? workshopDetailShape(sample) : ""
    });
  } catch (err) {
    await logFrontendEvent("workshop detail sync failed", { app_id: appID, error: err instanceof Error ? err.message : String(err) });
  }
  return items;
}

function mergeWorkshopItems(appID: string, subscribed: SteamWorkshopClientItem[], downloaded: SteamWorkshopClientItem[]): WorkshopItem[] {
  const byID = new Map<string, WorkshopItem>();
  subscribed.forEach((item, index) => {
    const id = workshopIDFromClientItem(item);
    if (!id) return;
    const disabled = workshopDisabledState(item);
    byID.set(id, {
      steam_app_id: appID,
      published_file_id: id,
      title: workshopTitleFromClientItem(item),
      subscribed: true,
      downloaded: false,
      disabled_locally: disabled.value,
      disabled_known: disabled.known,
      position: index,
      raw_json: workshopRawJSON(item)
    });
  });
  downloaded.forEach((item, index) => {
    const id = workshopIDFromClientItem(item);
    if (!id) return;
    const disabled = workshopDisabledState(item);
    const existing = byID.get(id);
    byID.set(id, {
      steam_app_id: appID,
      published_file_id: id,
      title: existing?.title || workshopTitleFromClientItem(item),
      subscribed: existing?.subscribed ?? false,
      downloaded: true,
      disabled_locally: disabled.known ? disabled.value : existing?.disabled_locally ?? false,
      disabled_known: disabled.known || existing?.disabled_known || false,
      position: existing?.position ?? subscribed.length + index,
      raw_json: existing?.raw_json || workshopRawJSON(item)
    });
  });
  return Array.from(byID.values()).sort((a, b) => a.position - b.position || a.published_file_id.localeCompare(b.published_file_id));
}

function shouldSyncWorkshopStateForApp(appID: string, force = false) {
  if (force) return true;
  const now = Date.now();
  const previous = workshopStateSyncLastAt.get(appID) ?? 0;
  if (previous > 0 && now - previous < 15_000) return false;
  workshopStateSyncLastAt.set(appID, now);
  return true;
}

async function syncWorkshopStateForApp(appID: string, options: { force?: boolean } = {}) {
  if (!shouldSyncWorkshopStateForApp(appID, Boolean(options.force))) return false;
  const appid = Number.parseInt(appID, 10);
  const steamApps = typeof SteamClient !== "undefined" ? SteamClient?.Apps : undefined;
  if (!Number.isFinite(appid) || !steamApps) return false;
  if (typeof steamApps.GetSubscribedWorkshopItems !== "function" && typeof steamApps.GetDownloadedWorkshopItems !== "function") {
    await logFrontendEvent("workshop sync steam api unavailable", { app_id: appID });
    return false;
  }

  let subscribed: SteamWorkshopClientItem[] = [];
  let downloaded: SteamWorkshopClientItem[] = [];
  let readFailed = false;
  try {
    if (typeof steamApps.GetSubscribedWorkshopItems === "function") {
      subscribed = await steamApps.GetSubscribedWorkshopItems(appid);
    }
  } catch (err) {
    readFailed = true;
    await logFrontendEvent("workshop subscribed sync failed", { app_id: appID, error: errorLogValue(err) });
  }
  try {
    if (typeof steamApps.GetDownloadedWorkshopItems === "function") {
      downloaded = await steamApps.GetDownloadedWorkshopItems(appid);
    }
  } catch (err) {
    readFailed = true;
    await logFrontendEvent("workshop downloaded sync failed", { app_id: appID, error: errorLogValue(err) });
  }
  if (readFailed) {
    await logFrontendEvent("workshop sync skipped after steam read failure", { app_id: appID });
    return false;
  }

  await enrichWorkshopItemDetails(appID, appid, steamApps, [...(subscribed ?? []), ...(downloaded ?? [])]);
  const items = mergeWorkshopItems(appID, subscribed ?? [], downloaded ?? []);
  const result = await call<[string, WorkshopItem[]], { ok: boolean; error?: string }>("sync_workshop", appID, items);
  if (!result.ok) {
    await logFrontendEvent("workshop sync backend rejected", { app_id: appID, error: result.error || "" });
    return false;
  }
  await logFrontendEvent("workshop state synced", { app_id: appID, subscribed: subscribed.length, downloaded: downloaded.length, items: items.length });
  return true;
}

async function seedWorkshopStateFromGames() {
  try {
    const result = await call<[], { ok: boolean; error?: string; games: ManagedGame[] }>("games");
    if (!result.ok) {
      await logFrontendEvent("workshop startup sync games unavailable", { error: result.error || "" });
      return;
    }
    const targets = (result.games ?? [])
      .filter((game) => game.steam_workshop?.detected && game.steam_workshop.management_supported && (game.steam_workshop.item_count ?? 0) > 0)
      .slice(0, 8);
    if (targets.length === 0) {
      await logFrontendEvent("workshop startup sync skipped", { reason: "no managed workshop games" });
      return;
    }
    for (const game of targets) {
      await syncWorkshopStateForApp(game.app_id, { force: true });
    }
    await logFrontendEvent("workshop startup sync completed", { games: targets.length });
  } catch (err) {
    await logFrontendEvent("workshop startup sync failed", { error: err instanceof Error ? err.message : String(err) });
  }
}

async function executeWorkshopAction(job: WorkshopActionJob) {
  const appID = String(job.payload?.app_id ?? "").trim();
  const itemID = String(job.payload?.item_id ?? "").trim();
  const kind = String(job.payload?.kind ?? "").trim();
  const appid = Number.parseInt(appID, 10);
  const steamApps = typeof SteamClient !== "undefined" ? SteamClient?.Apps : undefined;
  if (!Number.isFinite(appid) || !steamApps) {
    throw new Error("Steam Workshop API is unavailable in this Decky context.");
  }
  if (kind === "order") {
    if (typeof steamApps.SetWorkshopItemsLoadOrder !== "function") throw new Error("Steam Workshop load-order API is unavailable.");
    const itemIDs = workshopOrderIDsFromJob(job);
    if (itemIDs.length === 0) throw new Error("Steam Workshop load order did not include any item IDs.");
    await Promise.resolve(steamApps.SetWorkshopItemsLoadOrder(appid, itemIDs));
    return;
  }
  if (!itemID) {
    throw new Error("Steam Workshop action did not include an item ID.");
  }
  if (kind === "enable") {
    if (typeof steamApps.SetWorkshopItemsDisabledLocally !== "function") throw new Error("Steam Workshop enable API is unavailable.");
    await Promise.resolve(steamApps.SetWorkshopItemsDisabledLocally(appid, [itemID], false));
    return;
  }
  if (kind === "disable") {
    if (typeof steamApps.SetWorkshopItemsDisabledLocally !== "function") throw new Error("Steam Workshop disable API is unavailable.");
    await Promise.resolve(steamApps.SetWorkshopItemsDisabledLocally(appid, [itemID], true));
    return;
  }
  if (kind === "subscribe") {
    if (typeof steamApps.SubscribeWorkshopItem !== "function") throw new Error("Steam Workshop subscribe API is unavailable.");
    await Promise.resolve(steamApps.SubscribeWorkshopItem(appid, itemID, true));
    if (typeof steamApps.DownloadWorkshopItem === "function") {
      await Promise.resolve(steamApps.DownloadWorkshopItem(appid, itemID, true));
    }
    return;
  }
  if (kind === "unsubscribe") {
    if (typeof steamApps.SubscribeWorkshopItem !== "function") throw new Error("Steam Workshop unsubscribe API is unavailable.");
    await Promise.resolve(steamApps.SubscribeWorkshopItem(appid, itemID, false));
    return;
  }
  throw new Error(`Unsupported Steam Workshop action: ${kind}`);
}

function workshopOrderIDsFromJob(job: WorkshopActionJob): string[] {
  const raw = String(job.payload?.item_ids_json ?? "").trim();
  if (!raw) return [];
  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.map((item) => String(item ?? "").trim()).filter(Boolean);
  } catch (_err) {
    return [];
  }
}

async function syncWorkshopActions() {
  try {
    const result = await call<[], { ok: boolean; error?: string; actions: WorkshopActionJob[] }>("workshop_actions");
    if (!result.ok) {
      await logFrontendEvent("workshop action sync returned not ok", { error: result.error || "" });
      return;
    }
    if (result.actions.length > 0) {
      await logFrontendEvent("workshop action sync found actions", { count: result.actions.length });
    }
    for (const action of result.actions) {
      const key = `${action.id}:${action.payload?.app_id || ""}:${action.payload?.item_id || ""}:${action.payload?.kind || ""}`;
      if (completedWorkshopActions.has(key)) continue;
      const now = Date.now();
      const previousAttempt = workshopActionAttempts.get(key) ?? 0;
      if (previousAttempt > 0 && now - previousAttempt < 30_000) continue;
      workshopActionAttempts.set(key, now);

      const started = await call<[string], { ok: boolean; error?: string; proceed?: boolean; job?: WorkshopActionJob }>("start_workshop_action", action.id);
      if (!started.ok || !started.proceed) {
        if (!started.ok) await logFrontendEvent("workshop action start returned not ok", { job_id: action.id, error: started.error || "" });
        continue;
      }
      try {
        await logFrontendEvent("workshop action applying", { job_id: action.id, app_id: action.payload?.app_id || "", item_id: action.payload?.item_id || "", kind: action.payload?.kind || "" });
        await executeWorkshopAction(action);
        await new Promise((resolve) => window.setTimeout(resolve, 900));
        if (action.payload?.app_id) {
          await syncWorkshopStateForApp(action.payload.app_id, { force: true });
        }
        const report = await call<[string, Record<string, string | boolean>], { ok: boolean; error?: string; job?: Job }>("record_workshop_action", action.id, {
          applied: true,
          source: "decky-auto"
        });
        if (report.ok) {
          completedWorkshopActions.add(key);
          showLaunchToast("DMM Workshop action applied", action.payload?.action_name || "Steam Workshop action applied");
          await logFrontendEvent("workshop action applied", { job_id: action.id, app_id: action.payload?.app_id || "", item_id: action.payload?.item_id || "", kind: action.payload?.kind || "" });
        } else {
          await logFrontendEvent("workshop action report returned not ok", { job_id: action.id, error: report.error || "" });
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        showLaunchToast("DMM Workshop action failed", message, true);
        await logFrontendEvent("workshop action failed", { job_id: action.id, error: message });
        await call<[string, Record<string, string | boolean>], { ok: boolean }>("record_workshop_action", action.id, {
          applied: false,
          error: message,
          source: "decky-auto"
        });
      }
    }
  } catch (err) {
    await logFrontendEvent("workshop action sync failed", { error: err instanceof Error ? err.message : String(err) });
  }
}

function InstallerChoiceModal(props: { appID: string; candidate: InstallCandidate; profileID?: number; closeModal: () => void; onApplied: () => void }) {
  const [candidate, setCandidate] = useState<InstallCandidate>(props.candidate);
  const installer = installerForCandidate(candidate);
  const [selections, setSelections] = useState<Record<string, string[]>>(() => storedFomodSelections(props.candidate) ?? {});
  const [stepIndex, setStepIndex] = useState(0);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const selectedChoices = selectionCount(selections);
  const steps = installer ? visibleFomodSteps(installer) : [];
  const currentStepIndex = steps.length === 0 ? 0 : Math.max(0, Math.min(stepIndex, steps.length - 1));
  const currentStep = steps[currentStepIndex];
  const currentStepReady = fomodStepValid(currentStep, selections);
  const installerReady = fomodInstallerValid(installer, selections);
  const lastStep = currentStepIndex >= steps.length - 1;
  const primaryActionDisabled = !installer || !currentStepReady || (lastStep && !installerReady) || busy;
  const primaryActionLabel = busy ? "Applying" : lastStep ? "Apply Choices" : "Next";

  useEffect(() => {
    setCandidate(props.candidate);
    setSelections(storedFomodSelections(props.candidate) ?? {});
    setStepIndex(0);
  }, [props.candidate.id]);

  useEffect(() => {
    const max = Math.max(0, steps.length - 1);
    if (stepIndex > max) setStepIndex(max);
  }, [steps.length, stepIndex]);

  function pluginSelected(group: FomodGroup, plugin: FomodPlugin) {
    return (selections[group.id] ?? []).includes(plugin.id);
  }

  function clearGroupSelection(group: FomodGroup) {
    const next = { ...selections, [group.id]: [] };
    setSelections(next);
    void saveChoices(next);
  }

  function groupTextValue(group: FomodGroup) {
    return selections[group.id]?.[0] ?? "";
  }

  function setGroupTextSelection(group: FomodGroup, value: string, save: boolean) {
    const next = { ...selections, [group.id]: value.trim() === "" ? [] : [value] };
    setSelections(next);
    if (save) void saveChoices(next);
  }

  function setPluginSelection(group: FomodGroup, plugin: FomodPlugin, checked: boolean) {
    const type = fomodGroupType(group);
    if (fomodPluginLocked(group, plugin)) return;
    setSelections((current) => {
      const next = { ...current };
      if (type === "selectexactlyone" || type === "selectatmostone") {
        next[group.id] = checked ? [plugin.id] : [];
        void saveChoices(next);
        return next;
      }
      const selected = new Set(next[group.id] ?? []);
      if (checked) selected.add(plugin.id);
      else selected.delete(plugin.id);
      next[group.id] = Array.from(selected);
      void saveChoices(next);
      return next;
    });
  }

  function togglePluginSelection(group: FomodGroup, plugin: FomodPlugin) {
    if (busy || fomodPluginLocked(group, plugin)) return;
    const type = fomodGroupType(group);
    const selected = pluginSelected(group, plugin);
    if (type === "selectexactlyone") {
      setPluginSelection(group, plugin, true);
      return;
    }
    if (type === "selectatmostone") {
      setPluginSelection(group, plugin, !selected);
      return;
    }
    setPluginSelection(group, plugin, !selected);
  }

  async function saveChoices(nextSelections: Record<string, string[]>) {
    try {
      const result = await call<[string, number, Record<string, string[]>], { ok: boolean; error?: string; candidate?: InstallCandidate }>(
        "save_install_candidate_choices",
        props.appID,
        candidate.id,
        nextSelections
      );
      if (!result.ok) {
        await logFrontendEvent("installer choice modal save failed", { app_id: props.appID, candidate_id: candidate.id, error: result.error || "" });
        return;
      }
      if (result.candidate) setCandidate(result.candidate);
    } catch (err) {
      await logFrontendEvent("installer choice modal save threw", { app_id: props.appID, candidate_id: candidate.id, error: err instanceof Error ? err.message : String(err) });
    }
  }

  async function applyChoices() {
    if (!installer || busy) return;
    if (installerRequiresSelections(installer) && Object.keys(selections).length === 0) {
      setMessage("Installer choices are missing from backend state. Retry this installer item so DMM can rebuild the choices.");
      return;
    }
    if (!installerReady) {
      setMessage("One or more installer steps still need a valid selection.");
      return;
    }
    setBusy(true);
    setMessage("");
    const candidateKey = String(candidate.id);
    applyingInstallerChoiceCandidates.add(candidateKey);
    try {
      const targetProfileID = props.profileID || candidate.target_profile_id || 0;
      const result = await call<[string, number, Record<string, string[]>, number], { ok: boolean; error?: string; result?: { job?: Job; mod?: ManagedMod } }>(
        "apply_install_candidate",
        props.appID,
        candidate.id,
        selections,
        targetProfileID
      );
      if (!result.ok) {
        applyingInstallerChoiceCandidates.delete(candidateKey);
        setMessage(result.error || "Unable to apply installer choices.");
        await logFrontendEvent("installer choice modal apply failed", { app_id: props.appID, candidate_id: candidate.id, profile_id: targetProfileID, error: result.error || "" });
        return;
      }
      const applyJob = result.result?.job;
      await maybeShowDeckyActionToast(applyJob, "decky-installer-choice");
      if (applyJob?.status === "failed") {
        applyingInstallerChoiceCandidates.delete(candidateKey);
        setMessage(applyJob.message || "Unable to apply installer choices.");
        await logFrontendEvent("installer choice modal apply job failed", {
          app_id: props.appID,
          candidate_id: candidate.id,
          job_id: applyJob.id,
          message: applyJob.message || ""
        });
        return;
      }
      await logFrontendEvent("installer choice modal applied", { app_id: props.appID, candidate_id: candidate.id, profile_id: targetProfileID });
      props.onApplied();
      props.closeModal();
    } catch (err) {
      applyingInstallerChoiceCandidates.delete(candidateKey);
      const error = err instanceof Error ? err.message : String(err);
      setMessage(error);
      await logFrontendEvent("installer choice modal apply threw", { app_id: props.appID, candidate_id: candidate.id, error });
    } finally {
      setBusy(false);
    }
  }

  function continueOrApply() {
    if (!installer || busy || !currentStepReady) return;
    if (!lastStep) {
      setStepIndex(currentStepIndex + 1);
      return;
    }
    void applyChoices();
  }

  return (
    <ModalRoot onCancel={props.closeModal} bAllowFullSize bHideCloseIcon>
      <style>{deckyRuntimeStyles}</style>
      <Focusable
        flow-children="down"
        navEntryPreferPosition={NavEntryPositionPreferences.PREFERRED_CHILD}
        style={{
          boxSizing: "border-box",
          color: "#f8fafc",
          display: "grid",
          gap: "10px",
          gridTemplateRows: "auto minmax(0, 1fr)",
          height: "min(720px, calc(100vh - 96px))",
          maxHeight: "calc(100vh - 96px)",
          minHeight: "420px",
          overflow: "hidden",
          padding: "4px",
          width: "100%"
        }}
      >
        <div style={{ display: "grid", gap: "4px", minWidth: 0, width: "100%" }}>
          <div style={{ color: "#f8fafc", fontSize: "16px", fontWeight: 900, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{candidate.name}</div>
          <div style={{ color: currentStepReady ? "#72e0a2" : "#fbbf24", fontSize: "11px", fontWeight: 800 }}>
            {steps.length > 0 ? `Step ${currentStepIndex + 1} of ${steps.length}` : "No visible choices"} · {currentStepReady ? "Ready" : "Needs selection"}
          </div>
        </div>

        <Focusable
          className="dmm-installer-choice-scroll"
          flow-children="down"
          navEntryPreferPosition={NavEntryPositionPreferences.FIRST}
          style={{
            alignContent: "start",
            boxSizing: "border-box",
            display: "grid",
            gridAutoRows: "max-content",
            gap: "10px",
            minHeight: 0,
            overflowX: "hidden",
            overflowY: "auto",
            paddingLeft: "10px",
            paddingBottom: "36px",
            paddingRight: "4px",
            scrollPaddingBlock: "14px",
            width: "100%"
          }}
        >
          <div style={{ background: "#0b1220", border: "1px solid #303741", borderRadius: "6px", color: "#a1a1aa", fontSize: "12px", lineHeight: 1.35, padding: "10px", overflowWrap: "anywhere" }}>
            {candidate.reason || "Choose installer options before DMM adds this mod to the profile."}
          </div>
          {selectedChoices > 0 && (
            <div style={{ border: "1px solid #365244", borderRadius: "6px", color: "#99f6e4", fontSize: "12px", lineHeight: 1.3, padding: "8px", overflowWrap: "anywhere" }}>
              {selectedChoices} choice{selectedChoices === 1 ? "" : "s"} preselected from DMM's saved/default installer state.
            </div>
          )}
          {!installer && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>Installer choices are not available for this action.</div>}
          {installer && currentStep ? (
            <section style={{ display: "grid", gap: "10px" }}>
              <div style={{ background: "#111827", border: "1px solid rgba(125, 211, 252, 0.28)", borderRadius: "6px", display: "grid", gap: "4px", padding: "10px" }}>
                <strong>{installer.name || "Installer Choices"}</strong>
                <small style={{ color: "#a1a1aa" }}>{currentStep.name}</small>
              </div>
              {currentStep.groups?.map((group) => (
                <div key={group.id} style={{ border: `1px solid ${fomodGroupValid(group, selections) ? "#303741" : "#fbbf24"}`, borderRadius: "6px", display: "grid", gap: "8px", padding: "10px" }}>
                  <div style={{ color: "#7dd3fc", fontWeight: 900 }}>{group.name}</div>
                  {group.description && <div style={{ color: "#d4d4d8", fontSize: "12px", lineHeight: 1.35, overflowWrap: "anywhere" }}>{group.description}</div>}
                  {fomodGroupIsText(group) ? (
                    <div style={{ display: "grid", gap: "6px" }}>
                      <TextField
                        label=""
                        value={groupTextValue(group)}
                        disabled={busy}
                        onChange={(event) => setGroupTextSelection(group, event.currentTarget.value, false)}
                        onBlur={(event) => setGroupTextSelection(group, event.currentTarget.value, true)}
                      />
                      {group.placeholder && <small style={{ color: "#a1a1aa" }}>Example: {group.placeholder}</small>}
                    </div>
                  ) : (
                    <Focusable flow-children="down" style={{ display: "grid", gap: "8px" }}>
                      {fomodGroupType(group) === "selectatmostone" && (
                        <Focusable
                          className="dmm-sidebar-row"
                          focusClassName="dmm-sidebar-row-focused"
                          onActivate={() => {
                            if (!busy) clearGroupSelection(group);
                          }}
                          onClick={() => {
                            if (!busy) clearGroupSelection(group);
                          }}
                          style={{ ...deckyCompositeRowStyle(false, (selections[group.id] ?? []).length === 0), opacity: busy ? 0.62 : 1, padding: "10px" }}
                        >
                          <div style={{ alignItems: "start", display: "grid", gap: "8px", gridTemplateColumns: "22px minmax(0, 1fr)" }}>
                            <input type="radio" readOnly tabIndex={-1} checked={(selections[group.id] ?? []).length === 0} style={{ marginTop: "2px", pointerEvents: "none" }} />
                            <span style={{ display: "grid", gap: "3px", minWidth: 0 }}>
                              <strong>None</strong>
                              <small style={{ color: "#a1a1aa" }}>Do not install an option from this group.</small>
                            </span>
                          </div>
                        </Focusable>
                      )}
                      {group.plugins?.map((plugin) => {
                        const selected = pluginSelected(group, plugin);
                        const locked = fomodPluginLocked(group, plugin);
                        const selectable = fomodPluginSelectable(plugin);
                        return (
                          <Focusable
                            key={plugin.id}
                            className="dmm-sidebar-row"
                            focusClassName="dmm-sidebar-row-focused"
                            onActivate={() => togglePluginSelection(group, plugin)}
                            onClick={() => togglePluginSelection(group, plugin)}
                            style={{ ...deckyCompositeRowStyle(false, selected), opacity: busy || !selectable ? 0.58 : locked ? 0.78 : 1, padding: "10px" }}
                          >
                            <div style={{ alignItems: "start", display: "grid", gap: "8px", gridTemplateColumns: "22px minmax(0, 1fr)" }}>
                              <input
                                type={fomodGroupInputType(group)}
                                readOnly
                                tabIndex={-1}
                                checked={selected}
                                style={{ marginTop: "2px", pointerEvents: "none" }}
                              />
                              <span style={{ display: "grid", gap: "3px", minWidth: 0 }}>
                                <strong>{plugin.name}</strong>
                                {fomodPluginType(plugin) && <small style={{ color: "#a1a1aa" }}>{fomodPluginType(plugin)}</small>}
                                {plugin.description && <em style={{ color: "#d4d4d8", fontStyle: "normal", lineHeight: 1.3, overflowWrap: "anywhere" }}>{plugin.description}</em>}
                              </span>
                            </div>
                          </Focusable>
                        );
                      })}
                    </Focusable>
                  )}
                  {!fomodGroupValid(group, selections) && <div style={{ color: "#fbbf24", fontSize: "11px" }}>This group needs a valid selection before continuing.</div>}
                </div>
              ))}
            </section>
          ) : installer ? (
            <div style={{ color: "#a1a1aa" }}>This installer has no visible choices. Apply it to add the mod to the profile.</div>
          ) : null}
          {message && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{message}</div>}

          <Focusable className="dmm-action-grid" flow-children="right" navEntryPreferPosition={NavEntryPositionPreferences.FIRST} style={deckyActionGridStyle(currentStepIndex > 0 ? 3 : 2)}>
            {currentStepIndex > 0 && (
              <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => setStepIndex(Math.max(0, currentStepIndex - 1))} onClick={() => setStepIndex(Math.max(0, currentStepIndex - 1))} style={deckyCompactActionStyle("neutral", busy)}>
                Back
              </Focusable>
            )}
            <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={props.closeModal} onClick={props.closeModal} style={deckyCompactActionStyle("neutral")}>
              Later
            </Focusable>
            <Focusable
              className="dmm-focus-card"
              focusClassName="dmm-focus-card-focused"
              onActivate={continueOrApply}
              onClick={continueOrApply}
              style={{ ...deckyCompactActionStyle("neutral", primaryActionDisabled), opacity: primaryActionDisabled ? 0.56 : 1 }}
            >
              {primaryActionLabel}
            </Focusable>
          </Focusable>
        </Focusable>
      </Focusable>
    </ModalRoot>
  );
}

async function maybeShowInstallerChoiceModal(job: Job) {
  if (job.type !== "installer-choice" || job.status !== "waiting") return;
  let status: BackendStatus | null = null;
  try {
    status = await call<[], BackendStatus>("status");
  } catch (err) {
    await logFrontendEvent("installer choice modal status check failed", { job_id: job.id, error: err instanceof Error ? err.message : String(err) });
    return;
  }
  if (!status?.backend?.install.auto_show_fomod_installers) {
    await logFrontendEvent("installer choice modal skipped because auto display is off", { job_id: job.id });
    return;
  }
  const appID = String(job.payload?.app_id ?? "").trim();
  const candidateID = Number.parseInt(String(job.payload?.candidate_id ?? ""), 10);
  if (!appID || !Number.isFinite(candidateID)) return;
  const key = String(candidateID);
  if (applyingInstallerChoiceCandidates.has(key)) {
    await logFrontendEvent("installer choice modal skipped while apply is in progress", { app_id: appID, candidate_id: candidateID, job_id: job.id });
    return;
  }
  if (shownInstallerChoiceModals.has(key)) return;
  if (dismissedInstallerChoiceModals.has(key)) {
    await logFrontendEvent("installer choice modal skipped after user dismissed auto display", { app_id: appID, candidate_id: candidateID });
    return;
  }
  try {
    const result = await call<[string], { ok: boolean; error?: string; candidates: InstallCandidate[] }>("game_install_candidates", appID);
    if (!result.ok) {
      await logFrontendEvent("installer choice candidates load failed", { app_id: appID, candidate_id: candidateID, error: result.error || "" });
      showLaunchToast("DMM installer choices needed", "Open Decky Mod Manager or the phone UI to finish this installer.", false);
      return;
    }
    const candidate = result.candidates.find((item) => item.id === candidateID);
    if (!candidate || !installerForCandidate(candidate)) {
      await logFrontendEvent("installer choice candidate missing installer json", { app_id: appID, candidate_id: candidateID });
      showLaunchToast("DMM installer choices needed", "Open Decky Mod Manager or the phone UI to finish this installer.", false);
      return;
    }
    await openInstallerChoiceModalForCandidate(appID, candidate, "event");
  } catch (err) {
    await logFrontendEvent("installer choice modal open failed", { app_id: appID, candidate_id: candidateID, error: err instanceof Error ? err.message : String(err) });
    showLaunchToast("DMM installer choices needed", "Open Decky Mod Manager or the phone UI to finish this installer.", false);
  }
}

async function openInstallerChoiceModalForCandidate(appID: string, candidate: InstallCandidate, source: string, onApplied?: () => void, profileID = 0) {
  const key = String(candidate.id);
  if (shownInstallerChoiceModals.has(key)) return;
  if (source !== "event") dismissedInstallerChoiceModals.delete(key);
  if (candidate.status === "blocked") {
    await logFrontendEvent("installer choice modal skipped for blocked candidate", { app_id: appID, candidate_id: candidate.id, source });
    showLaunchToast("DMM installer needs review", candidate.reason || "Open the phone UI to review this installer item.", true);
    return;
  }
  if (!installerForCandidate(candidate)) {
    await logFrontendEvent("installer choice modal skipped for candidate without installer", { app_id: appID, candidate_id: candidate.id, source });
    showLaunchToast("DMM installer choices unavailable", "Open the phone UI to review this installer item.", false);
    return;
  }
  shownInstallerChoiceModals.add(key);
  try {
    let modal: { Close: () => void } | null = null;
    let applied = false;
    const closeModal = () => {
      shownInstallerChoiceModals.delete(key);
      if (!applied && source === "event") dismissedInstallerChoiceModals.add(key);
      modal?.Close();
    };
    modal = showModal(
      <InstallerChoiceModal
        appID={appID}
        candidate={candidate}
        profileID={profileID || candidate.target_profile_id || 0}
        closeModal={closeModal}
        onApplied={() => {
          applied = true;
          applyingInstallerChoiceCandidates.add(key);
          shownInstallerChoiceModals.delete(key);
          void seedJobNotifications({ seed: true });
          void syncLaunchActions();
          onApplied?.();
        }}
      />,
      window,
      { strTitle: "DMM Installer Choices", bNeverPopOut: true, popupWidth: 520, popupHeight: 720 }
    );
    await logFrontendEvent("installer choice modal opened", { app_id: appID, candidate_id: candidate.id, source });
  } catch (err) {
    shownInstallerChoiceModals.delete(key);
    await logFrontendEvent("installer choice modal open failed", { app_id: appID, candidate_id: candidate.id, source, error: err instanceof Error ? err.message : String(err) });
    showLaunchToast("DMM installer choices needed", "Open Decky Mod Manager or the phone UI to finish this installer.", false);
  }
}

function NexusBrowserModal(props: { appID: string; gameName: string; gameDomain: string; profileID?: number; closeModal: () => void }) {
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<NexusSearchSort>("updated");
  const [timeWindow, setTimeWindow] = useState<NexusTimeWindow>("all");
  const [vortexOnly, setVortexOnly] = useState(true);
  const [mods, setMods] = useState<NexusModResult[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [offset, setOffset] = useState(0);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const pageSize = 20;

  async function searchMods(nextSort = sort, nextWindow = timeWindow, nextOffset = 0, append = false, nextVortexOnly = vortexOnly) {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const result = await call<[string, string, string, string, number, number, boolean, string], { ok: boolean; error?: string; mods: NexusModResult[]; total_count: number }>(
        "nexus_mods",
        props.appID,
        query,
        nextSort,
        nextWindow,
        pageSize,
        nextOffset,
        nextVortexOnly,
        props.gameDomain
      );
      if (!result.ok) {
        setError(result.error || "Unable to search Nexus Mods.");
        if (!append) {
          setMods([]);
          setTotalCount(0);
        }
        return;
      }
      const nextMods = result.mods ?? [];
      setMods((current) => append ? [...current, ...nextMods] : nextMods);
      setTotalCount(result.total_count ?? result.mods?.length ?? 0);
      setOffset(nextOffset + nextMods.length);
      if (nextMods.length === 0 && !append) setMessage(nextVortexOnly ? "No Vortex-compatible mods matched this search." : "No Nexus mods matched this search.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function openModPage(mod: NexusModResult) {
    setError("");
    setMessage("");
    try {
      await logFrontendEvent("nexus browser mod page requested", {
        app_id: props.appID,
        profile_id: props.profileID ?? 0,
        game_domain: props.gameDomain,
        mod_id: mod.mod_id
      });
      const opened = await openDMMBrowserViewCapture(mod.url, {
        appID: props.appID,
        profileID: props.profileID ?? 0,
        source: "nexus-mod-page",
        title: `DMM Nexus - ${mod.name}`
      });
      await logFrontendEvent("nexus browser mod page opened", {
        app_id: props.appID,
        profile_id: props.profileID ?? 0,
        game_domain: props.gameDomain,
        mod_id: mod.mod_id,
        opened
      });
      if (!opened) {
        setError("DMM could not open the controlled Nexus browser. Check Debug Live Logs.");
        return;
      }
      props.closeModal();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      await logFrontendEvent("nexus browser mod page threw", {
        app_id: props.appID,
        profile_id: props.profileID ?? 0,
        game_domain: props.gameDomain,
        mod_id: mod.mod_id,
        error: err instanceof Error ? err.message : String(err)
      });
    }
  }

  function cycleSort() {
    const next = nextNexusSort(sort);
    setSort(next);
    setOffset(0);
    void searchMods(next, timeWindow, 0, false);
  }

  function cycleTimeWindow() {
    const next = nextNexusTimeWindow(timeWindow);
    setTimeWindow(next);
    setOffset(0);
    void searchMods(sort, next, 0, false);
  }

  function toggleVortexOnly() {
    const next = !vortexOnly;
    setVortexOnly(next);
    setOffset(0);
    void searchMods(sort, timeWindow, 0, false, next);
  }

  function submitSearch() {
    setOffset(0);
    void searchMods(sort, timeWindow, 0, false);
  }

  function loadMore() {
    if (busy || offset >= totalCount) return;
    void searchMods(sort, timeWindow, offset, true);
  }

  function consumeNexusBrowserGamepadEvent(event?: GamepadEvent | { preventDefault?: () => void; stopPropagation?: () => void; stopImmediatePropagation?: () => void }) {
    event?.preventDefault?.();
    event?.stopPropagation?.();
    event?.stopImmediatePropagation?.();
  }

  function searchFromGamepad(source: string, event?: GamepadEvent) {
    consumeNexusBrowserGamepadEvent(event);
    void logFrontendEvent("nexus browser modal gamepad search requested", {
      app_id: props.appID,
      game_domain: props.gameDomain,
      source,
      button: event?.detail?.button ?? 0,
      repeat: Boolean(event?.detail?.is_repeat)
    });
    if (event?.detail?.is_repeat) return;
    submitSearch();
  }

  function handleNexusBrowserButtonDown(event: GamepadEvent) {
    if (event.detail.button === GamepadButton.TRIGGER_RIGHT) {
      searchFromGamepad("trigger-right-down", event);
    }
  }

  function handleNexusBrowserButtonUp(event: GamepadEvent) {
    if (event.detail.button === GamepadButton.TRIGGER_RIGHT) {
      consumeNexusBrowserGamepadEvent(event);
      void logFrontendEvent("nexus browser modal trigger release consumed", {
        app_id: props.appID,
        game_domain: props.gameDomain,
        button: event.detail.button
      });
    }
  }

  function closeNexusBrowserFromGamepad(event?: GamepadEvent | { preventDefault?: () => void; stopPropagation?: () => void; stopImmediatePropagation?: () => void }) {
    consumeNexusBrowserGamepadEvent(event);
    void logFrontendEvent("nexus browser modal gamepad close requested", {
      app_id: props.appID,
      game_domain: props.gameDomain
    });
    props.closeModal();
  }

  function absorbNexusBrowserModalCancel() {
    void logFrontendEvent("nexus browser modal cancel absorbed", {
      app_id: props.appID,
      game_domain: props.gameDomain
    });
  }

  useEffect(() => {
    void searchMods("updated", "all", 0, false);
  }, []);

  return (
    <ModalRoot onCancel={absorbNexusBrowserModalCancel} bCancelDisabled bAllowFullSize bHideCloseIcon>
      <style>{deckyRuntimeStyles}</style>
	      <Focusable
	        actionDescriptionMap={{ [GamepadButton.CANCEL]: "Close", [GamepadButton.TRIGGER_RIGHT]: "Search" }}
	        flow-children="down"
	        onButtonDown={handleNexusBrowserButtonDown}
	        onButtonUp={handleNexusBrowserButtonUp}
	        onCancelButton={closeNexusBrowserFromGamepad}
	        navEntryPreferPosition={NavEntryPositionPreferences.PREFERRED_CHILD}
	        style={{
          boxSizing: "border-box",
          color: "#f8fafc",
          display: "grid",
          gap: "10px",
          gridTemplateRows: "auto minmax(0, 1fr)",
          height: "min(760px, calc(100vh - 96px))",
          maxHeight: "calc(100vh - 96px)",
          minHeight: "420px",
          overflow: "hidden",
          padding: "4px",
          width: "100%"
        }}
      >
        <div style={{ alignItems: "start", display: "grid", gap: "10px", gridTemplateColumns: "minmax(0, 1fr) 88px", width: "100%" }}>
          <div style={{ display: "grid", gap: "4px", minWidth: 0 }}>
            <div style={{ color: "#f8fafc", fontSize: "16px", fontWeight: 900, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{props.gameName}</div>
            <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2 }}>
              Showing {mods.length} of {compactNumber(totalCount)} {vortexOnly ? "Vortex-compatible" : "Nexus"} result{totalCount === 1 ? "" : "s"} from {props.gameDomain}.
            </div>
          </div>
          <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={props.closeModal} onClick={props.closeModal} style={deckyCompactActionStyle("neutral")}>
            Close
          </Focusable>
        </div>
        <Focusable
          flow-children="down"
          navEntryPreferPosition={NavEntryPositionPreferences.FIRST}
          style={{
            alignContent: "start",
            boxSizing: "border-box",
            display: "grid",
            gridAutoRows: "max-content",
            gap: "10px",
            minHeight: 0,
            overflowX: "hidden",
            overflowY: "auto",
            paddingBottom: "36px",
            paddingRight: "6px",
            scrollPaddingBlock: "10px",
            width: "100%"
          }}
        >
          <div style={{ background: "#0b1220", border: "1px solid #303741", borderRadius: "6px", boxSizing: "border-box", display: "grid", gap: "8px", padding: "8px", width: "100%" }}>
            <TextField
              aria-label="Search Nexus Mods"
              label="Search Nexus Mods"
              value={query}
              bShowClearAction
              focusOnMount
              onChange={(event) => setQuery(event.currentTarget.value)}
              onKeyDown={(event) => {
                if (event.key === "Enter") submitSearch();
              }}
            />
            <Focusable flow-children="right" style={{ ...deckyActionGridStyle(3), gap: "8px" }}>
              <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={cycleSort} onClick={cycleSort} style={deckyCompactActionStyle("neutral")}>
                Sort: {nexusSortLabel(sort)}
              </Focusable>
              <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={cycleTimeWindow} onClick={cycleTimeWindow} style={deckyCompactActionStyle("neutral")}>
                Updated: {nexusTimeWindowLabel(timeWindow)}
              </Focusable>
              <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={toggleVortexOnly} onClick={toggleVortexOnly} style={deckyCompactActionStyle("neutral")}>
                {vortexOnly ? "Vortex Only" : "All Mods"}
              </Focusable>
            </Focusable>
            <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={submitSearch} onClick={submitSearch} style={deckyCompactActionStyle("neutral", busy)}>
              {busy ? "Searching" : "Search"}
            </Focusable>
          </div>
          {(error || message) && (
            <div style={{ display: "grid", gap: "6px", minWidth: 0 }}>
              {error && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{error}</div>}
              {message && <div style={{ color: "#72e0a2", overflowWrap: "anywhere" }}>{message}</div>}
            </div>
          )}
          {mods.length === 0 && !busy && !error && (
            <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>{vortexOnly ? "No Vortex-compatible mods matched this search." : "No Nexus mods matched this search."}</div>
          )}
	          {mods.map((mod) => (
	            <Focusable
	              key={mod.mod_id}
	              className="dmm-sidebar-row"
	              focusClassName="dmm-sidebar-row-focused"
	              onActivate={() => void openModPage(mod)}
	              onClick={() => void openModPage(mod)}
	              style={{
	                ...deckyCompositeRowStyle(false),
	                alignSelf: "start",
	                background: "#111827",
	                flexShrink: 0,
	                minHeight: "132px",
	                padding: "10px"
	              }}
	            >
	              <div style={{ alignItems: "start", display: "grid", gap: "10px", gridTemplateColumns: "112px minmax(0, 1fr)", width: "100%" }}>
	                <div style={{ background: "#030712", border: "1px solid #303741", borderRadius: "6px", height: "64px", overflow: "hidden", width: "112px" }}>
	                  {mod.thumbnail_url ? (
	                    <img src={mod.thumbnail_url} style={{ height: "100%", objectFit: "cover", width: "100%" }} />
	                  ) : (
	                    <div style={{ alignItems: "center", color: "#71717a", display: "flex", fontSize: "11px", height: "100%", justifyContent: "center", textAlign: "center" }}>No image</div>
	                  )}
	                </div>
	                <div style={{ display: "grid", gap: "6px", minWidth: 0 }}>
	                  <div style={{ alignItems: "flex-start", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
	                    <div style={{ ...deckyTwoLineTextStyle, flex: "1 1 120px", fontWeight: 900 }}>{mod.name}</div>
	                    <span style={deckySourcePillStyle("nexus")}>Nexus</span>
	                  </div>
	                  <div style={{ color: "#d4d4d8", fontSize: "11px", lineHeight: 1.25, maxHeight: "3.75em", overflow: "hidden", overflowWrap: "anywhere" }}>{mod.summary}</div>
	                  <div style={{ color: "#a1a1aa", display: "flex", flexWrap: "wrap", fontSize: "11px", gap: "10px" }}>
	                    <span>v{mod.version || "unknown"}</span>
	                    <span>{compactNumber(mod.downloads)} downloads</span>
	                    <span>{compactNumber(mod.endorsements)} endorsements</span>
	                    <span>{mod.updated_at ? `Updated ${new Date(mod.updated_at).toLocaleDateString()}` : "Updated unknown"}</span>
	                  </div>
	                  <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 900 }}>A View Mod Page</div>
	                </div>
	              </div>
	            </Focusable>
	          ))}
          {mods.length > 0 && offset < totalCount && (
            <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={loadMore} onClick={loadMore} style={deckyCompactActionStyle("neutral", busy)}>
              {busy ? "Loading More" : "Load More"}
            </Focusable>
          )}
        </Focusable>
      </Focusable>
    </ModalRoot>
  );
}

async function handleDeckyDomainEvent(event: DomainEvent) {
  if (event.id > eventMonitorLastID) eventMonitorLastID = event.id;
  if (event.type === "decky.browser.open") {
    await handleDeckyBrowserOpenEvent(event);
  }
  if (event.type === "jobs.snapshot" && Array.isArray(event.payload)) {
    for (const item of event.payload) {
      if (!isJob(item)) continue;
      await maybeShowJobToast(item, { seed: true, source: "event-snapshot" });
      await maybeShowInstallerChoiceModal(item);
    }
  }
  if (event.type === "job.updated" && isJob(event.payload)) {
    await maybeShowJobToast(event.payload, { source: "event" });
    if (event.payload.type === "installer-choice" && event.payload.status !== "waiting" && event.payload.payload?.candidate_id) {
      const candidateKey = String(event.payload.payload.candidate_id);
      applyingInstallerChoiceCandidates.delete(candidateKey);
      shownInstallerChoiceModals.delete(candidateKey);
      dismissedInstallerChoiceModals.delete(candidateKey);
    }
    await maybeShowInstallerChoiceModal(event.payload);
  }
  if (eventShouldSyncLaunchActions(event)) {
    await syncLaunchActions();
  }
  if (eventShouldSyncWorkshopActions(event)) {
    await syncWorkshopActions();
  }
  window.dispatchEvent(new CustomEvent(DMM_EVENT_NAME, { detail: event }));
}

function connectEventMonitor() {
  if (eventMonitorSocket && (eventMonitorSocket.readyState === WebSocket.CONNECTING || eventMonitorSocket.readyState === WebSocket.OPEN)) return;
  if (eventMonitorReconnectTimer !== null) {
    window.clearTimeout(eventMonitorReconnectTimer);
    eventMonitorReconnectTimer = null;
  }
  const params = new URLSearchParams();
  if (eventMonitorLastID > 0) params.set("after", String(eventMonitorLastID));
  if (backendAuthToken) params.set("token", backendAuthToken);
  const query = params.toString();
  try {
    const socket = new WebSocket(`${DMM_BACKEND_WS_URL}${query ? `?${query}` : ""}`);
    eventMonitorSocket = socket;
    socket.onopen = () => {
      eventMonitorReconnectDelay = 1000;
      logFrontendEvent("event monitor connected");
    };
    socket.onmessage = (message) => {
      if (typeof message.data !== "string") return;
      try {
        void handleDeckyDomainEvent(JSON.parse(message.data) as DomainEvent);
      } catch (err) {
        void logFrontendEvent("event monitor message failed", { error: err instanceof Error ? err.message : String(err) });
      }
    };
    socket.onerror = () => {
      void logFrontendEvent("event monitor socket error");
      socket.close();
    };
    socket.onclose = () => {
      if (eventMonitorSocket !== socket) return;
      eventMonitorSocket = null;
      scheduleEventMonitorReconnect();
    };
  } catch (err) {
    void logFrontendEvent("event monitor connect failed", { error: err instanceof Error ? err.message : String(err) });
    scheduleEventMonitorReconnect();
  }
}

function scheduleEventMonitorReconnect() {
  if (eventMonitorReconnectTimer !== null || !backgroundMonitorsStarted) return;
  const delay = eventMonitorReconnectDelay;
  eventMonitorReconnectDelay = Math.min(eventMonitorReconnectDelay * 2, 15000);
  eventMonitorReconnectTimer = window.setTimeout(() => {
    eventMonitorReconnectTimer = null;
    connectEventMonitor();
  }, delay);
}

function closeEventMonitor() {
  if (eventMonitorReconnectTimer !== null) {
    window.clearTimeout(eventMonitorReconnectTimer);
    eventMonitorReconnectTimer = null;
  }
  if (eventMonitorSocket) {
    const socket = eventMonitorSocket;
    eventMonitorSocket = null;
    socket.close();
  }
}

function applyBackendAuthFromStatus(nextStatus?: BackendStatus | null) {
  const token = nextStatus?.auth?.token?.trim() ?? "";
  if (token === backendAuthToken) return;
  backendAuthToken = token;
  if (eventMonitorSocket) {
    const socket = eventMonitorSocket;
    eventMonitorSocket = null;
    socket.close();
  }
}

function startBackgroundMonitors() {
  if (backgroundMonitorsStarted) return;
  backgroundMonitorsStarted = true;
  logFrontendEvent("background monitors started");
  logSteamClientCapabilities();
  void (async () => {
    try {
      applyBackendAuthFromStatus(await call<[], BackendStatus>("status"));
    } catch (err) {
      await logFrontendEvent("background status refresh failed", { error: err instanceof Error ? err.message : String(err) });
    }
    await seedJobNotifications({ seed: true });
    await syncLaunchActions();
    await syncWorkshopActions();
    await seedWorkshopStateFromGames();
    connectEventMonitor();
  })();
}

function stopBackgroundMonitors() {
  if (!backgroundMonitorsStarted) return;
  backgroundMonitorsStarted = false;
  closeEventMonitor();
  if (steamBrowserNXMProbeRegistration) {
    unregisterSteamCallback(steamBrowserNXMProbeRegistration);
    steamBrowserNXMProbeRegistration = null;
  }
  destroyActiveDMMNativeBrowser("background-stop");
  logFrontendEvent("background monitors stopped");
}

function openDeckyModManagerRoute() {
  navigateDMMRoute(DMM_DECKY_ROUTE, "open-decky-mod-manager");
}

function DMMNativeBrowserTab(props: { browser: DMMNativeBrowser | null; closeBrowser: () => void; error: string; ready: boolean }) {
  if (props.error) {
    return (
      <div style={{ color: "#f87171", padding: "18px", overflowWrap: "anywhere" }}>
        {props.error}
      </div>
    );
  }

  if (!props.browser || !DMMBrowserContainer) {
    return (
      <div style={{ color: "#f87171", padding: "18px", overflowWrap: "anywhere" }}>
        DMM native browser is unavailable.
      </div>
    );
  }

  if (!props.ready) {
    return (
      <div style={{ alignItems: "center", display: "flex", height: "100%", justifyContent: "center", width: "100%" }}>
        <SteamSpinner />
      </div>
    );
  }

  return (
    <Focusable
      className="dmm-native-browser-focus"
      noFocusRing
      onCancelButton={props.closeBrowser}
      onCancelActionDescription="Back"
      onGamepadFocus={async (event: GamepadEvent) => {
        await sleep(1);
        const focusEvent = event as unknown as DMMFocusedNodeEvent;
        focusEvent.detail.focusedNode?.BChildTakeFocus?.();
      }}
    >
      <DMMBrowserContainer
        browser={props.browser}
        className={mainBrowserClasses?.ExternalBrowserContainer}
        visible
        hideForModals
        external
        displayURLBar={false}
        autoFocus={false}
      />
    </Focusable>
  );
}

function DMMNativeBrowserRoute() {
  const [browser, setBrowser] = useState<DMMNativeBrowser | null>(activeDMMNativeBrowser);
  const [error, setError] = useState("");
  const [activeTab, setActiveTab] = useState(DMM_NATIVE_BROWSER_TAB_ID);
  const [mountReady, setMountReady] = useState(false);
  const request = activeDMMBrowserRequest;

  useEffect(() => {
    const windowRouter = getDMMWindowRouter();
    const currentBrowser = activeDMMNativeBrowser;
    void logFrontendEvent("dmm native browser route mounted", {
      source: request?.source || "",
      path: window.location.pathname,
      has_active_browser: Boolean(currentBrowser),
      has_browser_container: Boolean(DMMBrowserContainer),
      external_browser_class: mainBrowserClasses?.ExternalBrowserContainer || ""
    });
    if (!currentBrowser) {
      const message = "DMM native browser route mounted without an active browser.";
      setError(message);
      void logFrontendEvent("dmm native browser route missing browser", { source: request?.source || "" });
      return;
    }
    let mountTimer: number | undefined;
    try {
      windowRouter?.HeaderStore?.SetCurrentBrowserAndBackstack?.(currentBrowser, true);
      currentBrowser.m_browserView?.SetVisible?.(true);
      currentBrowser.m_browserView?.SetFocus?.(true);
      setBrowser(currentBrowser);
      mountTimer = window.setTimeout(() => {
        setMountReady(true);
        void logFrontendEvent("dmm native browser route browser attached", {
          source: request?.source || "",
          url: request?.initialURL || "",
          has_browser_container: Boolean(DMMBrowserContainer),
          has_header_store: Boolean(windowRouter?.HeaderStore)
        });
      }, 800);
      void logFrontendEvent("dmm native browser route prepared", {
        source: request?.source || "",
        url: request?.initialURL || "",
        has_browser_container: Boolean(DMMBrowserContainer),
        has_header_store: Boolean(windowRouter?.HeaderStore)
      });
    } catch (err) {
      const message = errorLogValue(err);
      setError(message);
      void logFrontendEvent("dmm native browser route mount failed", { source: request?.source || "", error: message });
    }
    return () => {
      try {
        if (typeof mountTimer === "number") window.clearTimeout(mountTimer);
        const current = windowRouter?.HeaderStore?.GetCurrentBrowserAndBackstack?.();
        if (current?.browser?.name === DMM_NATIVE_BROWSER_NAME) {
          windowRouter?.HeaderStore?.SetCurrentBrowserAndBackstack?.(null, false);
        }
        destroyActiveDMMNativeBrowser("route-unmount");
      } catch (err) {
        void logFrontendEvent("dmm native browser route cleanup failed", { source: request?.source || "", error: errorLogValue(err) });
      }
    };
  }, []);

  const closeBrowser = () => {
    rememberDMMDeckyReturnContext(activeDMMBrowserRequest);
    destroyActiveDMMNativeBrowser("route-close");
    if (activeDMMBrowserRequest?.appID) {
      activeDMMBrowserRequest = null;
      navigateDMMRoute(DMM_DECKY_ROUTE, "browser-close");
      return;
    }
    activeDMMBrowserRequest = null;
    Navigation.NavigateBack();
  };

  const tabs = useMemo<DeckyUITab[]>(() => [{
    id: DMM_NATIVE_BROWSER_TAB_ID,
    title: "Browser",
    content: <DMMNativeBrowserTab browser={browser} closeBrowser={closeBrowser} error={error} ready={mountReady} />,
    footer: {
      onCancelButton: closeBrowser,
      onCancelActionDescription: "Back"
    }
  }], [browser, error, mountReady]);

  return (
    <div className="dmm-native-browser-tabs" style={{ background: "#050914", height: "calc(100% - 40px)", marginTop: "40px", overflow: "hidden", width: "100%" }}>
      <style>{`
        .dmm-native-browser-tabs {
          background: #050914;
        }
        .dmm-native-browser-tabs .${gamepadTabbedPageClasses?.Floating || ""} .${gamepadTabbedPageClasses?.TabContentsScroll || ""} {
          clip: initial;
        }
        .dmm-native-browser-tabs .${gamepadTabbedPageClasses?.TabHeaderRowWrapper || ""} {
          background: #060709;
        }
        .dmm-native-browser-tabs .${gamepadTabbedPageClasses?.FixCenterAlignScroll || ""} {
          padding: 5px 0;
        }
        .dmm-native-browser-tabs .${gamepadTabbedPageClasses?.TabContentsScroll || ""} {
          bottom: var(--gamepadui-current-footer-height);
          padding: 0;
        }
        .dmm-native-browser-tabs .dmm-native-browser-focus {
          height: 100%;
          min-height: 100%;
          overflow: hidden;
          position: relative;
          width: 100%;
        }
        .dmm-native-browser-tabs .${mainBrowserClasses?.ExternalBrowserContainer || ""} {
          background: #050914;
          height: 100%;
          top: 44px;
          width: 100%;
        }
        .dmm-native-browser-tabs .${steamSpinnerClasses?.SpinnerLoaderContainer || ""} {
          background: transparent;
        }
      `}</style>
      <Tabs
        activeTab={activeTab}
        autoFocusContents
        onShowTab={(nextTab: string) => setActiveTab(nextTab)}
        tabs={tabs}
      />
    </div>
  );
}

type FreshDeckyTab = "actions" | "games" | "settings";

const freshDeckyTabs: Array<{ id: FreshDeckyTab; label: string }> = [
  { id: "games", label: "Games" },
  { id: "actions", label: "Actions" },
  { id: "settings", label: "Settings" }
];

const freshDeckyShellStyle: CSSProperties = {
  background: "#090f1a",
  boxSizing: "border-box",
  color: "#f8fafc",
  display: "grid",
  gridTemplateRows: "40px minmax(0, 1fr)",
  height: "100%",
  minHeight: 0,
  minWidth: 0,
  overflow: "hidden",
  padding: "40px 24px 0",
  width: "100%"
};

const freshDeckyTabBarStyle: CSSProperties = {
  display: "grid",
  gap: "6px",
  gridTemplateColumns: "repeat(3, minmax(0, 1fr))",
  minWidth: 0,
  width: "100%"
};

const freshDeckyBodyStyle: CSSProperties = {
  alignContent: "start",
  boxSizing: "border-box",
  display: "grid",
  gap: "10px",
  minHeight: 0,
  overflowX: "hidden",
  overflowY: "auto",
  overscrollBehavior: "contain",
  padding: "16px 0 176px",
  scrollPaddingBottom: "176px",
  scrollPaddingTop: "28px",
  width: "100%"
};

const freshSectionStyle: CSSProperties = {
  background: "rgba(15, 23, 42, 0.74)",
  border: "1px solid rgba(71, 85, 105, 0.82)",
  borderRadius: "7px",
  boxSizing: "border-box",
  display: "grid",
  gap: "8px",
  minWidth: 0,
  padding: "10px",
  width: "100%"
};

const freshSettingsPrimaryCardStyle: CSSProperties = {
  ...freshSectionStyle,
  alignContent: "start",
  minHeight: "120px"
};

const freshSettingsCardStyle: CSSProperties = {
  ...freshSectionStyle,
  alignContent: "start",
  minHeight: "118px"
};

const freshSettingsToggleCardStyle: CSSProperties = {
  ...freshSectionStyle,
  alignContent: "center",
  minHeight: "76px"
};

const freshActionRowStyle: CSSProperties = {
  display: "grid",
  gap: "8px",
  gridTemplateColumns: "minmax(0, 1fr) minmax(0, 1fr)",
  minWidth: 0,
  width: "100%"
};

function freshTabStyle(active: boolean): CSSProperties {
  return {
    ...deckyFocusableCardBase,
    alignItems: "center",
    background: active ? "#0f766e" : "#192231",
    border: `1px solid ${active ? "#5eead4" : "#334155"}`,
    color: "#f8fafc",
    display: "flex",
    fontSize: "11px",
    fontWeight: 900,
    height: "34px",
    justifyContent: "center",
    lineHeight: 1,
    padding: "0 4px",
    textTransform: "uppercase",
    whiteSpace: "nowrap"
  };
}

function freshButtonStyle(kind: "primary" | "neutral" | "danger" = "neutral", disabled = false): CSSProperties {
  const palettes = {
    primary: { background: "#0f766e", border: "#5eead4", color: "#f8fafc" },
    neutral: { background: "#1f2937", border: "#475569", color: "#f8fafc" },
    danger: { background: "#3f1d1d", border: "#7f1d1d", color: "#fecaca" }
  };
  const palette = palettes[kind];
  return {
    ...deckyFocusableCardBase,
    alignItems: "center",
    background: palette.background,
    border: `1px solid ${palette.border}`,
    color: palette.color,
    display: "flex",
    fontSize: "12px",
    fontWeight: 900,
    justifyContent: "center",
    lineHeight: 1.12,
    minHeight: "38px",
    opacity: disabled ? 0.52 : 1,
    padding: "8px 7px",
    textAlign: "center",
    whiteSpace: "normal"
  };
}

function freshCardStyle(active = false): CSSProperties {
  return {
    ...deckyFocusableCardBase,
    background: active ? "rgba(15, 118, 110, 0.22)" : "rgba(17, 24, 39, 0.78)",
    border: `1px solid ${active ? "#0f766e" : "#334155"}`,
    display: "grid",
    gap: "6px",
    padding: "10px"
  };
}

function freshGameCardStyle(active = false): CSSProperties {
  return {
    ...freshCardStyle(active),
    alignItems: "center",
    boxSizing: "border-box",
    minHeight: "74px",
    overflow: "hidden",
    width: "100%"
  };
}

const freshGameCardContentStyle: CSSProperties = {
  alignItems: "center",
  boxSizing: "border-box",
  display: "grid",
  gap: "10px",
  gridTemplateColumns: "78px minmax(0, 1fr)",
  minHeight: "52px",
  minWidth: 0,
  width: "100%"
};

const freshGameImageStyle: CSSProperties = {
  aspectRatio: "16 / 9",
  borderRadius: "5px",
  display: "block",
  height: "44px",
  objectFit: "cover",
  width: "78px"
};

function freshModCardStyle(active = false): CSSProperties {
  return {
    ...freshCardStyle(active),
    boxSizing: "border-box",
    minHeight: "70px",
    overflow: "hidden",
    width: "100%"
  };
}

function FreshActionButton(props: { children: ReactNode; disabled?: boolean; kind?: "primary" | "neutral" | "danger"; onActivate: () => void }) {
  return (
    <Focusable
      className="dmm-focus-card"
      focusClassName="dmm-focus-card-focused"
      onActivate={() => {
        if (!props.disabled) props.onActivate();
      }}
      onClick={() => {
        if (!props.disabled) props.onActivate();
      }}
      style={freshButtonStyle(props.kind ?? "neutral", props.disabled)}
    >
      {props.children}
    </Focusable>
  );
}

function pairingURLFromStatus(status: BackendStatus | null) {
  return status?.url?.trim() ?? "";
}

function pairingDisplayAddress(status: BackendStatus | null) {
  return status?.plain_url?.trim() || status?.url?.split("#")[0]?.trim() || "Unavailable";
}

function pairingQRCodeDataURL(value: string) {
  if (!value) return "";
  const qr = qrcode(0, "M");
  qr.addData(value);
  qr.make();
  return qr.createDataURL(6, 2);
}

function PairPhoneModal(props: { status: BackendStatus | null; closeModal: () => void }) {
  const pairingURL = pairingURLFromStatus(props.status);
  const displayAddress = pairingDisplayAddress(props.status);
  const [showManualURL, setShowManualURL] = useState(false);
  const qrDataURL = useMemo(() => {
    try {
      return pairingQRCodeDataURL(pairingURL);
    } catch (err) {
      void logFrontendEvent("decky pairing qr generation failed", { error: errorLogValue(err) });
      return "";
    }
  }, [pairingURL]);

  return (
    <ModalRoot onCancel={props.closeModal} bAllowFullSize bHideCloseIcon>
      <style>{deckyRuntimeStyles}</style>
      <Focusable flow-children="down" style={{ color: "#f8fafc", display: "grid", gap: "12px", minWidth: 0, padding: "4px", width: "100%" }}>
        <div style={{ fontSize: "16px", fontWeight: 900 }}>Pair Phone</div>
        <div style={{ color: "#d4d4d8", fontSize: "12px", lineHeight: 1.35 }}>
          Scan this QR code from your phone or tablet while connected to the same network.
        </div>
        <div style={{ alignItems: "center", background: "#f8fafc", borderRadius: "8px", display: "flex", justifyContent: "center", minHeight: "236px", padding: "12px", width: "100%" }}>
          {qrDataURL ? (
            <img alt="DMM phone pairing QR code" src={qrDataURL} style={{ display: "block", height: "212px", imageRendering: "pixelated", width: "212px" }} />
          ) : (
            <div style={{ color: "#991b1b", fontWeight: 900 }}>QR unavailable</div>
          )}
        </div>
        <div style={{ ...freshSectionStyle, background: "#0b1220" }}>
          <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 900, textTransform: "uppercase" }}>Server Address</div>
          <div style={{ color: "#f8fafc", fontFamily: "monospace", fontSize: "11px", lineHeight: 1.35, overflowWrap: "anywhere" }}>{displayAddress}</div>
          <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.35 }}>
            The QR code includes the temporary pairing key. Only reveal the manual URL if scanning is unavailable.
          </div>
        </div>
        <FreshActionButton disabled={!pairingURL} onActivate={() => setShowManualURL((current) => !current)}>
          {showManualURL ? "Hide Manual URL" : "Show Manual URL"}
        </FreshActionButton>
        {showManualURL && (
          <div style={{ ...freshSectionStyle, background: "#111827", borderColor: "#334155" }}>
            <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 900, textTransform: "uppercase" }}>Manual Pairing URL</div>
            <div style={{ color: "#f8fafc", fontFamily: "monospace", fontSize: "10px", lineHeight: 1.35, overflowWrap: "anywhere" }}>
              {pairingURL || "Start the server to generate a pairing URL."}
            </div>
          </div>
        )}
        <FreshActionButton onActivate={props.closeModal}>Close</FreshActionButton>
      </Focusable>
    </ModalRoot>
  );
}

function FreshProfilePickerModal(props: {
  appID: string;
  profiles: Profile[];
  activeProfileID: number;
  onSelectProfile: (profile: Profile) => Promise<void>;
  onChanged: () => Promise<void>;
  closeModal: () => void;
}) {
  const [profileName, setProfileName] = useState("");
  const [copyActive, setCopyActive] = useState(true);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

  async function createProfile() {
    const name = profileName.trim();
    if (!name || busy) return;
    try {
      setBusy(true);
      setMessage("");
      const result = await call<[string, string, number], { ok: boolean; error?: string; profile?: Profile }>(
        "create_game_profile",
        props.appID,
        name,
        copyActive ? props.activeProfileID : 0
      );
      if (!result.ok || !result.profile) {
        setMessage(result.error || "Unable to create profile.");
        return;
      }
      await props.onChanged();
      await props.onSelectProfile(result.profile);
      props.closeModal();
    } catch (err) {
      setMessage(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <ModalRoot onCancel={props.closeModal} bAllowFullSize bHideCloseIcon>
      <style>{deckyRuntimeStyles}</style>
      <Focusable flow-children="down" style={{ color: "#f8fafc", display: "grid", gap: "10px", minWidth: 0, padding: "4px", width: "100%" }}>
        <div style={{ fontSize: "16px", fontWeight: 900 }}>Profiles</div>
        <Focusable flow-children="down" style={{ display: "grid", gap: "8px", maxHeight: "360px", overflowY: "auto", paddingRight: "4px" }}>
          {props.profiles.map((profile) => (
            <Focusable
              key={profile.id}
              className="dmm-sidebar-row"
              focusClassName="dmm-sidebar-row-focused"
              onActivate={() => {
                void props.onSelectProfile(profile).then(props.closeModal);
              }}
              onClick={() => {
                void props.onSelectProfile(profile).then(props.closeModal);
              }}
              style={freshCardStyle(profile.id === props.activeProfileID)}
            >
              <div style={{ fontWeight: 900 }}>{profile.name}</div>
              <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>{profileCountText(profile)}</div>
            </Focusable>
          ))}
        </Focusable>
        <div style={{ ...freshSectionStyle, background: "#0b1220" }}>
          <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 900, textTransform: "uppercase" }}>Add New Profile</div>
          <TextField label="Profile Name" value={profileName} bShowClearAction onChange={(event) => setProfileName(event.currentTarget.value)} />
          <ToggleField label="Copy current profile" checked={copyActive} onChange={setCopyActive} />
          <FreshActionButton kind="primary" disabled={!profileName.trim() || busy} onActivate={createProfile}>
            {busy ? "Creating" : "Create Profile"}
          </FreshActionButton>
          {message && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{message}</div>}
        </div>
        <FreshActionButton onActivate={props.closeModal}>Close</FreshActionButton>
      </Focusable>
    </ModalRoot>
  );
}

function FreshLocalArchiveModal(props: {
  appID: string;
  profileID: number;
  onImported: () => Promise<void>;
  closeModal: () => void;
}) {
  const [entries, setEntries] = useState<LocalArchiveBrowseEntry[]>([]);
  const [currentPath, setCurrentPath] = useState("");
  const [parentPath, setParentPath] = useState("");
  const [pathInput, setPathInput] = useState("");
  const [busyPath, setBusyPath] = useState("");
  const [message, setMessage] = useState("");

  async function browse(path = "") {
    try {
      setMessage("");
      const result = await call<[string, string], { ok: boolean; error?: string; roots: string[]; entries: LocalArchiveBrowseEntry[]; current_path: string; parent_path?: string }>(
        "browse_local_archives",
        props.appID,
        path
      );
      if (!result.ok) {
        setMessage(result.error || "Unable to browse Deck Downloads.");
        return;
      }
      setEntries(result.entries ?? []);
      setCurrentPath(result.current_path ?? "");
      setParentPath(result.parent_path ?? "");
      setPathInput(result.current_path ?? "");
    } catch (err) {
      setMessage(err instanceof Error ? err.message : String(err));
    }
  }

  async function importArchive(entry: LocalArchiveBrowseEntry) {
    if (entry.kind !== "file" || busyPath) return;
    try {
      setBusyPath(entry.path);
      setMessage("");
      const result = await call<[string, string, number], { ok: boolean; error?: string; result?: { job?: Job }; job?: Job }>(
        "import_local_archive",
        props.appID,
        entry.path,
        props.profileID
      );
      if (!result.ok) {
        setMessage(result.error || "Unable to import archive.");
        return;
      }
      const job = result.job ?? result.result?.job;
      if (job) await maybeShowDeckyActionToast(job, "fresh-local-archive-import");
      await props.onImported();
      props.closeModal();
    } catch (err) {
      setMessage(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyPath("");
    }
  }

  useEffect(() => {
    void browse("");
  }, []);

  return (
    <ModalRoot onCancel={props.closeModal} bAllowFullSize bHideCloseIcon>
      <style>{deckyRuntimeStyles}</style>
      <Focusable flow-children="down" style={{ color: "#f8fafc", display: "grid", gap: "10px", minWidth: 0, padding: "4px", width: "100%" }}>
        <div style={{ display: "grid", gap: "4px", minWidth: 0 }}>
          <div style={{ fontSize: "16px", fontWeight: 900 }}>Import Archive</div>
          <div style={{ color: "#a1a1aa", fontSize: "11px", overflowWrap: "anywhere" }}>{currentPath || "Deck Downloads"}</div>
        </div>
        <div style={freshActionRowStyle}>
          <FreshActionButton disabled={!parentPath} onActivate={() => void browse(parentPath)}>Up</FreshActionButton>
          <FreshActionButton onActivate={() => void browse("")}>Downloads</FreshActionButton>
        </div>
        <div style={{ ...freshSectionStyle, background: "#0b1220" }}>
          <TextField label="Path" value={pathInput} bShowClearAction onChange={(event) => setPathInput(event.currentTarget.value)} />
          <FreshActionButton onActivate={() => void browse(pathInput.trim())}>Open Path</FreshActionButton>
        </div>
        <Focusable flow-children="down" style={{ display: "grid", gap: "8px", maxHeight: "480px", overflowY: "auto", paddingRight: "4px" }}>
          {entries.length === 0 && <div style={{ color: "#a1a1aa" }}>No folders or supported archives found here.</div>}
          {entries.map((entry) => {
            const isFile = entry.kind === "file";
            const busy = busyPath === entry.path;
            return (
              <Focusable
                key={entry.path}
                className="dmm-sidebar-row"
                focusClassName="dmm-sidebar-row-focused"
                onActivate={() => {
                  if (isFile) void importArchive(entry);
                  else void browse(entry.path);
                }}
                onClick={() => {
                  if (isFile) void importArchive(entry);
                  else void browse(entry.path);
                }}
                style={freshCardStyle(false)}
              >
                <div style={{ display: "flex", gap: "6px", justifyContent: "space-between", minWidth: 0 }}>
                  <div style={{ ...deckyTwoLineTextStyle, fontWeight: 900 }}>{entry.name}</div>
                  {isFile && <span style={deckySourcePillStyle("local")}>Local</span>}
                </div>
                <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>
                  {isFile ? `${formatBytes(entry.bytes)} · ${busy ? "Importing" : "A Import"}` : "A Open Folder"}
                </div>
              </Focusable>
            );
          })}
        </Focusable>
        {message && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{message}</div>}
        <FreshActionButton onActivate={props.closeModal}>Close</FreshActionButton>
      </Focusable>
    </ModalRoot>
  );
}

function FreshDeckyModManagerRoute() {
  const [initialReturnContext] = useState(() => consumeDMMDeckyReturnContext());
  const [tab, setTab] = useState<FreshDeckyTab>(initialReturnContext?.tab === "settings" ? "settings" : "games");
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [games, setGames] = useState<ManagedGame[]>([]);
  const [catalogs, setCatalogs] = useState<CatalogStatus[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [selectedGameID, setSelectedGameID] = useState(initialReturnContext?.appID ?? "");
  const [runningGame, setRunningGame] = useState<RunningGame | null>(null);
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [mods, setMods] = useState<ManagedMod[]>([]);
  const [diagnostics, setGameDiagnostics] = useState<GameDiagnostics | null>(null);
  const [deploymentStatus, setDeploymentStatus] = useState<DeploymentStatus | null>(null);
  const [installCandidates, setInstallCandidates] = useState<InstallCandidate[]>([]);
  const [workshopItems, setWorkshopItems] = useState<WorkshopItem[]>([]);
  const [favoriteGameIDs, setFavoriteGameIDs] = useState<Set<string>>(new Set());
  const [gameRecent, setGameRecent] = useState<Record<string, number>>({});
  const [gameSearch, setGameSearch] = useState("");
  const [gameSort, setGameSortState] = useState<GameSort>("recent");
  const [showDebug, setShowDebug] = useState(false);
  const [diagnosticLogs, setDiagnosticLogs] = useState<Diagnostics | null>(null);
  const [dependencies, setDependencies] = useState<Dependency[]>([]);
  const [nxm, setNXM] = useState<NXMStatus | null>(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [busyModID, setBusyModID] = useState<number | null>(null);
  const [busyJobID, setBusyJobID] = useState("");
  const [modUpdateBusy, setModUpdateBusy] = useState(false);
  const [suppressRunningAutoOpen, setSuppressRunningAutoOpen] = useState(Boolean(initialReturnContext?.appID));
  const bodyRef = useRef<HTMLDivElement | null>(null);
  const selectedGameRef = useRef<HTMLDivElement | null>(null);
  const selectedGame = games.find((game) => game.app_id === selectedGameID) ?? null;
  const selectedProfile = profiles.find((profile) => profile.is_default) ?? profiles[0] ?? null;
  const selectedNexusDomain = (selectedGame?.nexus_domains ?? [])[0]?.trim().toLowerCase() ?? "";
  const actionJobs = jobs.filter(isDeckyActionCenterJob);
  const gameActionJobs = selectedGameID ? actionJobs.filter((job) => deckyJobBelongsToAppID(job, selectedGameID)) : [];
  const runtimeWarnings = (diagnostics?.runtime_requirements ?? []).filter((requirement) => requirement.status !== "ok");
  const validationWarnings = diagnostics?.validation_warnings ?? [];
  const installedCount = mods.length + workshopItems.length;
  const enabledCount = mods.filter((mod) => mod.enabled).length + workshopItems.filter((item) => item.disabled_known ? !item.disabled_locally : item.subscribed).length;

  function applyUIPreferences(ui?: UISettings) {
    setFavoriteGameIDs(new Set((ui?.favorite_game_ids ?? []).filter((item) => typeof item === "string" && item.trim() !== "")));
    setGameRecent(ui?.recent_games ?? {});
    setGameSortState(ui?.game_sort === "az" || ui?.game_sort === "za" ? ui.game_sort : "recent");
  }

  async function patchUIPreferences(patch: Record<string, string | number | boolean>) {
    const result = await call<[Record<string, string | number | boolean>], { ok: boolean; error?: string; status?: BackendStatus }>("patch_ui_preferences", patch);
    if (result.ok && result.status) {
      setStatus(result.status);
      applyUIPreferences(result.status.backend?.ui);
    }
  }

  async function loadBaseState() {
    const [nextStatus, gamesResult, jobsResult, catalogResult] = await Promise.all([
      call<[], BackendStatus>("status"),
      call<[], { ok: boolean; error?: string; games: ManagedGame[] }>("games"),
      call<[], { ok: boolean; error?: string; jobs: Job[] }>("jobs"),
      call<[], { ok: boolean; error?: string; catalogs: CatalogStatus[] }>("catalogs")
    ]);
    applyBackendAuthFromStatus(nextStatus);
    setStatus(nextStatus);
    applyUIPreferences(nextStatus.backend?.ui);
    if (gamesResult.ok) setGames(gamesResult.games);
    else setError(gamesResult.error || "Unable to load games.");
    if (jobsResult.ok) setJobs(jobsResult.jobs);
    if (catalogResult.ok) setCatalogs(catalogResult.catalogs);
    const running = currentRunningGame();
    setRunningGame(running);
    return { games: gamesResult.ok ? gamesResult.games : [], running };
  }

  async function loadSelectedGameState(appID: string) {
    if (!appID) {
      setProfiles([]);
      setMods([]);
      setGameDiagnostics(null);
      setDeploymentStatus(null);
      setInstallCandidates([]);
      setWorkshopItems([]);
      return;
    }
    const [profilesResult, modsResult, diagnosticsResult, deploymentResult, candidatesResult, workshopResult] = await Promise.all([
      call<[string], { ok: boolean; error?: string; profiles: Profile[] }>("game_profiles", appID).catch((err) => ({
        ok: false,
        error: err instanceof Error ? err.message : String(err),
        profiles: []
      })),
      call<[string], { ok: boolean; error?: string; mods: ManagedMod[] }>("game_mods", appID).catch((err) => ({
        ok: false,
        error: err instanceof Error ? err.message : String(err),
        mods: []
      })),
      call<[string], { ok: boolean; error?: string; diagnostics?: GameDiagnostics | null }>("game_diagnostics", appID).catch((err) => ({
        ok: false,
        error: err instanceof Error ? err.message : String(err),
        diagnostics: null
      })),
      call<[string], { ok: boolean; error?: string; status?: DeploymentStatus | null }>("game_deploy_status", appID).catch((err) => ({
        ok: false,
        error: err instanceof Error ? err.message : String(err),
        status: null
      })),
      call<[string], { ok: boolean; error?: string; candidates: InstallCandidate[] }>("game_install_candidates", appID).catch((err) => ({
        ok: false,
        error: err instanceof Error ? err.message : String(err),
        candidates: []
      })),
      call<[string], { ok: boolean; error?: string; items: WorkshopItem[]; state?: WorkshopState }>("game_workshop", appID).catch((err) => ({
        ok: false,
        error: err instanceof Error ? err.message : String(err),
        items: []
      }))
    ]);
    if (profilesResult.ok) setProfiles(profilesResult.profiles);
    else setError(profilesResult.error || "Unable to load profiles.");
    if (modsResult.ok) setMods(modsResult.mods);
    else setError(modsResult.error || "Unable to load mods.");
    setGameDiagnostics(diagnosticsResult.ok ? diagnosticsResult.diagnostics ?? null : null);
    setDeploymentStatus(deploymentResult.ok ? deploymentResult.status ?? null : null);
    setInstallCandidates(candidatesResult.ok ? candidatesResult.candidates : []);
    setWorkshopItems(workshopResult.ok ? workshopResult.items : []);
    const failedSlices: string[] = [];
    if (!profilesResult.ok) failedSlices.push(`profiles:${profilesResult.error || ""}`);
    if (!modsResult.ok) failedSlices.push(`mods:${modsResult.error || ""}`);
    if (!diagnosticsResult.ok) failedSlices.push(`diagnostics:${diagnosticsResult.error || ""}`);
    if (!deploymentResult.ok) failedSlices.push(`deployment:${deploymentResult.error || ""}`);
    if (!candidatesResult.ok) failedSlices.push(`install_candidates:${candidatesResult.error || ""}`);
    if (!workshopResult.ok) failedSlices.push(`workshop:${workshopResult.error || ""}`);
    if (failedSlices.length > 0) {
      await logFrontendEvent("fresh selected game partial load failure", {
        app_id: appID,
        failed: failedSlices.join("; ")
      });
    }
  }

  async function refreshFreshState() {
    try {
      setError("");
      const result = await loadBaseState();
      const runningManageReady = result.running ? result.games.find((game) => game.app_id === result.running?.app_id && gameManageReady(game)) : null;
      const nextSelected = selectedGameID || initialReturnContext?.appID || (!suppressRunningAutoOpen && runningManageReady ? runningManageReady.app_id : "");
      if (nextSelected && result.games.some((game) => game.app_id === nextSelected)) {
        setSelectedGameID(nextSelected);
        await loadSelectedGameState(nextSelected);
      } else {
        await loadSelectedGameState("");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function selectGame(appID: string) {
    try {
      setError("");
      setMessage("");
      setSuppressRunningAutoOpen(false);
      setSelectedGameID(appID);
      const recentAt = Date.now();
      setGameRecent((current) => ({ ...current, [appID]: recentAt }));
      void patchUIPreferences({ recent_game_id: appID, recent_at: recentAt });
      await loadSelectedGameState(appID);
      focusDeckyRef(selectedGameRef, "fresh-selected-game", { app_id: appID });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function selectProfile(profile: Profile) {
    if (!profile || profile.is_default) return;
    try {
      setError("");
      setMessage("");
      const result = await call<[number], { ok: boolean; error?: string; profile?: Profile; apply?: ProfileApplyResult }>("set_default_profile", profile.id);
      if (!result.ok) {
        setError(result.error || "Unable to select profile.");
        return;
      }
      await loadSelectedGameState(selectedGameID);
      setMessage(result.apply?.message || "Profile selected.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function openProfilePicker() {
    if (!selectedGameID || !selectedProfile) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <FreshProfilePickerModal
        appID={selectedGameID}
        profiles={profiles}
        activeProfileID={selectedProfile.id}
        onSelectProfile={selectProfile}
        onChanged={() => loadSelectedGameState(selectedGameID)}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Profiles", bNeverPopOut: true }
    );
  }

  function openArchiveImport() {
    if (!selectedGameID || !selectedProfile) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <FreshLocalArchiveModal appID={selectedGameID} profileID={selectedProfile.id} onImported={() => loadSelectedGameState(selectedGameID)} closeModal={closeModal} />,
      window,
      { strTitle: "Import Archive", bNeverPopOut: true, bHideActionIcons: true, popupWidth: 720, popupHeight: 780 }
    );
  }

  function openPairPhoneModal() {
    if (!status?.auth?.enabled || !pairingURLFromStatus(status)) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(<PairPhoneModal status={status} closeModal={closeModal} />, window, { strTitle: "Pair Phone", bNeverPopOut: true, bHideActionIcons: true, popupWidth: 520, popupHeight: 720 });
  }

  function openExploreMods() {
    if (!selectedGameID || !selectedGame || !selectedProfile) return;
    if (!selectedNexusDomain) {
      setError("This game does not have a browse-capable source yet.");
      return;
    }
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <NexusBrowserModal
        appID={selectedGameID}
        gameName={selectedGame.name}
        gameDomain={selectedNexusDomain}
        profileID={selectedProfile.id}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Explore Mods", bNeverPopOut: true, bHideActionIcons: true, popupWidth: 760, popupHeight: 820 }
    );
  }

  async function launchSelectedGame() {
    if (!selectedGameID) return;
    try {
      setError("");
      setMessage("");
      const steamURL = `steam://run/${selectedGameID}`;
      if (typeof SteamClient?.URL?.ExecuteSteamURL !== "function") {
        setError("Steam launch API is unavailable in this Decky session.");
        await logFrontendEvent("fresh launch game unavailable", { app_id: selectedGameID });
        return;
      }
      SteamClient.URL.ExecuteSteamURL(steamURL);
      setMessage(`Launching ${selectedGame?.name || selectedGameID}.`);
      await logFrontendEvent("fresh launch game requested", { app_id: selectedGameID });
    } catch (err) {
      const next = err instanceof Error ? err.message : String(err);
      setError(next);
      await logFrontendEvent("fresh launch game failed", { app_id: selectedGameID, error: next });
    }
  }

  function askRestoreDeployment() {
    if (!selectedGameID || !deploymentStatus?.restore_available) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <ConfirmModal
        strTitle="Restore Last Applied State"
        strDescription={deploymentStatus.restore_summary || deploymentStatus.recovery_summary || "DMM will restore the last DMM-applied deployment for this game. Only DMM-managed files recorded in the deployment manifest are touched."}
        strOKButtonText="Restore"
        strCancelButtonText="Cancel"
        onOK={() => {
          closeModal();
          void restoreDeployment();
        }}
        onCancel={closeModal}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Restore Deployment", bNeverPopOut: true }
    );
  }

  async function restoreDeployment() {
    if (!selectedGameID || !deploymentStatus?.restore_available) return;
    try {
      setError("");
      setMessage("");
      const result = await call<[string], { ok: boolean; error?: string; job?: Job; result?: unknown }>("restore_game_deployment", selectedGameID);
      if (!result.ok) {
        setError(result.error || "Unable to restore deployment.");
        return;
      }
      await maybeShowDeckyActionToast(result.job, "fresh-restore-deployment");
      await loadSelectedGameState(selectedGameID);
      setMessage(result.job?.message || "Restore started.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function checkModUpdates() {
    if (!selectedGameID || modUpdateBusy) return;
    try {
      setModUpdateBusy(true);
      setError("");
      setMessage("");
      const result = await call<[string], { ok: boolean; error?: string; checked?: number; results: ModUpdate[] }>("check_game_mod_updates", selectedGameID);
      if (!result.ok) {
        setError(result.error || "Unable to check updates.");
        return;
      }
      await loadSelectedGameState(selectedGameID);
      setMessage(`Checked ${result.checked ?? result.results.length} mods for updates.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setModUpdateBusy(false);
    }
  }

  async function toggleMod(mod: ManagedMod) {
    if (!selectedGameID || !selectedProfile || busyModID) return;
    try {
      setBusyModID(mod.id);
      setError("");
      setMessage("");
      const result = await call<[string, number, number, boolean], { ok: boolean; error?: string; apply?: ProfileApplyResult }>(
        "set_profile_mod_enabled",
        selectedGameID,
        selectedProfile.id,
        mod.id,
        !mod.enabled
      );
      if (!result.ok) {
        setError(result.error || "Unable to update mod.");
        return;
      }
      await maybeShowDeckyActionToast(result.apply?.job, "fresh-toggle-mod");
      await loadSelectedGameState(selectedGameID);
      setMessage(result.apply?.message || `${mod.enabled ? "Disabled" : "Enabled"} ${mod.name}.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  async function installModUpdate(mod: ManagedMod) {
    if (!selectedGameID || busyModID || mod.update?.status !== "available") return;
    try {
      setBusyModID(mod.id);
      setError("");
      setMessage("");
      const result = await call<[string, number], { ok: boolean; error?: string; result?: { job?: Job; browser_required?: boolean; resolved?: { source_url?: string }; file_url?: string }; job?: Job }>("update_game_mod", selectedGameID, mod.id);
      if (!result.ok) {
        setError(result.error || "Unable to install update.");
        return;
      }
      if (result.result?.browser_required) {
        const browserURL = String(result.result.file_url || result.result.resolved?.source_url || "").trim();
        if (!browserURL) {
          setError("This update needs a Nexus browser handoff, but DMM did not receive a page URL.");
          return;
        }
        const opened = await openDMMBrowserViewCapture(browserURL, {
          appID: selectedGameID,
          profileID: selectedProfile?.id ?? 0,
          source: "decky-mod-update",
          title: `DMM Update - ${mod.name}`
        });
        if (!opened) {
          setError("DMM could not open the controlled Nexus browser. Check Debug Live Logs.");
          return;
        }
        await maybeShowDeckyActionToast(result.job ?? result.result.job, "fresh-update-mod-browser-required");
        setMessage(`Opening ${mod.name} on Nexus. Click Nexus Mod Manager Download there to capture the update.`);
        return;
      }
      await maybeShowDeckyActionToast(result.job ?? result.result?.job, "fresh-update-mod");
      await loadSelectedGameState(selectedGameID);
      setMessage(result.job?.message || result.result?.job?.message || "Update install started.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  async function reinstallMod(mod: ManagedMod, promptInstallerChoices = false) {
    if (!selectedGameID || busyModID) return;
    try {
      setBusyModID(mod.id);
      setError("");
      setMessage("");
      const result = await call<[string, number, boolean], { ok: boolean; error?: string; result?: { job?: Job; candidate?: InstallCandidate } }>(
        "reinstall_game_mod",
        selectedGameID,
        mod.id,
        promptInstallerChoices
      );
      if (!result.ok) {
        setError(result.error || "Unable to reinstall mod.");
        return;
      }
      const candidate = result.result?.candidate;
      if (candidate && installerForCandidate(candidate)) {
        await openInstallerChoiceModalForCandidate(selectedGameID, candidate, "fresh-reconfigure", () => {
          void loadSelectedGameState(selectedGameID);
        }, selectedProfile?.id ?? 0);
      }
      await maybeShowDeckyActionToast(result.result?.job, "fresh-reinstall-mod");
      await loadSelectedGameState(selectedGameID);
      setMessage(result.result?.job?.message || (promptInstallerChoices ? "Installer choices opened." : "Reinstall started."));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  function askRemoveMod(mod: ManagedMod) {
    if (!selectedGameID || !selectedProfile) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <ConfirmModal
        strTitle={`Remove ${mod.name}`}
        strDescription="DMM will remove this mod from the selected profile and apply the profile. Cached downloads are kept."
        strOKButtonText="Remove"
        strCancelButtonText="Cancel"
        onOK={() => {
          closeModal();
          void removeMod(mod);
        }}
        onCancel={closeModal}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Remove Mod", bNeverPopOut: true }
    );
  }

  async function removeMod(mod: ManagedMod) {
    if (!selectedGameID || !selectedProfile || busyModID) return;
    try {
      setBusyModID(mod.id);
      setError("");
      setMessage("");
      const result = await call<[string, number, number], { ok: boolean; error?: string; result?: { apply?: ProfileApplyResult } }>(
        "remove_profile_mod",
        selectedGameID,
        selectedProfile.id,
        mod.id
      );
      if (!result.ok) {
        setError(result.error || "Unable to remove mod.");
        return;
      }
      await maybeShowDeckyActionToast(result.result?.apply?.job, "fresh-remove-mod");
      await loadSelectedGameState(selectedGameID);
      setMessage(result.result?.apply?.message || "Mod removed.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  async function runExtensionNoticeLaunchTool(job: Job, appID: string) {
    const payload = job.payload ?? {};
    const launchOptions = String(payload.tool_launch_options || "").trim();
    const toolID = String(payload.tool_id || "").trim();
    const toolName = extensionNoticeToolName(job) || toolID || "Extension tool";
    const unavailable = extensionNoticeRunToolError(job);
    if (!extensionNoticeRunToolAvailable(job)) {
      throw new Error(unavailable || "This extension tool action is not available.");
    }
    if (!appID) {
      throw new Error("Extension tool action did not include a Steam app ID.");
    }
    if (!launchOptions) {
      throw new Error("Extension tool action did not include launch options.");
    }

    const started = await call<[string], { ok: boolean; error?: string; proceed?: boolean; job?: Job }>("start_extension_notice_action", job.id);
    if (!started.ok) {
      throw new Error(started.error || "Unable to start extension tool action.");
    }
    if (!started.proceed) {
      setMessage("Extension tool action is already being handled.");
      return;
    }

    try {
      const steamApps = typeof SteamClient !== "undefined" ? SteamClient?.Apps : undefined;
      if (typeof steamApps?.RunGame !== "function") {
        throw new Error("Steam run-game API is unavailable in this Decky context.");
      }
      await logFrontendEvent("extension notice launch tool running", { job_id: job.id, app_id: appID, tool_id: toolID });
      steamApps.RunGame(appID, launchOptions, 0, STEAM_LAUNCH_SOURCE_DASH_APP_LAUNCH_CMD_LINE);
      await sleep(700);
      const report = await call<[string, Record<string, string | boolean>], { ok: boolean; error?: string; job?: Job }>("record_extension_notice_action", job.id, {
        applied: true,
        source: "decky-manual"
      });
      if (!report.ok) {
        throw new Error(report.error || "Unable to record extension tool launch.");
      }
      await maybeShowDeckyActionToast(report.job, "fresh-extension-notice-action");
      showLaunchToast("DMM extension tool", `${toolName} launch requested.`);
      setMessage(`${toolName} launch requested.`);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      await logFrontendEvent("extension notice launch tool failed", { job_id: job.id, app_id: appID, tool_id: toolID, error: message });
      await call<[string, Record<string, string | boolean>], { ok: boolean }>("record_extension_notice_action", job.id, {
        applied: false,
        error: message,
        source: "decky-manual"
      });
      throw err;
    }
  }

  async function activateActionJob(job: Job) {
    try {
      setBusyJobID(job.id);
      setError("");
      setMessage("");
      const appID = String(job.app_id || job.payload?.app_id || "").trim();
      if (job.type === "installer-choice" && appID) {
        const result = await call<[string], { ok: boolean; error?: string; candidates: InstallCandidate[] }>("game_install_candidates", appID);
        const candidateID = Number(job.payload?.candidate_id ?? 0);
        const candidate = result.ok ? result.candidates.find((item) => item.id === candidateID) ?? result.candidates[0] : null;
        if (candidate && installerForCandidate(candidate)) {
          await openInstallerChoiceModalForCandidate(appID, candidate, "fresh-actions", () => loadSelectedGameState(appID), selectedProfile?.id ?? 0);
          await loadBaseState();
          return;
        }
      }
      if (job.type === "captured-install" && job.status === "waiting") {
        const profileID = Number(job.payload?.profile_id || job.payload?.target_profile_id || selectedProfile?.id || 0);
        const result = await call<[string, number], { ok: boolean; error?: string; job?: Job; result?: { job?: Job } }>("install_captured_install", job.id, profileID);
        if (!result.ok) setError(result.error || "Unable to install.");
        else await maybeShowDeckyActionToast(result.job ?? result.result?.job, "fresh-action-install");
      } else if (job.type === "captured-install" && job.status === "failed") {
        const result = await call<[string], { ok: boolean; error?: string; job?: Job; result?: { job?: Job } }>("retry_captured_install", job.id);
        if (!result.ok) setError(result.error || "Unable to retry.");
        else await maybeShowDeckyActionToast(result.job ?? result.result?.job, "fresh-action-retry");
      } else if (job.type === "extension-notice" && extensionNoticeRunToolAvailable(job)) {
        await runExtensionNoticeLaunchTool(job, appID);
      } else if (job.type === "extension-notice" && extensionNoticeHelpURL(job)) {
        await openDMMBrowserViewCapture(extensionNoticeHelpURL(job), {
          appID,
          profileID: selectedProfile?.id ?? 0,
          source: "fresh-extension-notice-help",
          title: extensionNoticeActionLabel(job)
        });
      } else if (job.type === "extension-notice" && extensionNoticeActionKind(job) === EXTENSION_NOTICE_ACTION_RUN_LAUNCH_TOOL && extensionNoticeRunToolError(job)) {
        setError(extensionNoticeRunToolError(job));
      } else if (appID) {
        setTab("games");
        setSelectedGameID(appID);
        await loadSelectedGameState(appID);
      }
      await loadBaseState();
      if (selectedGameID) await loadSelectedGameState(selectedGameID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyJobID("");
    }
  }

  async function cancelActionJob(job: Job) {
    if (!deckyJobCanCancel(job)) return;
    try {
      setBusyJobID(job.id);
      const result = await call<[string], { ok: boolean; error?: string; job?: Job; result?: { job?: Job } }>("cancel_job", job.id);
      if (!result.ok) setError(result.error || "Unable to cancel action.");
      else await maybeShowDeckyActionToast(result.job ?? result.result?.job, "fresh-cancel-action");
      await loadBaseState();
      if (selectedGameID) await loadSelectedGameState(selectedGameID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyJobID("");
    }
  }

  async function setAutoInstallCapturedDownloads(autoInstall: boolean) {
    const result = await call<[boolean], { ok: boolean; error?: string }>("set_auto_install_captured_downloads", autoInstall);
    if (!result.ok) setError(result.error || "Unable to update install settings.");
    await refreshFreshState();
  }

  async function setAutoEnableInstalledMods(autoEnable: boolean) {
    const result = await call<[boolean], { ok: boolean; error?: string }>("set_auto_enable_installed_mods", autoEnable);
    if (!result.ok) setError(result.error || "Unable to update enable settings.");
    await refreshFreshState();
  }

  async function setAutoShowFOMODInstallers(autoShow: boolean) {
    const result = await call<[boolean], { ok: boolean; error?: string }>("set_auto_show_fomod_installers", autoShow);
    if (!result.ok) setError(result.error || "Unable to update installer settings.");
    await refreshFreshState();
  }

  async function setLanOnly(lanOnly: boolean) {
    const result = await call<[boolean], { ok: boolean; error?: string }>("set_lan_only", lanOnly);
    if (!result.ok) setError(result.error || "Unable to update security settings.");
    await refreshFreshState();
  }

  async function refreshDebugState() {
    const [deps, nextNXM, logs] = await Promise.all([
      call<[], Dependency[]>("dependencies"),
      call<[], NXMStatus>("nxm_status"),
      call<[], Diagnostics>("diagnostics")
    ]);
    setDependencies(deps);
    setNXM(nextNXM);
    setDiagnosticLogs(logs);
  }

  async function toggleFreshServer() {
    const next = await call<[], BackendStatus>(status?.running ? "stop_server" : "start_server");
    applyBackendAuthFromStatus(next);
    setStatus(next);
    await refreshFreshState();
  }

  useEffect(() => {
    void refreshFreshState();
    const timer = window.setInterval(() => void refreshFreshState(), 5000);
    return () => window.clearInterval(timer);
  }, []);

  useEffect(() => {
    if (!showDebug) return;
    void refreshDebugState();
    const timer = window.setInterval(() => void refreshDebugState(), 2500);
    return () => window.clearInterval(timer);
  }, [showDebug]);

  useEffect(() => {
    const listener = (rawEvent: Event) => {
      const event = (rawEvent as CustomEvent<DomainEvent>).detail;
      if (!event) return;
      if (event.type === "ui.changed" && isUISettings(event.payload)) {
        applyUIPreferences(event.payload);
        return;
      }
      if (["job.updated", "jobs.snapshot", "profile_mods.changed", "deployment.changed", "install.changed", "launch.changed", "mod_updates.changed", "workshop.changed", "game.changed"].includes(event.type)) {
        void refreshFreshState();
      }
    };
    window.addEventListener(DMM_EVENT_NAME, listener);
    return () => window.removeEventListener(DMM_EVENT_NAME, listener);
  }, [selectedGameID, suppressRunningAutoOpen]);

  useEffect(() => {
    if (tab !== "games" || selectedGameID || suppressRunningAutoOpen || games.length === 0) return;
    const running = currentRunningGame();
    const runningReady = running ? games.find((game) => game.app_id === running.app_id && gameManageReady(game)) : null;
    if (runningReady) void selectGame(runningReady.app_id);
  }, [tab, selectedGameID, suppressRunningAutoOpen, games]);

  const visibleGames = useMemo(() => {
    const search = gameSearch.trim().toLowerCase();
    const favorites = favoriteGameIDs;
    return [...games]
      .filter((game) => gameManageReady(game))
      .filter((game) => !search || game.name.toLowerCase().includes(search) || game.app_id.includes(search))
      .sort((a, b) => {
        const favoriteDelta = Number(favorites.has(b.app_id)) - Number(favorites.has(a.app_id));
        if (favoriteDelta !== 0) return favoriteDelta;
        if (gameSort === "az") return a.name.localeCompare(b.name);
        if (gameSort === "za") return b.name.localeCompare(a.name);
        const recentDelta = (gameRecent[b.app_id] ?? 0) - (gameRecent[a.app_id] ?? 0);
        if (recentDelta !== 0) return recentDelta;
        return a.name.localeCompare(b.name);
      });
  }, [games, gameSearch, favoriteGameIDs, gameSort, gameRecent]);

  function clearSelectedGame() {
    setSelectedGameID("");
    setSuppressRunningAutoOpen(true);
    setProfiles([]);
    setMods([]);
    setGameDiagnostics(null);
    setDeploymentStatus(null);
    setInstallCandidates([]);
    setWorkshopItems([]);
  }

  function cycleTab(delta: -1 | 1) {
    const index = Math.max(0, freshDeckyTabs.findIndex((item) => item.id === tab));
    const next = freshDeckyTabs[(index + delta + freshDeckyTabs.length) % freshDeckyTabs.length];
    setTab(next.id);
    if (next.id !== "games") setSuppressRunningAutoOpen(false);
  }

  function snapBodyToTopIfFocusIsNearTop(event?: { target?: EventTarget | null }) {
    const body = bodyRef.current;
    const target = event?.target instanceof HTMLElement ? event.target : null;
    if (!body || !target || body.scrollTop <= 0 || !body.contains(target)) return;
    const firstFocusable = Array.from(
      body.querySelectorAll<HTMLElement>(".dmm-sidebar-row, .dmm-focus-card, input, textarea, button, [tabindex]:not([tabindex='-1'])")
    ).find((element) => element.offsetParent !== null && !element.hasAttribute("disabled") && element.getAttribute("aria-disabled") !== "true");
    if (!firstFocusable || (target !== firstFocusable && !firstFocusable.contains(target) && !target.contains(firstFocusable))) return;
    requestAnimationFrame(() => {
      if (body.scrollTop > 0) body.scrollTo({ top: 0, behavior: "auto" });
    });
  }

  function handleRouteCancel(event: Pick<GamepadEvent, "preventDefault" | "stopPropagation">) {
    if (tab === "games" && selectedGameID) {
      event.preventDefault();
      event.stopPropagation();
      clearSelectedGame();
      return true;
    }
    return false;
  }

  function handleRouteButtonDown(event: GamepadEvent) {
    if (event.detail.button === GamepadButton.CANCEL && handleRouteCancel(event)) return;
    if (event.detail.button === GamepadButton.BUMPER_LEFT) {
      event.preventDefault();
      event.stopPropagation();
      cycleTab(-1);
      return;
    }
    if (event.detail.button === GamepadButton.BUMPER_RIGHT) {
      event.preventDefault();
      event.stopPropagation();
      cycleTab(1);
    }
  }

  function renderActions() {
    const runningReady = runningGame && games.find((game) => game.app_id === runningGame.app_id && gameManageReady(game));
    return (
      <>
        {runningGame && (
          <Focusable
            className="dmm-sidebar-row"
            focusClassName="dmm-sidebar-row-focused"
            onActivate={() => {
              if (runningReady) {
                setTab("games");
                void selectGame(runningGame.app_id);
              }
            }}
            onClick={() => {
              if (runningReady) {
                setTab("games");
                void selectGame(runningGame.app_id);
              }
            }}
            style={freshCardStyle(Boolean(runningReady))}
          >
            <div style={{ color: runningReady ? "#99f6e4" : "#fbbf24", fontSize: "11px", fontWeight: 900, textTransform: "uppercase" }}>
              {runningReady ? "Now Playing" : "Running"}
            </div>
            <div style={{ fontSize: "15px", fontWeight: 900, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{runningGame.name}</div>
            <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>{runningReady ? "A Open Game" : "No DMM install support yet"}</div>
          </Focusable>
        )}
        {actionJobs.length === 0 && (
          <div style={freshSectionStyle}>
            <div style={{ color: "#99f6e4", fontWeight: 900 }}>All clear</div>
            <div style={{ color: "#a1a1aa", fontSize: "12px" }}>Downloads, installer choices, and Deck actions that need attention will appear here.</div>
          </div>
        )}
        {actionJobs.map((job) => {
          const source = sourceForJob(job);
          const primary = deckyJobPrimaryActionLabel(job) || "Open";
          return (
            <Focusable
              key={job.id}
              className="dmm-sidebar-row"
              focusClassName="dmm-sidebar-row-focused"
              onActivate={() => void activateActionJob(job)}
              onClick={() => void activateActionJob(job)}
              onSecondaryActionDescription={deckyJobCanCancel(job) ? "Cancel" : undefined}
              onSecondaryButton={(event) => {
                event.preventDefault();
                event.stopPropagation();
                void cancelActionJob(job);
              }}
              style={freshCardStyle(job.status === "waiting" || job.status === "running")}
            >
              <div style={{ display: "flex", gap: "6px", justifyContent: "space-between", minWidth: 0 }}>
                <div style={{ ...deckyTwoLineTextStyle, fontWeight: 900 }}>{job.title}</div>
                <span style={deckySourcePillStyle(source)}>{sourceLabel(source)}</span>
              </div>
              <div style={{ color: job.status === "failed" ? "#f87171" : "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>{deckyJobStatusLabel(job)}</div>
              {job.message && <div style={{ color: "#d4d4d8", fontSize: "12px", lineHeight: 1.25, overflowWrap: "anywhere" }}>{job.message}</div>}
              {job.type === "extension-notice" && extensionNoticeRunToolError(job) && (
                <div style={{ color: "#fbbf24", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>{extensionNoticeRunToolError(job)}</div>
              )}
              <DeckyJobProgress job={job} />
              <DeckyJobIssueReview job={job} />
              <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 900 }}>
                {busyJobID === job.id ? "Working" : `A ${primary}`}{deckyJobCanCancel(job) ? " · Y Cancel" : ""}
              </div>
            </Focusable>
          );
        })}
      </>
    );
  }

  function renderGameList() {
    return (
      <>
        <TextField label="Search Games" value={gameSearch} bShowClearAction onChange={(event) => setGameSearch(event.currentTarget.value)} />
        <FreshActionButton onActivate={() => {
          const next = nextGameSort(gameSort);
          setGameSortState(next);
          void patchUIPreferences({ game_sort: next });
        }}>
          Sort: {gameSortLabel(gameSort)}
        </FreshActionButton>
        {visibleGames.length === 0 && <div style={freshSectionStyle}>No manage-ready games match this search.</div>}
        {visibleGames.map((game) => {
          const favorite = favoriteGameIDs.has(game.app_id);
          return (
            <Focusable
              key={game.app_id}
              className="dmm-sidebar-row"
              focusClassName="dmm-sidebar-row-focused"
              onActivate={() => void selectGame(game.app_id)}
              onClick={() => void selectGame(game.app_id)}
              onSecondaryActionDescription={favorite ? "Unfavorite" : "Favorite"}
              onSecondaryButton={(event) => {
                event.preventDefault();
                event.stopPropagation();
                const next = new Set(favoriteGameIDs);
                if (favorite) next.delete(game.app_id);
                else next.add(game.app_id);
                setFavoriteGameIDs(next);
                void patchUIPreferences({ favorite_game_id: game.app_id, favorite: !favorite });
              }}
              style={freshGameCardStyle(favorite)}
            >
              <div style={freshGameCardContentStyle}>
                <img src={steamHeaderImage(game.app_id)} style={freshGameImageStyle} />
                <div style={{ minWidth: 0 }}>
                  <div style={{ ...deckyTwoLineTextStyle, fontWeight: 900 }}>{favorite ? "★ " : ""}{game.name}</div>
                  <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>A Open · Y {favorite ? "Unfavorite" : "Favorite"}</div>
                </div>
              </div>
            </Focusable>
          );
        })}
      </>
    );
  }

  function renderSelectedGame() {
    const canBrowse = Boolean(selectedNexusDomain && catalogs.find((catalog) => catalog.id === "nexus")?.status === "ready");
    return (
      <>
        <Focusable
          ref={selectedGameRef}
          className="dmm-sidebar-row"
          focusClassName="dmm-sidebar-row-focused"
          preferredFocus
          onCancelButton={(event) => {
            event.preventDefault();
            event.stopPropagation();
            clearSelectedGame();
          }}
          style={freshGameCardStyle(true)}
        >
          <div style={freshGameCardContentStyle}>
            <img src={steamHeaderImage(selectedGameID)} style={freshGameImageStyle} />
            <div style={{ minWidth: 0 }}>
              <div style={{ ...deckyTwoLineTextStyle, fontSize: "15px", fontWeight: 900 }}>{selectedGame?.name || selectedGameID}</div>
              <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>{enabledCount} enabled / {installedCount} installed · B Games</div>
            </div>
          </div>
        </Focusable>
        <FreshActionButton onActivate={openProfilePicker} disabled={!selectedProfile}>
          Profile: {selectedProfile?.name || "None"} ▼
        </FreshActionButton>
        <FreshActionButton kind="primary" onActivate={launchSelectedGame}>
          Launch Game
        </FreshActionButton>
        <div style={freshActionRowStyle}>
          <FreshActionButton disabled={!canBrowse} onActivate={openExploreMods}>Explore Mods</FreshActionButton>
          <FreshActionButton disabled={!selectedProfile} onActivate={openArchiveImport}>Import Archive</FreshActionButton>
        </div>
        <FreshActionButton disabled={modUpdateBusy || mods.length === 0} onActivate={checkModUpdates}>
          {modUpdateBusy ? "Checking Updates" : "Check Updates"}
        </FreshActionButton>
        {deploymentStatus?.restore_available && (
          <Focusable
            className="dmm-sidebar-row"
            focusClassName="dmm-sidebar-row-focused"
            onActivate={askRestoreDeployment}
            onClick={askRestoreDeployment}
            style={{ ...freshCardStyle(false), borderColor: "#0f766e" }}
          >
            <div style={{ color: "#99f6e4", fontWeight: 900 }}>Recovery Available</div>
            <div style={{ color: "#d4d4d8", fontSize: "12px", lineHeight: 1.25, overflowWrap: "anywhere" }}>{deploymentStatus.restore_summary || deploymentStatus.recovery_summary || "Restore the last DMM-applied state for this game."}</div>
            <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 900 }}>A Restore Last Applied State</div>
          </Focusable>
        )}
        {runtimeWarnings.map((requirement) => {
          const helpURL = runtimeRequirementHelpURL(requirement);
          return (
            <Focusable
              key={requirement.id}
              className="dmm-sidebar-row"
              focusClassName="dmm-sidebar-row-focused"
              onActivate={() => {
                if (requirement.kind === "launch-tool") void syncLaunchActions({ force: true });
                else if (helpURL) void openDMMBrowserViewCapture(helpURL, { appID: selectedGameID, profileID: selectedProfile?.id ?? 0, source: "fresh-runtime-help", title: requirement.name });
              }}
              onClick={() => {
                if (requirement.kind === "launch-tool") void syncLaunchActions({ force: true });
                else if (helpURL) void openDMMBrowserViewCapture(helpURL, { appID: selectedGameID, profileID: selectedProfile?.id ?? 0, source: "fresh-runtime-help", title: requirement.name });
              }}
              style={{ ...freshCardStyle(false), borderColor: "#d97706" }}
            >
              <div style={{ color: "#fbbf24", fontWeight: 900 }}>Warning: {requirement.name}</div>
              <div style={{ color: "#d4d4d8", fontSize: "12px", lineHeight: 1.25, overflowWrap: "anywhere" }}>{requirement.message}</div>
              {(requirement.kind === "launch-tool" || helpURL) && <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 900 }}>A {requirement.kind === "launch-tool" ? "Retry Launch Setup" : "Open Help"}</div>}
            </Focusable>
          );
        })}
        {validationWarnings.map((warning, index) => (
          <div key={`${warning}:${index}`} style={{ ...freshSectionStyle, borderColor: "#d97706", color: "#fbbf24" }}>Warning: {warning}</div>
        ))}
        {gameActionJobs.map((job) => (
          <Focusable
            key={job.id}
            className="dmm-sidebar-row"
            focusClassName="dmm-sidebar-row-focused"
            onActivate={() => void activateActionJob(job)}
            onClick={() => void activateActionJob(job)}
            style={{ ...freshCardStyle(job.status === "waiting" || job.status === "running"), borderColor: job.status === "failed" ? "#7f1d1d" : "#334155" }}
          >
            <div style={{ color: job.status === "failed" ? "#f87171" : "#fbbf24", fontWeight: 900 }}>{job.title}</div>
            {job.message && <div style={{ color: "#d4d4d8", fontSize: "12px", lineHeight: 1.25, overflowWrap: "anywhere" }}>{job.message}</div>}
            {job.type === "extension-notice" && extensionNoticeRunToolError(job) && (
              <div style={{ color: "#fbbf24", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>{extensionNoticeRunToolError(job)}</div>
            )}
            <DeckyJobProgress job={job} />
            <DeckyJobIssueReview job={job} />
            <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 900 }}>A {deckyJobPrimaryActionLabel(job) || "Open"}</div>
          </Focusable>
        ))}
        {installCandidates.filter((candidate) => candidate.status === "blocked").map((candidate) => (
          <div key={candidate.id} style={{ ...freshSectionStyle, borderColor: "#7f1d1d" }}>
            <div style={{ color: "#f87171", fontWeight: 900 }}>Warning: {candidate.name}</div>
            <div style={{ color: "#d4d4d8", fontSize: "12px", lineHeight: 1.25, overflowWrap: "anywhere" }}>{candidate.reason}</div>
          </div>
        ))}
        {mods.length === 0 && workshopItems.length === 0 && (
          <div style={freshSectionStyle}>
            <div style={{ fontWeight: 900 }}>No installed mods</div>
            <div style={{ color: "#a1a1aa", fontSize: "12px" }}>Use Explore Mods or Import Archive to add mods to this profile.</div>
          </div>
        )}
        {mods.map((mod) => {
          const source = sourceForManagedMod(mod);
          const busy = busyModID === mod.id;
          return (
            <Focusable
              key={mod.id}
              className="dmm-sidebar-row"
              focusClassName="dmm-sidebar-row-focused"
              onActivate={() => void toggleMod(mod)}
              onClick={() => void toggleMod(mod)}
              onSecondaryActionDescription={mod.update?.status === "available" ? "Update" : "Reinstall"}
              onSecondaryButton={(event) => {
                event.preventDefault();
                event.stopPropagation();
                if (mod.update?.status === "available") void installModUpdate(mod);
                else void reinstallMod(mod, false);
              }}
              onOptionsActionDescription="Remove"
              onOptionsButton={(event) => {
                event.preventDefault();
                event.stopPropagation();
                askRemoveMod(mod);
              }}
              onMenuActionDescription="Reconfigure"
              onMenuButton={(event) => {
                event.preventDefault();
                event.stopPropagation();
                void reinstallMod(mod, true);
              }}
              style={freshModCardStyle(mod.enabled)}
            >
              <div style={{ alignItems: "start", display: "grid", gap: "6px", gridTemplateColumns: "minmax(0, 1fr) auto", minWidth: 0, width: "100%" }}>
                <div style={{ ...deckyTwoLineTextStyle, fontWeight: 900 }}>{mod.name}</div>
                <span style={deckySourcePillStyle(source)}>{sourceLabel(source)}</span>
              </div>
              <div style={{ color: mod.enabled ? "#99f6e4" : "#a1a1aa", fontSize: "11px", fontWeight: 900 }}>
                {busy ? "Working" : mod.enabled ? "Enabled" : "Disabled"} · {deckyModStateLabel(mod)}
              </div>
              {mod.update?.status === "available" && <div style={{ color: "#fbbf24", fontSize: "11px", fontWeight: 900 }}>Update available{mod.update.latest_version ? ` · ${mod.update.latest_version}` : ""}</div>}
              <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 900 }}>A {mod.enabled ? "Disable" : "Enable"} · Y {mod.update?.status === "available" ? "Update" : "Reinstall"} · Options Remove · Menu Reconfigure</div>
            </Focusable>
          );
        })}
        {workshopItems.map((item) => {
          const disabled = item.disabled_known && item.disabled_locally;
          return (
            <div key={item.published_file_id} style={freshModCardStyle(!disabled)}>
              <div style={{ alignItems: "start", display: "grid", gap: "6px", gridTemplateColumns: "minmax(0, 1fr) auto", minWidth: 0, width: "100%" }}>
                <div style={{ ...deckyTwoLineTextStyle, fontWeight: 900 }}>{item.title || item.published_file_id}</div>
                <span style={deckySourcePillStyle(item.source_tag || "steam_workshop")}>{sourceLabel(item.source_tag || "steam_workshop")}</span>
              </div>
              <div style={{ color: disabled ? "#a1a1aa" : "#99f6e4", fontSize: "11px", fontWeight: 900 }}>{item.disabled_known ? disabled ? "Disabled" : "Enabled" : "Steam managed"}</div>
            </div>
          );
        })}
      </>
    );
  }

  function renderGames() {
    if (!status?.running) return <div style={freshSectionStyle}>Start the server before managing games.</div>;
    return selectedGameID ? renderSelectedGame() : renderGameList();
  }

  function renderSettings() {
    return (
      <>
        <Focusable
          className="dmm-sidebar-row"
          focusClassName="dmm-sidebar-row-focused"
          onActivate={() => void toggleFreshServer()}
          onClick={() => void toggleFreshServer()}
          style={freshSettingsPrimaryCardStyle}
        >
          <div style={{ fontWeight: 900 }}>Server</div>
          <div>Status: {status?.running ? "Running" : "Stopped"}</div>
          <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Address: {pairingDisplayAddress(status)}</div>
          <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 900 }}>A {status?.running ? "Stop Server" : "Start Server"}</div>
        </Focusable>
        <div style={freshSettingsCardStyle}>
          <div style={{ fontWeight: 900 }}>Security</div>
          <ToggleField label="LAN only" checked={status?.backend?.lan_only ?? true} disabled={!status?.running} onChange={(value) => void setLanOnly(value)} />
          <FreshActionButton disabled={!status?.auth?.enabled || !pairingURLFromStatus(status)} onActivate={openPairPhoneModal}>
            Pair Phone
          </FreshActionButton>
          <FreshActionButton disabled={!status?.auth?.enabled} onActivate={async () => {
            const next = await call<[], BackendStatus>("reset_api_token");
            applyBackendAuthFromStatus(next);
            setStatus(next);
            await refreshFreshState();
          }}>
            Reset Phone Pairing
          </FreshActionButton>
        </div>
        <div style={freshSettingsCardStyle}>
          <div style={{ fontWeight: 900 }}>Automation</div>
          <ToggleField label="Auto-install downloaded mods" checked={status?.backend?.install.auto_install_captured_downloads ?? false} disabled={!status?.running} onChange={(value) => void setAutoInstallCapturedDownloads(value)} />
          <ToggleField label="Auto-enable installed mods" checked={status?.backend?.install.auto_enable_installed_mods ?? false} disabled={!status?.running} onChange={(value) => void setAutoEnableInstalledMods(value)} />
          <ToggleField label="Auto-display installer choices" checked={status?.backend?.install.auto_show_fomod_installers ?? true} disabled={!status?.running} onChange={(value) => void setAutoShowFOMODInstallers(value)} />
        </div>
        <div style={freshSettingsToggleCardStyle}>
          <ToggleField label="Show Debug" checked={showDebug} onChange={setShowDebug} />
        </div>
        {showDebug && (
          <div style={freshSettingsCardStyle}>
            <div style={{ fontWeight: 900 }}>Debug</div>
            <div>Build: {status?.build?.short_commit || status?.build?.commit?.slice(0, 12) || "unknown"}</div>
            <div>NXM: {nxm?.registered ? "Registered" : "Not registered"}</div>
            <div>Dependencies: {dependencies.filter((dep) => dep.installed).length}/{dependencies.length} installed</div>
            <FreshActionButton onActivate={() => void refreshDebugState()}>Refresh Debug</FreshActionButton>
            <pre style={{ background: "#020617", border: "1px solid #334155", borderRadius: "6px", color: "#d4d4d8", fontFamily: "monospace", fontSize: "10px", lineHeight: 1.35, margin: 0, maxHeight: "340px", overflow: "auto", padding: "8px", whiteSpace: "pre-wrap" }}>
              {diagnosticsTerminalText(diagnosticLogs)}
            </pre>
          </div>
        )}
      </>
    );
  }

  const content = tab === "actions" ? renderActions() : tab === "games" ? renderGames() : renderSettings();

  return (
    <Focusable flow-children="down" onButtonDown={handleRouteButtonDown} onCancelButton={handleRouteCancel} style={freshDeckyShellStyle}>
      <style>{deckyRuntimeStyles}</style>
      <Focusable flow-children="right" navEntryPreferPosition={NavEntryPositionPreferences.FIRST} style={freshDeckyTabBarStyle}>
        {freshDeckyTabs.map((item) => (
          <Focusable
            key={item.id}
            className="dmm-focus-card"
            focusClassName="dmm-focus-card-focused"
            onActivate={() => setTab(item.id)}
            onClick={() => setTab(item.id)}
            style={freshTabStyle(tab === item.id)}
          >
            {item.label}
          </Focusable>
        ))}
      </Focusable>
      <Focusable
        key={`${tab}:${selectedGameID || "list"}`}
        ref={bodyRef}
        flow-children="down"
        navEntryPreferPosition={NavEntryPositionPreferences.PREFERRED_CHILD}
        onFocusCapture={snapBodyToTopIfFocusIsNearTop}
        preferredFocus
        style={freshDeckyBodyStyle}
      >
        {content}
        {message && <div style={{ ...freshSectionStyle, color: "#99f6e4" }}>{message}</div>}
        {error && <div style={{ ...freshSectionStyle, borderColor: "#7f1d1d", color: "#f87171" }}>{error}</div>}
      </Focusable>
    </Focusable>
  );
}

function QuickAccessContent() {
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [error, setError] = useState("");

  async function refreshStatus() {
    try {
      setError("");
      const nextStatus = await call<[], BackendStatus>("status");
      applyBackendAuthFromStatus(nextStatus);
      setStatus(nextStatus);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function toggleServer() {
    try {
      setError("");
      const method = status?.running ? "stop_server" : "start_server";
      const next = await call<[], BackendStatus>(method);
      applyBackendAuthFromStatus(next);
      setStatus(next);
      if (method === "start_server") {
        await seedJobNotifications({ seed: true });
        await syncLaunchActions();
        connectEventMonitor();
      } else {
        closeEventMonitor();
      }
      await refreshStatus();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function openQuickPairPhoneModal() {
    if (!status?.auth?.enabled || !pairingURLFromStatus(status)) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(<PairPhoneModal status={status} closeModal={closeModal} />, window, { strTitle: "Pair Phone", bNeverPopOut: true, bHideActionIcons: true, popupWidth: 520, popupHeight: 720 });
  }

  useEffect(() => {
    void refreshStatus();
    const listener = () => void refreshStatus();
    const timer = window.setInterval(() => void refreshStatus(), 5000);
    window.addEventListener(DMM_EVENT_NAME, listener);
    return () => {
      window.clearInterval(timer);
      window.removeEventListener(DMM_EVENT_NAME, listener);
    };
  }, []);

  return (
    <PanelSection>
      <style>{deckyRuntimeStyles}</style>
      <Focusable flow-children="down" style={deckyQuickAccessFrameStyle}>
        <PanelSectionRow>
          <ButtonItem layout="below" onClick={openDeckyModManagerRoute}>
            Open DMM
          </ButtonItem>
        </PanelSectionRow>
        <PanelSectionRow>
          <ButtonItem layout="below" onClick={toggleServer}>
            {status?.running ? "Stop Server" : "Start Server"}
          </ButtonItem>
        </PanelSectionRow>
        <PanelSectionRow>
          <ButtonItem layout="below" onClick={openQuickPairPhoneModal} disabled={!status?.auth?.enabled || !pairingURLFromStatus(status)}>
            Pair Phone
          </ButtonItem>
        </PanelSectionRow>
        <PanelSectionRow>
          <div style={{ display: "grid", gap: "6px", width: "100%" }}>
            <div>Status: {status?.running ? "Running" : "Stopped"}</div>
            <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Address: {pairingDisplayAddress(status)}</div>
            {status?.backend && <div>Games: {status.backend.game_count}</div>}
            {status?.backend && <div>Nexus: {status.backend.nexus.api_key_configured ? "Configured" : "Missing"}</div>}
            {error && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{error}</div>}
            {status?.error && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{status.error}</div>}
          </div>
        </PanelSectionRow>
      </Focusable>
    </PanelSection>
  );
}

export default definePlugin(() => {
  startBackgroundMonitors();
  routerHook.addRoute(DMM_BROWSER_ROUTE, () => <DMMNativeBrowserRoute />);
  routerHook.addRoute(DMM_DECKY_ROUTE, () => <FreshDeckyModManagerRoute />);
  return {
    name: "Decky Mod Manager",
    title: <div className={staticClasses.Title}>Decky Mod Manager</div>,
    alwaysRender: true,
    content: <QuickAccessContent />,
    icon: <FaPowerOff />,
    onDismount() {
      routerHook.removeRoute(DMM_BROWSER_ROUTE);
      routerHook.removeRoute(DMM_DECKY_ROUTE);
      stopBackgroundMonitors();
    }
  };
});
