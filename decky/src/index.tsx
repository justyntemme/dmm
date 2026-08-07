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
import { ComponentType, CSSProperties, ReactNode, useEffect, useMemo, useRef, useState } from "react";

declare const SteamClient:
  | {
      Apps?: {
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
  tab: Tab;
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
  pid?: number;
  build?: BuildInfo;
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
  nexus: boolean;
  steam_workshop: boolean;
  installers: boolean;
  installer_choices: boolean;
  runtime_requirements: boolean;
  launch_tools: boolean;
  plugin_activation: boolean;
  load_order: boolean;
  game_versions: boolean;
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

type NexusSearchSort = "downloads" | "unique_downloads" | "popular" | "updated" | "name" | "relevance";
type NexusTimeWindow = "all" | "one_week" | "three_weeks" | "one_month" | "three_months" | "one_year";
type GameVisibility = "supported" | "all";

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

type Tab = "main" | "games" | "settings" | "debug";
type GameSort = "recent" | "az" | "za";
type DeckyModSort = "profile" | "source" | "az" | "za" | "enabled";

const DMM_DECKY_ROUTE = "/decky-mod-manager";
const DMM_BROWSER_ROUTE = "/decky-mod-manager-browser";
const DMM_NATIVE_BROWSER_NAME = "DeckyModManagerBrowser";
const DMM_NATIVE_BROWSER_TAB_ID = "dmm-native-browser-tab";
const deckyTabOrder: Tab[] = ["main", "games", "settings", "debug"];
const deckyTabLabels: Record<Tab, string> = {
  main: "Manage",
  games: "Games",
  settings: "Settings",
  debug: "Debug"
};

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

const deckyRouteShellStyle: CSSProperties = {
  background: "#0b1220",
  boxSizing: "border-box",
  color: "#f8fafc",
  minHeight: "100%",
  minWidth: "100%"
};

const deckyRouteContentStyle: CSSProperties = {
  boxSizing: "border-box",
  display: "grid",
  gap: "10px",
  gridTemplateRows: "40px minmax(0, 1fr)",
  height: "calc(100% - 40px)",
  marginTop: "40px",
  overflow: "hidden",
  paddingTop: "4px",
  width: "100%"
};

const deckyRouteTabBarStyle: CSSProperties = {
  boxSizing: "border-box",
  display: "grid",
  gap: "6px",
  gridTemplateColumns: "repeat(4, minmax(0, 1fr))",
  maxWidth: "100%",
  minWidth: 0,
  overflow: "hidden",
  padding: "0 28px",
  width: "100%"
};

const deckyRouteTabBodyStyle: CSSProperties = {
  boxSizing: "border-box",
  display: "grid",
  gap: "12px",
  maxWidth: "100%",
  minHeight: 0,
  overflowX: "hidden",
  overflowY: "auto",
  padding: "14px 28px 176px",
  scrollPaddingTop: "8px",
  scrollPaddingBottom: "168px",
  width: "100%"
};

function deckyRouteTabButtonStyle(active: boolean): CSSProperties {
  return {
    ...deckyFocusableCardBase,
    alignItems: "center",
    background: active ? "#0f766e" : "#1f2937",
    border: `1px solid ${active ? "#5eead4" : "#374151"}`,
    color: "#f8fafc",
    display: "flex",
    fontSize: "11px",
    fontWeight: 900,
    height: "36px",
    justifyContent: "center",
    letterSpacing: 0,
    lineHeight: 1,
    padding: "0 4px",
    textAlign: "center",
    textTransform: "uppercase",
    whiteSpace: "nowrap"
  };
}

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

function sourceForManagedMod(mod: ManagedMod) {
  return mod.source_tag || mod.catalog;
}

function sourceForInstallCandidate(candidate: InstallCandidate) {
  return candidate.source_tag || candidate.catalog;
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
    github: { border: "#52525b", color: "#f4f4f5", background: "#18181b" },
    "github-releases": { border: "#52525b", color: "#f4f4f5", background: "#18181b" },
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

function deckyTabBody(tab: Tab, content: ReactNode, contentKey: string, onCancelButton?: (event: GamepadEvent) => void) {
  return (
    <Focusable
      key={`${tab}:${contentKey}`}
      flow-children="down"
      navEntryPreferPosition={NavEntryPositionPreferences.PREFERRED_CHILD}
      onCancelActionDescription={onCancelButton ? "Back" : undefined}
      onCancelButton={onCancelButton}
      preferredFocus
      style={deckyRouteTabBodyStyle}
    >
      {content}
    </Focusable>
  );
}

function jobToastBody(job: Job): string {
  if (job.status === "waiting") return job.message || "Open Action Center on Decky or the phone/tablet UI to continue.";
  if (job.status === "running" || job.status === "queued") return job.message || "DMM is working on this action.";
  if (job.status === "completed") return job.message || "The action completed.";
  if (job.status === "failed") return job.message || "Open DMM to review the error.";
  return job.message || job.title;
}

function jobToastTitle(job: Job): string {
  if (job.status === "failed") {
    if (job.type === "deploy") return "DMM deployment failed";
    if (job.type === "rollback") return "DMM rollback failed";
    if (job.type === "purge") return "DMM purge failed";
    if (job.type === "repair") return "DMM repair failed";
    return "DMM action failed";
  }
  if (job.type === "deploy") return "DMM deployment";
  if (job.type === "rollback") return "DMM rollback";
  if (job.type === "purge") return "DMM purge";
  if (job.type === "repair") return "DMM repair";
  if (job.type === "recover-downloads") return "DMM recovery";
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
const completedLaunchActions = new Set<string>();
const launchActionAttempts = new Map<string, number>();
const completedWorkshopActions = new Set<string>();
const workshopActionAttempts = new Map<string, number>();
const workshopStateSyncLastAt = new Map<string, number>();
const handledDeckyBrowserOpenEvents = new Set<number>();
const DMM_TOAST_STORAGE_PREFIX = "decky-mod-manager:job-toast:";
const DMM_EVENT_NAME = "dmm-domain-event";
const DMM_BACKEND_WS_URL = "ws://127.0.0.1:17942/api/events/ws";
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

function eventShouldSyncLaunchActions(event: DomainEvent) {
  return ["profile_mods.changed", "deployment.changed", "install.changed", "launch.changed"].includes(event.type);
}

function eventShouldSyncWorkshopActions(event: DomainEvent) {
  return event.type === "job.updated" && isJob(event.payload) && event.payload.type === "steam-workshop-action";
}

function deckyModStateLabel(mod: ManagedMod) {
  if (mod.status === "needs_recovery") return "Needs repair";
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
  return visibility === "all" ? "All Installed" : "Supported";
}

function gameHasExtension(game?: ManagedGame | null) {
  return Boolean(game?.extension?.supported);
}

function deckyGameCapabilityBadges(game: ManagedGame): Array<{ label: string; kind: string }> {
  const extension = game.extension;
  if (!extension?.supported) return [{ label: "Unsupported", kind: "unsupported" }];
  const badges = [{ label: "DMM", kind: "dmm" }];
  if (extension.nexus) badges.push({ label: "Nexus", kind: "nexus" });
  if (extension.steam_workshop) badges.push({ label: "Workshop", kind: "workshop" });
  if (extension.installers || extension.installer_choices) badges.push({ label: "Install", kind: "installers" });
  if (extension.load_order || extension.plugin_activation) badges.push({ label: "Order", kind: "load-order" });
  if (extension.launch_tools) badges.push({ label: "Launch", kind: "launch" });
  return badges;
}

function deckyCapabilityPillStyle(kind: string): CSSProperties {
  const colors: Record<string, { border: string; color: string; background: string }> = {
    dmm: { border: "#0f766e", color: "#ccfbf1", background: "#134e4a" },
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
  if (job) showJobToast(job);
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
    if (onBrowserRoute) {
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
  return ["captured-install", "installer-choice", "deploy", "purge", "repair", "recover-downloads", "rollback", "steam-workshop-action"].includes(job.type);
}

function isJob(value: unknown): value is Job {
  return Boolean(value && typeof value === "object" && typeof (value as Job).id === "string" && typeof (value as Job).type === "string");
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

function visibleFomodSteps(installer: FomodInstaller) {
  return (installer.steps ?? []).filter((step) => step.visible !== false);
}

function installerRequiresSelections(installer: FomodInstaller) {
  return visibleFomodSteps(installer).some((step) => (step.groups ?? []).some((group) => (group.plugins ?? []).length > 0));
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
  const stateKey = `${job.status}:${job.message || ""}`;
  const previous = notifiedJobStates.get(job.id) ?? storedJobToastState(job.id);
  notifiedJobStates.set(job.id, stateKey);
  const updatedAt = Date.parse(job.updated_at || "");
  const recent = Number.isFinite(updatedAt) && Date.now() - updatedAt < 120_000;
  const requireRecent = seed || source === "event" || source === "event-snapshot";
  if (previous !== stateKey && (!requireRecent || recent) && ["waiting", "running", "completed", "failed"].includes(job.status)) {
    rememberJobToastState(job.id, stateKey);
    await logFrontendEvent("job toast shown", { job_id: job.id, status: job.status, seed, recent, type: job.type, source });
    showJobToast(job);
  }
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
        setMessage(result.error || "Unable to apply installer choices.");
        await logFrontendEvent("installer choice modal apply failed", { app_id: props.appID, candidate_id: candidate.id, profile_id: targetProfileID, error: result.error || "" });
        return;
      }
      if (result.result?.job) showJobToast(result.result.job);
      await logFrontendEvent("installer choice modal applied", { app_id: props.appID, candidate_id: candidate.id, profile_id: targetProfileID });
      props.onApplied();
      props.closeModal();
    } catch (err) {
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
    <ConfirmModal
      strTitle={candidate.name}
      strDescription={
        <div style={{ display: "grid", gap: "12px", maxHeight: "62vh", overflowY: "auto", paddingRight: "4px" }}>
          <div style={{ color: "#a1a1aa" }}>{candidate.reason || "Choose installer options before DMM adds this mod to the profile."}</div>
          {selectedChoices > 0 && (
            <div style={{ border: "1px solid #365244", borderRadius: "6px", color: "#99f6e4", padding: "8px" }}>
              {selectedChoices} choice{selectedChoices === 1 ? "" : "s"} preselected from DMM's saved/default installer state.
            </div>
          )}
          {!installer && <div style={{ color: "#f87171" }}>Installer choices are not available for this action.</div>}
          {installer && (
            <div style={{ border: "1px solid rgba(125, 211, 252, 0.28)", borderRadius: "6px", display: "grid", gap: "10px", padding: "10px" }}>
              <div style={{ alignItems: "center", display: "flex", gap: "8px", justifyContent: "space-between" }}>
                <div style={{ display: "grid", gap: "2px", minWidth: 0 }}>
                  <strong>{installer.name || "Installer Choices"}</strong>
                  <small style={{ color: "#a1a1aa" }}>{steps.length > 0 ? `Step ${currentStepIndex + 1} of ${steps.length}` : "No visible choices"}</small>
                </div>
                <span style={{ color: currentStepReady ? "#72e0a2" : "#fbbf24", fontSize: "11px", fontWeight: 800 }}>{currentStepReady ? "Ready" : "Needs selection"}</span>
              </div>
              {currentStep ? (
                <section style={{ display: "grid", gap: "8px" }}>
                  <div style={{ fontWeight: 800 }}>{currentStep.name}</div>
                  {currentStep.groups?.map((group) => (
                    <fieldset key={group.id} style={{ border: `1px solid ${fomodGroupValid(group, selections) ? "#303741" : "#fbbf24"}`, borderRadius: "6px", display: "grid", gap: "8px", margin: 0, padding: "10px" }}>
                      <legend style={{ color: "#7dd3fc", fontWeight: 800, padding: "0 4px" }}>{group.name}</legend>
                      {fomodGroupType(group) === "selectatmostone" && (
                        <label style={{ alignItems: "flex-start", display: "grid", gap: "8px", gridTemplateColumns: "22px minmax(0, 1fr)" }}>
                          <input
                            type="radio"
                            name={`candidate-${candidate.id}-${group.id}`}
                            checked={(selections[group.id] ?? []).length === 0}
                            disabled={busy}
                            onChange={() => clearGroupSelection(group)}
                          />
                          <span style={{ display: "grid", gap: "3px", minWidth: 0 }}>
                            <strong>None</strong>
                            <small style={{ color: "#a1a1aa" }}>Do not install an option from this group.</small>
                          </span>
                        </label>
                      )}
                      {group.plugins?.map((plugin) => (
                        <label key={plugin.id} style={{ alignItems: "flex-start", display: "grid", gap: "8px", gridTemplateColumns: "22px minmax(0, 1fr)", opacity: fomodPluginSelectable(plugin) ? 1 : 0.58 }}>
                          <input
                            type={fomodGroupInputType(group)}
                            name={`candidate-${candidate.id}-${group.id}`}
                            checked={pluginSelected(group, plugin)}
                            disabled={busy || fomodPluginLocked(group, plugin)}
                            onChange={(event) => setPluginSelection(group, plugin, event.currentTarget.checked)}
                          />
                          <span style={{ display: "grid", gap: "3px", minWidth: 0 }}>
                            <strong>{plugin.name}</strong>
                            {fomodPluginType(plugin) && <small style={{ color: "#a1a1aa" }}>{fomodPluginType(plugin)}</small>}
                            {plugin.description && <em style={{ color: "#d4d4d8", fontStyle: "normal", overflowWrap: "anywhere" }}>{plugin.description}</em>}
                          </span>
                        </label>
                      ))}
                      {!fomodGroupValid(group, selections) && <div style={{ color: "#fbbf24", fontSize: "11px" }}>This group needs a valid selection before continuing.</div>}
                    </fieldset>
                  ))}
                </section>
              ) : (
                <div style={{ color: "#a1a1aa" }}>This installer has no visible choices. Apply it to add the mod to the profile.</div>
              )}
              <div style={{ alignItems: "center", display: "flex", gap: "8px", justifyContent: "space-between" }}>
                <button
                  type="button"
                  disabled={busy || currentStepIndex === 0}
                  onClick={() => setStepIndex(Math.max(0, currentStepIndex - 1))}
                  style={{ minHeight: "34px", padding: "6px 10px" }}
                >
                  Back
                </button>
                <span style={{ color: "#a1a1aa", fontSize: "11px" }}>{lastStep ? "Ready to apply" : "Continue through the installer"}</span>
              </div>
            </div>
          )}
          {message && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{message}</div>}
        </div>
      }
      strOKButtonText={busy ? "Applying..." : lastStep ? "Apply Choices" : "Next"}
      strCancelButtonText="Later"
      bOKDisabled={busy || !installer || !currentStepReady || (lastStep && !installerReady)}
      onOK={continueOrApply}
      onCancel={props.closeModal}
      closeModal={props.closeModal}
    />
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
          shownInstallerChoiceModals.delete(key);
          dismissedInstallerChoiceModals.delete(key);
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
  const [sort, setSort] = useState<NexusSearchSort>("downloads");
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
      const result = await call<[string, string, string, string, number, number, boolean], { ok: boolean; error?: string; mods: NexusModResult[]; total_count: number }>(
        "nexus_mods",
        props.appID,
        query,
        nextSort,
        nextWindow,
        pageSize,
        nextOffset,
        nextVortexOnly
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

  useEffect(() => {
    void searchMods("downloads", "all", 0, false);
  }, []);

  return (
    <ModalRoot closeModal={props.closeModal} onCancel={props.closeModal} bAllowFullSize bHideCloseIcon>
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
            <input
              aria-label="Search Nexus Mods"
              placeholder="Search Nexus Mods"
              style={deckyCompactInputStyle}
              value={query}
              onChange={(event) => setQuery(event.currentTarget.value)}
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
            <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" preferredFocus onActivate={submitSearch} onClick={submitSearch} style={deckyCompactActionStyle("neutral", busy)}>
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
          {mods.map((mod) => {
            return (
              <div
                key={mod.mod_id}
                className="dmm-sidebar-row"
                style={{
                  ...deckyCompositeRowStyle(false),
                  alignSelf: "start",
                  background: "#111827",
                  flexShrink: 0,
                  minHeight: "146px",
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
                  </div>
                </div>
                <Focusable className="dmm-action-grid" flow-children="right" style={deckyActionGridStyle(1)}>
                  <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => void openModPage(mod)} onClick={() => void openModPage(mod)} style={deckyCompactActionStyle("neutral")}>
                    View Mod Page
                  </Focusable>
                </Focusable>
              </div>
            );
          })}
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
  const after = eventMonitorLastID > 0 ? `?after=${eventMonitorLastID}` : "";
  try {
    const socket = new WebSocket(`${DMM_BACKEND_WS_URL}${after}`);
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

function startBackgroundMonitors() {
  if (backgroundMonitorsStarted) return;
  backgroundMonitorsStarted = true;
  logFrontendEvent("background monitors started");
  logSteamClientCapabilities();
  seedJobNotifications({ seed: true });
  syncLaunchActions();
  syncWorkshopActions();
  seedWorkshopStateFromGames();
  connectEventMonitor();
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
    destroyActiveDMMNativeBrowser("route-close");
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

function DeckyModManagerRoute() {
  const [initialReturnContext] = useState(() => consumeDMMDeckyReturnContext());
  const selectedDeckyGameRef = useRef<HTMLDivElement | null>(null);
  const [tab, setTab] = useState<Tab>(initialReturnContext?.tab ?? "main");
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [dependencies, setDependencies] = useState<Dependency[]>([]);
  const [nxm, setNXM] = useState<NXMStatus | null>(null);
  const [importUrl, setImportUrl] = useState<string>("");
  const [importResult, setImportResult] = useState<string>("");
  const [launchResult, setLaunchResult] = useState<string>("");
  const [diagnostics, setDiagnostics] = useState<Diagnostics | null>(null);
  const [updateResult, setUpdateResult] = useState<string>("");
  const [updateBusy, setUpdateBusy] = useState<boolean>(false);
  const [error, setError] = useState<string>("");
  const [managedGames, setManagedGames] = useState<ManagedGame[]>([]);
  const [catalogs, setCatalogs] = useState<CatalogStatus[]>([]);
  const [selectedDeckyGameID, setSelectedDeckyGameID] = useState<string>(initialReturnContext?.appID ?? "");
  const [runningGame, setRunningGame] = useState<RunningGame | null>(null);
  const [deckyProfiles, setDeckyProfiles] = useState<Profile[]>([]);
  const [deckyMods, setDeckyMods] = useState<ManagedMod[]>([]);
  const [deckyInstallCandidates, setDeckyInstallCandidates] = useState<InstallCandidate[]>([]);
  const [deckyWorkshopItems, setDeckyWorkshopItems] = useState<WorkshopItem[]>([]);
  const [deckyWorkshopSupported, setDeckyWorkshopSupported] = useState<boolean>(false);
  const [deckyLoadOrder, setDeckyLoadOrder] = useState<PluginLoadOrder | null>(null);
  const [deckyDeploymentStatus, setDeckyDeploymentStatus] = useState<DeploymentStatus | null>(null);
  const [deckyDeployPlan, setDeckyDeployPlan] = useState<DeployPlan | null>(null);
  const [modsResult, setModsResult] = useState<string>("");
  const [modSearch, setModSearch] = useState<string>("");
  const [modSort, setModSort] = useState<DeckyModSort>("profile");
  const [modOrderMode, setModOrderMode] = useState<boolean>(false);
  const [gameSearch, setGameSearch] = useState<string>("");
  const [gameSort, setGameSortState] = useState<GameSort>("recent");
  const [gameVisibility, setGameVisibility] = useState<GameVisibility>("supported");
  const [favoriteGameIDs, setFavoriteGameIDs] = useState<Set<string>>(new Set());
  const [gameRecent, setGameRecent] = useState<Record<string, number>>({});
  const [busyModID, setBusyModID] = useState<number | null>(null);
  const [modUpdateBusy, setModUpdateBusy] = useState<boolean>(false);
  const [busyWorkshopKey, setBusyWorkshopKey] = useState<string>("");
  const [focusedModID, setFocusedModID] = useState<number | null>(null);
  const [focusedGameID, setFocusedGameID] = useState<string>("");
  const [focusedProfileID, setFocusedProfileID] = useState<number | null>(null);
  const [focusedCandidateID, setFocusedCandidateID] = useState<number | null>(null);
  const [focusedConflictTarget, setFocusedConflictTarget] = useState<string>("");
  const [busyConflictTarget, setBusyConflictTarget] = useState<string>("");
  const routeRefreshTimer = useRef<number | null>(null);
  const routeRefreshNeedsStatus = useRef(false);
  const routeRefreshNeedsGames = useRef(false);
  const routeRefreshNeedsGameState = useRef(false);
  const diagnosticLogText = useMemo(() => diagnosticsTerminalText(diagnostics), [diagnostics]);

  useEffect(() => {
    if (tab !== "debug") return;
    void loadDiagnostics({ quiet: true });
    const timer = window.setInterval(() => void loadDiagnostics({ quiet: true }), 2500);
    return () => window.clearInterval(timer);
  }, [tab]);

  async function refresh() {
    try {
      setError("");
      const nextStatus = await call<[], BackendStatus>("status");
      setStatus(nextStatus);
      applyDeckyUIPreferences(nextStatus);
      setDependencies(await call<[], Dependency[]>("dependencies"));
      setNXM(await call<[], NXMStatus>("nxm_status"));
      const catalogResult = await call<[], { ok: boolean; error?: string; catalogs: CatalogStatus[] }>("catalogs");
      if (catalogResult.ok) {
        setCatalogs(catalogResult.catalogs);
      } else {
        await logFrontendEvent("decky catalogs load failed", { error: catalogResult.error || "" });
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function scheduleDeckyRouteRefresh(options: { status?: boolean; games?: boolean; gameState?: boolean }) {
    routeRefreshNeedsStatus.current = routeRefreshNeedsStatus.current || Boolean(options.status);
    routeRefreshNeedsGames.current = routeRefreshNeedsGames.current || Boolean(options.games);
    routeRefreshNeedsGameState.current = routeRefreshNeedsGameState.current || Boolean(options.gameState);
    if (routeRefreshTimer.current !== null) return;
    routeRefreshTimer.current = window.setTimeout(() => {
      routeRefreshTimer.current = null;
      const needsStatus = routeRefreshNeedsStatus.current;
      const needsGames = routeRefreshNeedsGames.current;
      const needsGameState = routeRefreshNeedsGameState.current;
      routeRefreshNeedsStatus.current = false;
      routeRefreshNeedsGames.current = false;
      routeRefreshNeedsGameState.current = false;
      void (async () => {
        try {
          let games = managedGames;
          if (needsStatus) {
            await refresh();
          }
          if (needsGames) {
            games = await loadDeckyGames();
          }
          if (needsGameState && selectedDeckyGameID) {
            if (needsGames && games.length > 0 && !games.some((game) => game.app_id === selectedDeckyGameID)) {
              clearSelectedDeckyGame();
              return;
            }
            await loadDeckyGameState(selectedDeckyGameID);
          }
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          setError(message);
          await logFrontendEvent("decky route event refresh failed", { error: message });
        }
      })();
    }, 120);
  }

  function applyDeckyUIPreferences(nextStatus: BackendStatus) {
    applyDeckyUIPreferencesFromUI(nextStatus.backend?.ui);
  }

  function applyDeckyUIPreferencesFromUI(ui?: UISettings) {
    setFavoriteGameIDs(new Set((ui?.favorite_game_ids ?? []).filter((item) => typeof item === "string" && item.trim() !== "")));
    setGameRecent(ui?.recent_games ?? {});
    setGameSortState(ui?.game_sort === "az" || ui?.game_sort === "za" ? ui.game_sort : "recent");
  }

  async function patchDeckyUIPreferences(patch: Record<string, string | number | boolean>) {
    try {
      const result = await call<[Record<string, string | number | boolean>], { ok: boolean; error?: string; status?: BackendStatus }>(
        "patch_ui_preferences",
        patch
      );
      if (!result.ok) {
        await logFrontendEvent("decky ui preferences save failed", { error: result.error || "" });
        return;
      }
      if (result.status) {
        setStatus(result.status);
        applyDeckyUIPreferences(result.status);
      }
    } catch (err) {
      await logFrontendEvent("decky ui preferences save threw", { error: err instanceof Error ? err.message : String(err) });
    }
  }

  function markDeckyGameRecent(appID: string) {
    const recentAt = Date.now();
    const next = Object.fromEntries(Object.entries({ ...gameRecent, [appID]: recentAt }).sort((a, b) => b[1] - a[1]).slice(0, 50));
    setGameRecent(next);
    void patchDeckyUIPreferences({ recent_game_id: appID, recent_at: recentAt });
  }

  function toggleDeckyFavoriteGame(appID: string) {
    const next = new Set(favoriteGameIDs);
    if (next.has(appID)) next.delete(appID);
    else next.add(appID);
    setFavoriteGameIDs(next);
    void patchDeckyUIPreferences({ favorite_game_id: appID, favorite: next.has(appID) });
  }

  function cycleDeckyGameSort() {
    const next = nextGameSort(gameSort);
    setGameSortState(next);
    void patchDeckyUIPreferences({ game_sort: next });
  }

  function toggleDeckyGameVisibility() {
    setGameVisibility((current) => current === "supported" ? "all" : "supported");
  }

  function cycleDeckyModSort() {
    if (modOrderMode) return;
    setModSort((current) => nextDeckyModSort(current));
  }

  function clearSelectedDeckyGame() {
    setSelectedDeckyGameID("");
    setFocusedModID(null);
    setFocusedCandidateID(null);
    setModSearch("");
    setModOrderMode(false);
  }

  function handleDeckyTabCancel(event: GamepadEvent) {
    if (tab !== "games" || !selectedDeckyGameID) return;
    event.preventDefault();
    event.stopPropagation();
    clearSelectedDeckyGame();
    void logFrontendEvent("decky selected game cleared", { source: "cancel-button" });
  }

  async function loadDeckyGames() {
    const result = await call<[], { ok: boolean; error?: string; games: ManagedGame[] }>("games");
    if (!result.ok) {
      setError(result.error ?? "Unable to load games.");
      return [];
    }
    setManagedGames(result.games);
    return result.games;
  }

  async function loadDeckyGameState(appID: string) {
    if (!appID) {
      setDeckyProfiles([]);
      setDeckyMods([]);
      setDeckyInstallCandidates([]);
      setDeckyWorkshopItems([]);
      setDeckyWorkshopSupported(false);
      setDeckyLoadOrder(null);
      setDeckyDeploymentStatus(null);
      setDeckyDeployPlan(null);
      return null;
    }
    const [profilesResult, modsResult, candidatesResult, workshopResult, loadOrderResult, deployStatusResult, deployPreviewResult] = await Promise.all([
      call<[string], { ok: boolean; error?: string; profiles: Profile[] }>("game_profiles", appID),
      call<[string], { ok: boolean; error?: string; mods: ManagedMod[] }>("game_mods", appID),
      call<[string], { ok: boolean; error?: string; candidates: InstallCandidate[] }>("game_install_candidates", appID),
      call<[string], { ok: boolean; error?: string; state?: WorkshopState; items: WorkshopItem[] }>("game_workshop", appID),
      call<[string], { ok: boolean; error?: string; load_order?: PluginLoadOrder }>("game_load_order", appID),
      call<[string], { ok: boolean; error?: string; status?: DeploymentStatus }>("game_deploy_status", appID),
      call<[string], { ok: boolean; error?: string; plan?: DeployPlan | null }>("game_deploy_preview", appID)
    ]);
    if (!profilesResult.ok) {
      setError(profilesResult.error ?? "Unable to load profiles.");
      return null;
    }
    if (!modsResult.ok) {
      setError(modsResult.error ?? "Unable to load mods.");
      return null;
    }
    setDeckyProfiles(profilesResult.profiles);
    setDeckyMods(modsResult.mods);
    if (candidatesResult.ok) {
      setDeckyInstallCandidates(candidatesResult.candidates);
    } else {
      setDeckyInstallCandidates([]);
      setError(candidatesResult.error ?? "Unable to load installer items.");
    }
    if (workshopResult.ok) {
      setDeckyWorkshopItems(workshopResult.items);
      setDeckyWorkshopSupported(Boolean(workshopResult.state?.supported));
    } else {
      setDeckyWorkshopItems([]);
      setDeckyWorkshopSupported(false);
    }
    if (loadOrderResult.ok && loadOrderResult.load_order) {
      setDeckyLoadOrder(loadOrderResult.load_order);
    } else {
      setDeckyLoadOrder(null);
    }
    if (deployStatusResult.ok && deployStatusResult.status) {
      setDeckyDeploymentStatus(deployStatusResult.status);
    } else {
      setDeckyDeploymentStatus(null);
    }
    if (deployPreviewResult.ok && deployPreviewResult.plan) {
      setDeckyDeployPlan(deployPreviewResult.plan);
    } else {
      setDeckyDeployPlan(null);
    }
    if (workshopResult.ok && workshopResult.state?.supported) void syncWorkshopStateForApp(appID).then((synced) => {
      if (synced) {
        void call<[string], { ok: boolean; state?: WorkshopState; items: WorkshopItem[] }>("game_workshop", appID).then((next) => {
          if (!next.ok) return;
          setDeckyWorkshopItems(next.items);
          setDeckyWorkshopSupported(Boolean(next.state?.supported));
        });
      }
    });
    return {
      mods: modsResult.mods,
      candidates: candidatesResult.ok ? candidatesResult.candidates : [],
      workshopItems: workshopResult.ok ? workshopResult.items : [],
      loadOrder: loadOrderResult.ok ? loadOrderResult.load_order : null,
      deploymentStatus: deployStatusResult.ok ? deployStatusResult.status : null,
      deployPlan: deployPreviewResult.ok ? deployPreviewResult.plan : null
    };
  }

  async function refreshDeckyMods(appID = selectedDeckyGameID) {
    try {
      setError("");
      setModsResult("");
      const games = await loadDeckyGames();
      const running = currentRunningGame();
      setRunningGame(running);
      const runningSupported = running && games.some((game) => game.app_id === running.app_id);
      const selected = appID || selectedDeckyGameID || (runningSupported ? running.app_id : "");
      const nextID = selected && games.some((game) => game.app_id === selected) ? selected : "";
      setSelectedDeckyGameID(nextID);
      await loadDeckyGameState(nextID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function selectDeckyGame(appID: string) {
    try {
      setError("");
      setModsResult("");
      setModSearch("");
      setModOrderMode(false);
      setSelectedDeckyGameID(appID);
      markDeckyGameRecent(appID);
      await loadDeckyGameState(appID);
      focusDeckyRef(selectedDeckyGameRef, "selected-game", { app_id: appID, source: "select-game" });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function selectDeckyProfile(profile: Profile) {
    if (!selectedDeckyGameID || profile.is_default) return;
    try {
      setError("");
      setModsResult("");
      const result = await call<[number], { ok: boolean; error?: string; profile?: Profile; apply?: ProfileApplyResult }>("set_default_profile", profile.id);
      if (!result.ok) {
        setError(result.error ?? "Unable to select profile.");
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      const applyMessage = result.apply?.message || "Profile selected. Restart the game for changes to affect a running session.";
      if (result.apply?.status === "blocked" || result.apply?.status === "failed") {
        setError(applyMessage);
      } else {
        setModsResult(applyMessage);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function toggleDeckyMod(mod: ManagedMod, enabled: boolean) {
    const profile = deckyProfiles.find((item) => item.is_default) ?? deckyProfiles[0];
    if (!selectedDeckyGameID || !profile) return;
    try {
      setError("");
      setModsResult("");
      setBusyModID(mod.id);
      const result = await call<[string, number, number, boolean], { ok: boolean; error?: string; mod?: ManagedMod; apply?: ProfileApplyResult }>(
        "set_profile_mod_enabled",
        selectedDeckyGameID,
        profile.id,
        mod.id,
        enabled
      );
      if (!result.ok) {
        await logFrontendEvent("decky mod toggle failed", { app_id: selectedDeckyGameID, mod_id: mod.id, error: result.error || "" });
        setError(result.error ?? "Unable to update mod.");
        await loadDeckyGameState(selectedDeckyGameID);
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      const applyMessage = result.apply?.message || "Enabled mods applied. Restart the game if it is already running.";
      if (result.apply?.status === "blocked" || result.apply?.status === "failed") {
        setError(applyMessage);
      } else {
        setModsResult(applyMessage);
      }
      if (result.apply?.job) showJobToast(result.apply.job);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  function toggleDeckyModOrderMode() {
    setModOrderMode((current) => {
      const next = !current;
      if (next) setModSort("profile");
      setModsResult(next ? "Order mode enabled. Move focused mods in the selected profile." : "Order mode disabled.");
      return next;
    });
  }

  async function moveDeckyModInProfile(mod: ManagedMod, direction: -1 | 1) {
    const profile = deckyProfiles.find((item) => item.is_default) ?? deckyProfiles[0];
    if (!selectedDeckyGameID || !profile) return;
    const ordered = [...deckyMods].sort((a, b) => a.priority - b.priority || a.name.localeCompare(b.name));
    const from = ordered.findIndex((item) => item.id === mod.id);
    const to = from + direction;
    if (from < 0 || to < 0 || to >= ordered.length) {
      setModsResult(direction < 0 ? "This mod is already first in the profile." : "This mod is already last in the profile.");
      return;
    }
    [ordered[from], ordered[to]] = [ordered[to], ordered[from]];
    try {
      setError("");
      setModsResult("");
      setBusyModID(mod.id);
      const result = await call<[string, number, number[]], { ok: boolean; error?: string; mods?: ManagedMod[]; apply?: ProfileApplyResult }>(
        "set_profile_mod_order",
        selectedDeckyGameID,
        profile.id,
        ordered.map((item) => item.id)
      );
      if (!result.ok) {
        await logFrontendEvent("decky mod order failed", { app_id: selectedDeckyGameID, mod_id: mod.id, error: result.error || "" });
        setError(result.error ?? "Unable to update mod order.");
        await loadDeckyGameState(selectedDeckyGameID);
        return;
      }
      setFocusedModID(mod.id);
      if (result.mods) setDeckyMods(result.mods);
      await loadDeckyGameState(selectedDeckyGameID);
      const applyMessage = result.apply?.message || "Profile order applied. Restart the game if it is already running.";
      if (result.apply?.status === "blocked" || result.apply?.status === "failed") {
        setError(applyMessage);
      } else {
        setModsResult(applyMessage);
      }
      if (result.apply?.job) showJobToast(result.apply.job);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  async function setDeckyFileConflictWinner(target: ConflictChoiceTarget, winnerInstalledModID: number) {
    const profile = deckyProfiles.find((item) => item.is_default) ?? deckyProfiles[0];
    if (!selectedDeckyGameID || !profile || busyConflictTarget) return;
    try {
      setError("");
      setModsResult("");
      setBusyConflictTarget(target.target_path);
      const result = await call<[string, number, string, number], { ok: boolean; error?: string; apply?: ProfileApplyResult }>(
        "set_file_conflict_winner",
        selectedDeckyGameID,
        profile.id,
        target.target_path,
        winnerInstalledModID
      );
      if (!result.ok) {
        await logFrontendEvent("decky file conflict winner failed", { app_id: selectedDeckyGameID, target_path: target.target_path, winner_installed_mod_id: winnerInstalledModID, error: result.error || "" });
        setError(result.error ?? "Unable to set file winner.");
        await loadDeckyGameState(selectedDeckyGameID);
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      const applyMessage = result.apply?.message || "File winner saved and profile applied.";
      if (result.apply?.status === "blocked" || result.apply?.status === "failed") {
        setError(applyMessage);
      } else {
        setModsResult(applyMessage);
      }
      if (result.apply?.job) showJobToast(result.apply.job);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyConflictTarget("");
    }
  }

  async function clearDeckyFileConflictWinner(target: ConflictChoiceTarget) {
    const profile = deckyProfiles.find((item) => item.is_default) ?? deckyProfiles[0];
    if (!selectedDeckyGameID || !profile || busyConflictTarget) return;
    try {
      setError("");
      setModsResult("");
      setBusyConflictTarget(target.target_path);
      const result = await call<[string, number, string], { ok: boolean; error?: string; apply?: ProfileApplyResult }>(
        "clear_file_conflict_winner",
        selectedDeckyGameID,
        profile.id,
        target.target_path
      );
      if (!result.ok) {
        await logFrontendEvent("decky file conflict reset failed", { app_id: selectedDeckyGameID, target_path: target.target_path, error: result.error || "" });
        setError(result.error ?? "Unable to reset file winner.");
        await loadDeckyGameState(selectedDeckyGameID);
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      const applyMessage = result.apply?.message || "File winner reset to profile order.";
      if (result.apply?.status === "blocked" || result.apply?.status === "failed") {
        setError(applyMessage);
      } else {
        setModsResult(applyMessage);
      }
      if (result.apply?.job) showJobToast(result.apply.job);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyConflictTarget("");
    }
  }

  function askRemoveDeckyMod(mod: ManagedMod) {
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <ConfirmModal
        strTitle={`Remove ${mod.name}`}
        strDescription="DMM will remove this mod from the selected profile and apply the profile. Cached downloads are kept for recovery."
        strOKButtonText="Remove Mod"
        strCancelButtonText="Cancel"
        onOK={() => {
          closeModal();
          void removeDeckyMod(mod);
        }}
        onCancel={closeModal}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Remove Mod", bNeverPopOut: true }
    );
  }

  function askClearDeckyInstallCandidates() {
    if (!selectedDeckyGameID || deckyInstallCandidates.length === 0) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
        <ConfirmModal
          strTitle="Clear Installer Items"
          strDescription="DMM will remove blocked installer items and installer-choice actions for this game. Downloaded archives are kept in the cache."
          strOKButtonText="Clear Items"
        strCancelButtonText="Cancel"
        onOK={() => {
          closeModal();
          void clearDeckyInstallCandidates();
        }}
        onCancel={closeModal}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Clear Installer Items", bNeverPopOut: true }
    );
  }

  function askRestoreDeckyDeployment() {
    if (!selectedDeckyGameID || !selectedDeckyGame || !deckyDeploymentStatus?.restore_available) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <ConfirmModal
        strTitle={`Restore ${selectedDeckyGame.name}`}
        strDescription={deckyDeploymentStatus.restore_summary || "DMM will restore only files recorded in the active DMM deployment manifest."}
        strOKButtonText="Restore"
        strCancelButtonText="Cancel"
        onOK={() => {
          closeModal();
          void restoreDeckyDeployment();
        }}
        onCancel={closeModal}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Restore DMM Files", bNeverPopOut: true }
    );
  }

  async function restoreDeckyDeployment() {
    if (!selectedDeckyGameID || !deckyDeploymentStatus?.restore_available) return;
    try {
      setError("");
      setModsResult("");
      const result = await call<[string], { ok: boolean; error?: string; job?: Job; result?: unknown }>("restore_game_deployment", selectedDeckyGameID);
      if (!result.ok) {
        setError(result.error ?? "Unable to restore the last DMM-applied state.");
        await loadDeckyGameState(selectedDeckyGameID);
        return;
      }
      if (result.job) showJobToast(result.job);
      setModsResult(result.job?.message || "Restore completed.");
      await loadDeckyGameState(selectedDeckyGameID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function clearDeckyInstallCandidates() {
    if (!selectedDeckyGameID) return;
    try {
      setError("");
      setModsResult("");
      const result = await call<[string], { ok: boolean; error?: string; result?: { deleted?: number } }>("clear_game_install_candidates", selectedDeckyGameID);
      if (!result.ok) {
        setError(result.error ?? "Unable to clear installer items.");
        return;
      }
      setModsResult(`Cleared ${result.result?.deleted ?? deckyInstallCandidates.length} installer item(s).`);
      await loadDeckyGameState(selectedDeckyGameID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function openDeckyInstallerChoice(candidate: InstallCandidate) {
    if (!selectedDeckyGameID) return;
    void openInstallerChoiceModalForCandidate(selectedDeckyGameID, candidate, "decky-sidebar", () => {
      void loadDeckyGameState(selectedDeckyGameID);
    }, selectedProfile?.id ?? 0);
  }

  async function removeDeckyMod(mod: ManagedMod) {
    const profile = deckyProfiles.find((item) => item.is_default) ?? deckyProfiles[0];
    if (!selectedDeckyGameID || !profile) {
      setError("Select a profile before removing a mod.");
      return;
    }
    try {
      setError("");
      setModsResult("");
      setBusyModID(mod.id);
      const result = await call<[string, number, number], { ok: boolean; error?: string; result?: { mod?: ManagedMod; apply?: ProfileApplyResult } }>("remove_profile_mod", selectedDeckyGameID, profile.id, mod.id);
      if (!result.ok) {
        setError(result.error ?? "Unable to remove mod.");
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      const applyMessage = result.result?.apply?.message || "Mod removed. Restart the game if it is already running.";
      if (result.result?.apply?.status === "blocked" || result.result?.apply?.status === "failed") setError(applyMessage);
      else setModsResult(applyMessage);
      if (result.result?.apply?.job) showJobToast(result.result.apply.job);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  async function reinstallDeckyMod(mod: ManagedMod) {
    if (!selectedDeckyGameID) return;
    try {
      setError("");
      setModsResult("");
      setBusyModID(mod.id);
      const result = await call<[string, number], { ok: boolean; error?: string; result?: { job?: Job; mod?: ManagedMod } }>("reinstall_game_mod", selectedDeckyGameID, mod.id);
      if (!result.ok) {
        setError(result.error ?? "Unable to reinstall mod.");
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      const job = result.result?.job;
      if (job) {
        setModsResult(job.message || "Reinstall complete.");
        showJobToast(job);
      } else {
        setModsResult("Reinstall complete.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  async function updateDeckyMod(mod: ManagedMod) {
    if (!selectedDeckyGameID || mod.update?.status !== "available") return;
    try {
      setError("");
      setModsResult("");
      setBusyModID(mod.id);
      const result = await call<[string, number], { ok: boolean; error?: string; result?: { job?: Job; browser_required?: boolean; file_url?: string; resolved?: { source_url?: string } }; job?: Job }>("update_game_mod", selectedDeckyGameID, mod.id);
      if (!result.ok) {
        await logFrontendEvent("decky mod update failed", { app_id: selectedDeckyGameID, mod_id: mod.id, error: result.error || "" });
        setError(result.error ?? "Unable to install update.");
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      const job = result.job ?? result.result?.job;
      if (job) {
        setModsResult(job.message || "Update queued.");
        showJobToast(job);
        if (job.status === "failed" && !result.result?.browser_required) setError(job.message || "Unable to install update.");
      } else {
        setModsResult("Update queued.");
      }
      if (result.result?.browser_required) {
        const fileURL = result.result.file_url || result.result.resolved?.source_url || "";
        setModsResult("Open this update in DMM's Nexus browser, then click Nexus Mod Manager Download to capture it.");
        if (fileURL) {
          await logFrontendEvent("decky mod update browser required", {
            app_id: selectedDeckyGameID,
            mod_id: mod.id,
            url: fileURL
          });
          const opened = await openDMMBrowserViewCapture(fileURL, {
            appID: selectedDeckyGameID,
            profileID: selectedProfile?.id ?? 0,
            source: "decky-mod-update",
            title: `Update ${mod.name} - Nexus Mods`
          });
          await logFrontendEvent("decky mod update browser opened", {
            app_id: selectedDeckyGameID,
            mod_id: mod.id,
            opened
          });
          if (!opened) {
            setError("DMM could not open the controlled Nexus browser. Check Debug Live Logs.");
          }
        } else {
          setError("Open the Nexus file page from a browser and use its Mod Manager Download flow for this update.");
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyModID(null);
    }
  }

  async function checkDeckyModUpdates() {
    if (!selectedDeckyGameID || modUpdateBusy) return;
    try {
      setError("");
      setModsResult("");
      setModUpdateBusy(true);
      const result = await call<[string], { ok: boolean; error?: string; checked?: number; results: Array<{ installed_mod_id: number } & ModUpdate> }>("check_game_mod_updates", selectedDeckyGameID);
      if (!result.ok) {
        setError(result.error ?? "Unable to check updates.");
        return;
      }
      const updates = new Map<number, ModUpdate>();
      for (const item of result.results ?? []) {
        updates.set(item.installed_mod_id, {
          catalog: item.catalog,
          source_tag: item.source_tag,
          status: item.status,
          latest_file_id: item.latest_file_id,
          latest_file_name: item.latest_file_name,
          latest_version: item.latest_version,
          latest_uploaded_at: item.latest_uploaded_at,
          message: item.message,
          checked_at: item.checked_at
        });
      }
      setDeckyMods((items) => items.map((mod) => updates.has(mod.id) ? { ...mod, update: updates.get(mod.id) } : mod));
      const available = (result.results ?? []).filter((item) => item.status === "available").length;
      const checked = result.checked ?? 0;
      setModsResult(available > 0 ? `${available} update${available === 1 ? "" : "s"} available.` : `Checked ${checked} supported mod${checked === 1 ? "" : "s"}.`);
      await loadDeckyGameState(selectedDeckyGameID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setModUpdateBusy(false);
    }
  }

  function workshopItemName(item: WorkshopItem) {
    return (item.title || item.published_file_id || "Workshop item").trim();
  }

  async function queueDeckyWorkshopAction(item: WorkshopItem, kind: "enable" | "disable" | "unsubscribe") {
    if (!selectedDeckyGameID || !deckyWorkshopSupported) return;
    const key = `${item.published_file_id}:${kind}`;
    try {
      setError("");
      setModsResult("");
      setBusyWorkshopKey(key);
      const result = await call<[string, string, string], { ok: boolean; error?: string; job?: Job; result?: { job?: Job } }>(
        "queue_workshop_action",
        selectedDeckyGameID,
        item.published_file_id,
        kind
      );
      if (!result.ok) {
        setError(result.error ?? "Unable to queue Steam Workshop action.");
        return;
      }
      const job = result.job ?? result.result?.job;
      setModsResult(`${kind === "unsubscribe" ? "Unsubscribe" : kind === "disable" ? "Disable" : "Enable"} queued for ${workshopItemName(item)}.`);
      if (job) showJobToast(job);
      await syncWorkshopActions();
      await loadDeckyGameState(selectedDeckyGameID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyWorkshopKey("");
    }
  }

  async function moveDeckyWorkshopItem(item: WorkshopItem, direction: -1 | 1) {
    if (!selectedDeckyGameID || !deckyWorkshopSupported || busyWorkshopKey) return;
    const ordered = [...deckyWorkshopItems].sort((a, b) => a.position - b.position || a.published_file_id.localeCompare(b.published_file_id));
    const from = ordered.findIndex((entry) => entry.published_file_id === item.published_file_id);
    const to = from + direction;
    if (from < 0 || to < 0 || to >= ordered.length) {
      setModsResult(direction < 0 ? "This Workshop item is already first." : "This Workshop item is already last.");
      return;
    }
    [ordered[from], ordered[to]] = [ordered[to], ordered[from]];
    const itemIDs = ordered.map((entry) => entry.published_file_id);
    try {
      setError("");
      setModsResult("");
      setBusyWorkshopKey(`${item.published_file_id}:order`);
      const result = await call<[string, string[]], { ok: boolean; error?: string; job?: Job; result?: { job?: Job } }>(
        "queue_workshop_order",
        selectedDeckyGameID,
        itemIDs
      );
      if (!result.ok) {
        setError(result.error ?? "Unable to queue Steam Workshop load order.");
        await loadDeckyGameState(selectedDeckyGameID);
        return;
      }
      const job = result.job ?? result.result?.job;
      setDeckyWorkshopItems(ordered.map((entry, position) => ({ ...entry, position })));
      setModsResult("Workshop load order queued through Steam.");
      if (job) showJobToast(job);
      await syncWorkshopActions();
      await loadDeckyGameState(selectedDeckyGameID);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyWorkshopKey("");
    }
  }

  function askUnsubscribeWorkshopItem(item: WorkshopItem) {
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <ConfirmModal
        strTitle={`Unsubscribe ${workshopItemName(item)}`}
        strDescription="Steam will remove this Workshop subscription for the selected game. DMM-managed mods are not changed."
        strOKButtonText="Unsubscribe"
        strCancelButtonText="Cancel"
        onOK={() => {
          closeModal();
          void queueDeckyWorkshopAction(item, "unsubscribe");
        }}
        onCancel={closeModal}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Unsubscribe Workshop Item", bNeverPopOut: true }
    );
  }

  async function toggleServer() {
    try {
      setError("");
      const method = status?.running ? "stop_server" : "start_server";
      setStatus(await call<[], BackendStatus>(method));
      await refresh();
      if (method === "start_server") {
        await seedJobNotifications({ seed: true });
        await syncLaunchActions();
        connectEventMonitor();
      } else {
        closeEventMonitor();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function setLanOnly(lanOnly: boolean) {
    try {
      setError("");
      const result = await call<[boolean], { ok: boolean; error?: string }>("set_lan_only", lanOnly);
      if (!result.ok) setError(result.error ?? "Unable to update server settings.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function setAutoInstallCapturedDownloads(autoInstall: boolean) {
    try {
      setError("");
      const result = await call<[boolean], { ok: boolean; error?: string; status?: unknown }>("set_auto_install_captured_downloads", autoInstall);
      if (!result.ok) setError(result.error ?? "Unable to update install settings.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function setAutoEnableInstalledMods(autoEnable: boolean) {
    try {
      setError("");
      const result = await call<[boolean], { ok: boolean; error?: string; status?: unknown }>("set_auto_enable_installed_mods", autoEnable);
      if (!result.ok) setError(result.error ?? "Unable to update enable settings.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function setAutoShowFOMODInstallers(autoShow: boolean) {
    try {
      setError("");
      const result = await call<[boolean], { ok: boolean; error?: string; status?: unknown }>("set_auto_show_fomod_installers", autoShow);
      if (!result.ok) setError(result.error ?? "Unable to update FOMOD installer settings.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function setMaxConcurrentCapturedDownloads(maxDownloads: number) {
    try {
      setError("");
      const result = await call<[number], { ok: boolean; error?: string; status?: unknown }>("set_max_concurrent_captured_downloads", maxDownloads);
      if (!result.ok) setError(result.error ?? "Unable to update download settings.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function cycleMaxConcurrentCapturedDownloads() {
    const current = status?.backend?.download?.max_concurrent_captured_downloads ?? 2;
    const next = current >= 4 ? 1 : current + 1;
    void setMaxConcurrentCapturedDownloads(next);
  }

  async function setMaxConcurrentCapturedDownloadsPerGame(maxDownloads: number) {
    try {
      setError("");
      const result = await call<[number], { ok: boolean; error?: string; status?: unknown }>("set_max_concurrent_captured_downloads_per_game", maxDownloads);
      if (!result.ok) setError(result.error ?? "Unable to update per-game download settings.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  function cycleMaxConcurrentCapturedDownloadsPerGame() {
    const globalMax = status?.backend?.download?.max_concurrent_captured_downloads ?? 2;
    const current = status?.backend?.download?.max_concurrent_captured_downloads_per_game ?? 1;
    const next = current >= globalMax ? 1 : current + 1;
    void setMaxConcurrentCapturedDownloadsPerGame(next);
  }

  function openDeckyNexusBrowser() {
    if (!selectedDeckyGameID || !selectedDeckyGame || !selectedNexusDomain) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <NexusBrowserModal
        appID={selectedDeckyGameID}
        gameName={selectedDeckyGame.name}
        gameDomain={selectedNexusDomain}
        profileID={selectedProfile?.id ?? 0}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Explore Mods - Nexus", bNeverPopOut: true, bHideActionIcons: true, popupWidth: 760, popupHeight: 820 }
    );
  }

  async function registerNXM() {
    try {
      setError("");
      const result = await call<[], { ok: boolean; error?: string; status: NXMStatus }>("register_nxm_handler");
      setNXM(result.status);
      if (!result.ok) setError(result.error ?? "Unable to register NXM handler.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function testNXM() {
    try {
      setError("");
      const result = await call<[], { ok: boolean; error?: string }>("test_nxm_handler");
      if (!result.ok) setError(result.error ?? "Unable to run NXM handler.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function testNXMDispatch() {
    try {
      setError("");
      const result = await call<[], { ok: boolean; error?: string }>("test_nxm_dispatch");
      if (!result.ok) setError(result.error ?? "Unable to dispatch test NXM link.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function probeSteamBrowserNXM() {
    try {
      setError("");
      const nextStatus = status ?? await call<[], BackendStatus>("status");
      if (!status) setStatus(nextStatus);
      const port = nextStatus.port || 17942;
      const probeURL = `http://127.0.0.1:${port}/debug/nxm-probe`;
      await startSteamBrowserNXMProbe(probeURL, selectedDeckyGameID);
      setUpdateResult("Steam browser NXM probe started. Click the probe link in the browser, then check Live Logs.");
      await loadDiagnostics({ quiet: true });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      setUpdateResult(message);
      await logFrontendEvent("steam browser nxm probe threw", { error: message });
    }
  }

  async function probeDMMBrowserViewNXM() {
    try {
      setError("");
      const nextStatus = status ?? await call<[], BackendStatus>("status");
      if (!status) setStatus(nextStatus);
      const port = nextStatus.port || 17942;
      const probeURL = `http://127.0.0.1:${port}/debug/nxm-probe`;
      const opened = await openDMMBrowserViewCapture(probeURL, {
        appID: selectedDeckyGameID,
        profileID: selectedProfile?.id ?? 0,
        source: "debug-browser-view-nxm-probe",
        title: "DMM BrowserView NXM Probe"
      });
      setUpdateResult(opened ? "DMM BrowserView NXM probe opened. Click the probe link, then check Live Logs." : "DMM BrowserView NXM probe could not open. Check Live Logs.");
      await loadDiagnostics({ quiet: true });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      setUpdateResult(message);
      await logFrontendEvent("dmm browser view nxm probe threw", { error: message });
    }
  }

  async function loadDiagnostics(options: { quiet?: boolean } = {}) {
    try {
      if (!options.quiet) setError("");
      setDiagnostics(await call<[], Diagnostics>("diagnostics"));
    } catch (err) {
      if (!options.quiet) setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function installLatestUpdate() {
    try {
      setError("");
      setUpdateResult("");
      setUpdateBusy(true);
      const result = await call<[], UpdateResult>("install_latest_update");
      const message = updateResultMessage(result);
      setUpdateResult(message);
      if (!result.ok) {
        setError(result.error || message);
        await logFrontendEvent("debug update failed", { error: result.error || message, log: result.log || "" });
        await loadDiagnostics({ quiet: true });
        return;
      }
      toaster.toast({
        title: "DMM update started",
        body: "The latest package was downloaded. The Deck will reboot if installation succeeds.",
        duration: 8000,
        critical: false,
        playSound: true,
        showToast: true
      });
      await logFrontendEvent("debug update started", {
        bytes: result.bytes ?? 0,
        installer_pid: result.installer_pid ?? 0,
        log: result.log || ""
      });
      await loadDiagnostics({ quiet: true });
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
      setUpdateResult(message);
      await logFrontendEvent("debug update threw", { error: message });
      await loadDiagnostics({ quiet: true });
    } finally {
      setUpdateBusy(false);
    }
  }

  function askInstallLatestUpdate() {
    if (updateBusy) return;
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <ConfirmModal
        strTitle="Install Latest DMM"
        strDescription="DMM will download the latest GitHub release package, replace the Decky plugin through the privileged installer, and reboot the Deck if installation succeeds."
        strOKButtonText="Install Update"
        strCancelButtonText="Cancel"
        onOK={() => {
          closeModal();
          void installLatestUpdate();
        }}
        onCancel={closeModal}
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Install Latest DMM", bNeverPopOut: true }
    );
  }

  async function addCapturedInstall() {
    try {
      setError("");
      setImportResult("");
      const result = await call<[string, string, number], { ok: boolean; error?: string; result?: { job?: Job } }>(
        "add_captured_install",
        importUrl,
        selectedDeckyGameID ?? "",
        selectedProfile?.id ?? 0
      );
      if (!result.ok) {
        setError(result.error ?? "Unable to capture mod link.");
        return;
      }
      setImportUrl("");
      const job = result.result?.job;
      setImportResult(job?.message || job?.title || "Mod link captured.");
      if (job) {
        const stateKey = `${job.status}:${job.message || ""}`;
        notifiedJobStates.set(job.id, stateKey);
        await logFrontendEvent("job toast shown", { job_id: job.id, status: job.status, source: "decky-add-import" });
        showJobToast(job as Job);
      }
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function retryLaunchSetup() {
    completedLaunchActions.clear();
    launchActionAttempts.clear();
    await syncLaunchActions({ force: true, sink: setLaunchResult });
    await refresh();
  }

  useEffect(() => {
    logFrontendEvent("content mounted");
    refresh();
    return () => {
      if (routeRefreshTimer.current !== null) {
        window.clearTimeout(routeRefreshTimer.current);
        routeRefreshTimer.current = null;
      }
      logFrontendEvent("content unmounted");
    };
  }, []);

  useEffect(() => {
    if (tab === "games" && status?.running) {
      void refreshDeckyMods();
    }
  }, [tab, status?.running]);

  useEffect(() => {
    if (tab !== "games" || !selectedDeckyGameID) return;
    focusDeckyRef(selectedDeckyGameRef, "selected-game", { app_id: selectedDeckyGameID, source: "selected-game-render" });
  }, [tab, selectedDeckyGameID]);

  useEffect(() => {
    const listener = (rawEvent: Event) => {
      const event = (rawEvent as CustomEvent<DomainEvent>).detail;
      if (event?.type === "ui.changed") {
        if (isUISettings(event.payload)) applyDeckyUIPreferencesFromUI(event.payload);
        return;
      }
      if (!event) return;
      if (event.type === "game.changed") {
        scheduleDeckyRouteRefresh({ status: true, games: true, gameState: tab === "games" && Boolean(selectedDeckyGameID) });
        return;
      }
      if (event.type === "jobs.snapshot" && tab === "games" && selectedDeckyGameID && Array.isArray(event.payload)) {
        if ((event.payload as Job[]).some((job) => isJob(job) && jobMatchesAppID(job, selectedDeckyGameID))) {
          scheduleDeckyRouteRefresh({ gameState: true });
        }
        return;
      }
      if (tab !== "games" || !selectedDeckyGameID) return;
      if (event.type === "job.updated" && isJob(event.payload) && jobMatchesAppID(event.payload, selectedDeckyGameID)) {
        scheduleDeckyRouteRefresh({ gameState: true });
        return;
      }
      if (["profile_mods.changed", "deployment.changed", "install.changed", "launch.changed", "mod_updates.changed", "workshop.changed"].includes(event.type) && eventMatchesAppID(event, selectedDeckyGameID)) {
        scheduleDeckyRouteRefresh({ gameState: true });
      }
    };
    window.addEventListener(DMM_EVENT_NAME, listener);
    return () => window.removeEventListener(DMM_EVENT_NAME, listener);
  }, [tab, selectedDeckyGameID, managedGames]);

  useEffect(() => {
    const syncRunningGame = () => {
      const running = currentRunningGame();
      setRunningGame(running);
      const game = managedGames.find((item) => item.app_id === running?.app_id);
      if (!running || !game || !gameHasExtension(game) || tab !== "games" || selectedDeckyGameID) return;
      setSelectedDeckyGameID(running.app_id);
      markDeckyGameRecent(running.app_id);
      void loadDeckyGameState(running.app_id);
    };
    syncRunningGame();
    const sessions = typeof SteamClient !== "undefined" ? SteamClient?.GameSessions : undefined;
    const registration = sessions?.RegisterForAppLifetimeNotifications?.(() => syncRunningGame());
    return () => unregisterSteamCallback(registration);
  }, [tab, managedGames, selectedDeckyGameID]);

  const selectedDeckyGame = managedGames.find((game) => game.app_id === selectedDeckyGameID) ?? null;
  const selectedNexusDomain = selectedDeckyGame?.nexus_domains?.[0] ?? "";
  const selectedExploreSources = useMemo(() => {
    if (!selectedDeckyGame) return [] as DeckyExploreSource[];
    const sources: DeckyExploreSource[] = [];
    const catalogSourceTag = (catalog: CatalogStatus) => catalog.source_tag || catalog.id;
    const firstNote = (catalog: CatalogStatus) => catalog.notes?.find((note) => note.trim() !== "") ?? "";
    const nexusCatalog = catalogs.find((catalog) => catalog.id === "nexus");
    if (nexusCatalog) {
      const enabled = Boolean(selectedNexusDomain && nexusCatalog.status === "ready" && (nexusCatalog.search || nexusCatalog.browse));
      sources.push({
        id: "nexus",
        catalog: catalogSourceTag(nexusCatalog),
        title: "Nexus Mods",
        detail: enabled
          ? "Search Nexus, open the real mod page, then press Mod Manager Download."
          : selectedNexusDomain
            ? "Nexus is not ready. Configure the Nexus API key from the phone/tablet Settings screen."
            : "This extension does not declare a Nexus domain yet.",
        action: enabled ? "Explore Mods" : catalogStatusLabel(nexusCatalog.status),
        enabled,
        behavior: enabled ? "browse" : "info"
      });
    } else if (selectedDeckyGame.extension?.supported && !selectedNexusDomain) {
      sources.push({
        id: "nexus",
        catalog: "nexus",
        title: "Nexus Mods",
        detail: "This extension does not declare a Nexus domain yet.",
        action: "Unavailable",
        enabled: false,
        behavior: "info"
      });
    }
    for (const catalog of catalogs) {
      if (catalog.kind === "platform" || catalog.id === "nexus" || catalog.id === "local") continue;
      const sourceTag = catalogSourceTag(catalog);
      const readyURLImport = catalog.status === "ready" && Boolean(catalog.url_import && catalog.download);
      if (readyURLImport) {
        sources.push({
          id: catalog.id,
          catalog: sourceTag,
          title: catalog.name,
          detail: `Paste a ${catalog.name} URL in Mod Link while ${selectedDeckyGame.name} is selected. DMM will use the normal captured-install pipeline.`,
          action: "Paste URL",
          enabled: true,
          behavior: "paste"
        });
        continue;
      }
      if (catalog.status === "needs_credentials" || catalog.credentials_required) {
        sources.push({
          id: catalog.id,
          catalog: sourceTag,
          title: catalog.name,
          detail: firstNote(catalog) || "This source needs an API key before DMM can import URLs.",
          action: "Needs Key",
          enabled: false,
          behavior: "info"
        });
        continue;
      }
      if (catalog.status === "deferred") {
        sources.push({
          id: catalog.id,
          catalog: sourceTag,
          title: catalog.name,
          detail: firstNote(catalog) || "This source is deferred until a supported official automated API or client path is verified.",
          action: "Deferred",
          enabled: false,
          behavior: "info"
        });
        continue;
      }
    }
    const localCatalog = catalogs.find((catalog) => catalog.id === "local");
    if (localCatalog?.archive_upload) {
      sources.push({
        id: "local",
        catalog: catalogSourceTag(localCatalog),
        title: "Local Archive",
        detail: "Archive upload is available from the phone/tablet UI for this selected game.",
        action: "Use Phone",
        enabled: false,
        behavior: "info"
      });
    }
    const workshopCatalog = catalogs.find((catalog) => catalog.id === "steam_workshop");
    if (selectedDeckyGame.extension?.steam_workshop) {
      sources.push({
        id: "steam-workshop",
        catalog: workshopCatalog?.source_tag || "steam_workshop",
        title: workshopCatalog?.name || "Steam Workshop",
        detail: "Installed Workshop items are managed below. DMM browsing is not enabled for Workshop.",
        action: "Managed Below",
        enabled: false,
        behavior: "info"
      });
    }
    return sources;
  }, [selectedDeckyGame, selectedNexusDomain, catalogs]);
  const selectedProfile = deckyProfiles.find((item) => item.is_default) ?? deckyProfiles[0] ?? null;
  const supportedGameCount = managedGames.filter(gameHasExtension).length;
  const runningSupported = Boolean(runningGame && gameHasExtension(managedGames.find((game) => game.app_id === runningGame.app_id)));
  const favoriteGameKey = [...favoriteGameIDs].sort().join("|");
  const effectiveModSort = modOrderMode ? "profile" : modSort;
  const visibleManagedGames = useMemo(() => {
    const normalizedGameSearch = gameSearch.trim().toLowerCase();
    const favoriteIDs = new Set(favoriteGameKey ? favoriteGameKey.split("|") : []);
    return [...managedGames].filter((game) => {
      if (gameVisibility === "supported" && !gameHasExtension(game)) return false;
      if (!normalizedGameSearch) return true;
      return game.name.toLowerCase().includes(normalizedGameSearch) || game.app_id.includes(normalizedGameSearch);
    })
    .sort((a, b) => {
      const favoriteDelta = Number(favoriteIDs.has(b.app_id)) - Number(favoriteIDs.has(a.app_id));
      if (favoriteDelta !== 0) return favoriteDelta;
      const supportedDelta = Number(gameHasExtension(b)) - Number(gameHasExtension(a));
      if (supportedDelta !== 0) return supportedDelta;
      if (gameSort === "az") return a.name.localeCompare(b.name);
      if (gameSort === "za") return b.name.localeCompare(a.name);
      const recentDelta = (gameRecent[b.app_id] ?? 0) - (gameRecent[a.app_id] ?? 0);
      if (recentDelta !== 0) return recentDelta;
      return a.name.localeCompare(b.name);
    });
  }, [managedGames, gameSearch, favoriteGameKey, gameSort, gameRecent, gameVisibility]);
  const visibleDeckyMods = useMemo(() => {
    const normalizedModSearch = modSearch.trim().toLowerCase();
    const filtered = deckyMods.filter((mod) =>
      !normalizedModSearch ||
        [mod.name, mod.status, sourceForManagedMod(mod), sourceLabel(sourceForManagedMod(mod)), mod.source_game_domain, mod.source_mod_id, mod.source_file_id]
          .some((value) => String(value ?? "").toLowerCase().includes(normalizedModSearch))
      );
    return [...filtered].sort((a, b) => {
      if (effectiveModSort === "source") {
        const sourceDelta = sourceLabel(sourceForManagedMod(a)).localeCompare(sourceLabel(sourceForManagedMod(b)));
        if (sourceDelta !== 0) return sourceDelta;
      }
      if (effectiveModSort === "az") return a.name.localeCompare(b.name) || a.priority - b.priority;
      if (effectiveModSort === "za") return b.name.localeCompare(a.name) || a.priority - b.priority;
      if (effectiveModSort === "enabled") {
        const enabledDelta = Number(b.enabled) - Number(a.enabled);
        if (enabledDelta !== 0) return enabledDelta;
      }
      return a.priority - b.priority || a.name.localeCompare(b.name);
    });
  }, [deckyMods, modSearch, effectiveModSort]);
  const deckyConflictTargets = useMemo(() => deckyConflictChoiceTargets(deckyDeployPlan, deckyMods), [deckyDeployPlan, deckyMods]);
  const visibleWorkshopItems = useMemo(() => {
    const normalizedModSearch = modSearch.trim().toLowerCase();
    if (!normalizedModSearch) return deckyWorkshopItems;
    return deckyWorkshopItems.filter((item) =>
      [item.title, item.published_file_id, item.disabled_locally ? "disabled" : "enabled", item.downloaded ? "downloaded" : "pending"]
        .some((value) => String(value ?? "").toLowerCase().includes(normalizedModSearch))
    );
  }, [deckyWorkshopItems, modSearch]);
  const visibleManagedGameIDs = visibleManagedGames.map((game) => game.app_id).join("|");
  const visibleDeckyModIDs = visibleDeckyMods.map((mod) => String(mod.id)).join("|");

  useEffect(() => {
    if (selectedDeckyGameID || visibleManagedGames.length === 0) {
      if (focusedGameID && !visibleManagedGames.some((game) => game.app_id === focusedGameID)) {
        setFocusedGameID("");
      }
      return;
    }
    if (focusedGameID && !visibleManagedGames.some((game) => game.app_id === focusedGameID)) {
      setFocusedGameID("");
    }
  }, [selectedDeckyGameID, visibleManagedGameIDs, focusedGameID]);

  useEffect(() => {
    if (!selectedDeckyGameID || visibleDeckyMods.length === 0) {
      if (focusedModID !== null) setFocusedModID(null);
      return;
    }
    if (focusedModID === null || !visibleDeckyMods.some((mod) => mod.id === focusedModID)) {
      setFocusedModID(visibleDeckyMods[0].id);
    }
  }, [selectedDeckyGameID, visibleDeckyModIDs, focusedModID]);

  const mainContent = (
    <PanelSectionRow>
      <div className="dmm-sidebar-surface" style={{ ...deckySidebarSurfaceStyle, gap: "7px", paddingBottom: "88px" }}>
        <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={toggleServer} onClick={toggleServer} style={deckyCompactActionStyle("neutral")}>
          {status?.running ? "Stop Server" : "Start Server"}
        </Focusable>
        <Focusable
          className="dmm-focus-card"
          focusClassName="dmm-focus-card-focused"
          onActivate={() => {
            if (status?.running) void retryLaunchSetup();
          }}
          onClick={() => {
            if (status?.running) void retryLaunchSetup();
          }}
          style={{ ...deckyCompactActionStyle("neutral"), opacity: status?.running ? 1 : 0.5 }}
        >
          Retry Launch Setup
        </Focusable>
        <div style={{ background: "#111827", border: "1px solid #303741", borderRadius: "6px", boxSizing: "border-box", display: "grid", gap: "3px", padding: "7px", width: "100%" }}>
          <div>Status: {status?.running ? "Running" : "Stopped"}</div>
          <div>URL: {status?.url ?? "Unavailable"}</div>
          {status?.pid && <div>PID: {status.pid}</div>}
          {status?.backend && <div>Games: {status.backend.game_count} · Nexus: {status.backend.nexus.api_key_configured ? "Configured" : "Missing"}</div>}
          {launchResult && <div style={{ color: "#72e0a2", marginTop: "4px", overflowWrap: "anywhere" }}>{launchResult}</div>}
          {error && <div style={{ color: "#f87171", marginTop: "4px", overflowWrap: "anywhere" }}>{error}</div>}
          {status?.error && <div style={{ color: "#f87171", marginTop: "4px", overflowWrap: "anywhere" }}>{status.error}</div>}
        </div>
        <div style={{ background: "#111827", border: "1px solid #303741", borderRadius: "6px", boxSizing: "border-box", display: "grid", gap: "5px", padding: "7px", width: "100%" }}>
          <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800, lineHeight: 1, textTransform: "uppercase" }}>Mod Link</div>
          {selectedDeckyGame && (
            <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 800, lineHeight: 1.2, overflowWrap: "anywhere" }}>
              Target: {selectedDeckyGame.name}{selectedProfile ? ` · ${selectedProfile.name}` : ""}
            </div>
          )}
          <input
            aria-label="Mod URL"
            placeholder="Paste Nexus, nxm://, provider, or archive URL"
            style={{ ...deckyCompactInputStyle, minHeight: "34px", padding: "6px 9px" }}
            value={importUrl}
            onChange={(event) => setImportUrl(event.currentTarget.value)}
          />
          <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={addCapturedInstall} onClick={addCapturedInstall} style={{ ...deckyCompactActionStyle("neutral"), minHeight: "34px", padding: "7px 6px" }}>
            Add Mod Link
          </Focusable>
          <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, overflowWrap: "anywhere" }}>Downloads immediately; install choices appear in Action Center.</div>
          {importResult && <div style={{ color: "#72e0a2", overflowWrap: "anywhere" }}>{importResult}</div>}
        </div>
      </div>
    </PanelSectionRow>
  );

  const modsContent = (
    <>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => refreshDeckyMods()} disabled={!status?.running}>
          Refresh Mods
        </ButtonItem>
      </PanelSectionRow>
      {!status?.running && (
        <PanelSectionRow>
          <div style={{ color: "#fbbf24", overflowWrap: "anywhere" }}>Start the server before managing profile mods from Decky.</div>
        </PanelSectionRow>
      )}
      {status?.running && runningGame && (
        <PanelSectionRow>
          <div style={{ alignItems: "center", display: "flex", gap: "10px", width: "100%" }}>
            <img src={steamHeaderImage(runningGame.app_id)} style={{ borderRadius: "5px", height: "42px", objectFit: "cover", width: "74px" }} />
            <div style={{ minWidth: 0 }}>
              <div style={{ color: runningSupported ? "#72e0a2" : "#fbbf24", fontSize: "11px", fontWeight: 800, textTransform: "uppercase" }}>{runningSupported ? "Running game selected" : "Running game not supported yet"}</div>
              <div style={{ fontWeight: 800, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{runningGame.name}</div>
            </div>
          </div>
        </PanelSectionRow>
      )}
      {status?.running && !selectedDeckyGameID && (
        <PanelSectionRow>
          <div className="dmm-sidebar-surface" style={deckySidebarSurfaceStyle}>
            <div style={{ fontWeight: 800, marginBottom: "8px" }}>Select Game</div>
            <TextField label="Search Games" value={gameSearch} bShowClearAction onChange={(event) => setGameSearch(event.currentTarget.value)} />
            <ButtonItem layout="below" onClick={cycleDeckyGameSort}>
              Sort: {gameSortLabel(gameSort)}
            </ButtonItem>
            <ButtonItem layout="below" onClick={toggleDeckyGameVisibility}>
              Show: {gameVisibilityLabel(gameVisibility)}
            </ButtonItem>
            <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800, lineHeight: 1.2 }}>
              {supportedGameCount} supported · {favoriteGameIDs.size} favorite{favoriteGameIDs.size === 1 ? "" : "s"}
            </div>
            {managedGames.length === 0 && <div style={{ color: "#a1a1aa" }}>No games loaded.</div>}
            {managedGames.length > 0 && visibleManagedGames.length === 0 && <div style={{ color: "#a1a1aa" }}>No games match this search.</div>}
            <Focusable flow-children="down" navEntryPreferPosition={NavEntryPositionPreferences.FIRST} style={deckySidebarListStyle}>
              {visibleManagedGames.map((game) => {
                const focused = focusedGameID === game.app_id;
                const favorite = favoriteGameIDs.has(game.app_id);
                const badges = deckyGameCapabilityBadges(game);
                return (
                  <Focusable
                    key={game.app_id}
                    className="dmm-sidebar-surface dmm-sidebar-row"
                    data-dmm-game-id={game.app_id}
                    focusClassName="dmm-sidebar-row-focused"
                    onActivate={() => void selectDeckyGame(game.app_id)}
                    onClick={() => void selectDeckyGame(game.app_id)}
                    onOKButton={(event) => {
                      event.preventDefault();
                      event.stopPropagation();
                      void selectDeckyGame(game.app_id);
                    }}
                    onSecondaryActionDescription={favorite ? "Unfavorite" : "Favorite"}
                    onSecondaryButton={(event) => {
                      event.preventDefault();
                      event.stopPropagation();
                      toggleDeckyFavoriteGame(game.app_id);
                    }}
                    onGamepadFocus={() => setFocusedGameID(game.app_id)}
                    onFocus={() => setFocusedGameID(game.app_id)}
                    onMouseEnter={() => setFocusedGameID(game.app_id)}
                    style={{
                      ...deckyCompositeRowStyle(focused, favorite),
                      padding: "10px"
                    }}
                  >
                    <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", fontWeight: 800 }}>{game.name}</div>
                    <div style={{ display: "flex", flexWrap: "wrap", gap: "4px", minWidth: 0 }}>
                      {badges.map((badge) => (
                        <span key={`${game.app_id}-${badge.kind}`} style={deckyCapabilityPillStyle(badge.kind)}>{badge.label}</span>
                      ))}
                    </div>
                    <div style={{ color: favorite ? "#99f6e4" : "#a1a1aa", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>
                      {favorite ? "Favorite" : `App ${game.app_id}`} · A Select · Y {favorite ? "Unfavorite" : "Favorite"}
                    </div>
                  </Focusable>
                );
              })}
            </Focusable>
          </div>
        </PanelSectionRow>
      )}
          {status?.running && selectedDeckyGameID && (
        <>
          <PanelSectionRow>
	            <Focusable
	              ref={selectedDeckyGameRef}
	              className="dmm-sidebar-surface dmm-sidebar-row"
	              data-dmm-selected-game-primary="true"
	              focusClassName="dmm-sidebar-row-focused"
	              onCancelActionDescription="Change Game"
	              onCancelButton={(event) => {
	                event.preventDefault();
	                event.stopPropagation();
                clearSelectedDeckyGame();
              }}
              preferredFocus
              style={{
                ...deckyCompositeRowStyle(false, true),
                alignItems: "center",
                display: "grid",
                gap: "10px",
                gridTemplateColumns: "74px minmax(0, 1fr)",
                padding: "10px"
              }}
            >
              <img src={steamHeaderImage(selectedDeckyGameID)} style={{ borderRadius: "5px", height: "42px", objectFit: "cover", width: "74px" }} />
              <div style={{ minWidth: 0 }}>
	                <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800, textTransform: "uppercase" }}>{selectedProfile ? `Profile: ${selectedProfile.name}` : "No profile"}</div>
	                <div style={{ fontWeight: 800, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{selectedDeckyGame?.name ?? selectedDeckyGameID}</div>
	                <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, marginTop: "4px", overflowWrap: "anywhere" }}>
	                  B Change Game
	                </div>
	              </div>
	            </Focusable>
	          </PanelSectionRow>
	          <PanelSectionRow>
	            <div className="dmm-sidebar-surface" style={{ ...deckySidebarSurfaceStyle, gap: "8px" }}>
	              <div style={{ alignItems: "center", display: "flex", justifyContent: "space-between", minWidth: 0 }}>
	                <div style={{ color: "#f8fafc", fontWeight: 900 }}>Explore Mods</div>
	                <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>{selectedExploreSources.filter((source) => source.enabled).length} ready</div>
	              </div>
	              <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.25, overflowWrap: "anywhere" }}>
	                Choose a source for {selectedDeckyGame?.name ?? "this game"}. Downloads are captured through the provider's Mod Manager Download flow.
	              </div>
	              {selectedExploreSources.length === 0 && (
	                <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>
	                  {selectedDeckyGame?.extension?.supported
	                    ? "This game's DMM extension does not expose a browsable mod source yet."
	                    : "This game does not have a DMM extension yet."}
	                </div>
	              )}
	              {selectedExploreSources.map((source) => (
	                <Focusable
	                  key={source.id}
	                  className="dmm-sidebar-row"
	                  focusClassName="dmm-sidebar-row-focused"
	                  onActivate={(event) => {
	                    event.preventDefault();
	                    event.stopPropagation();
	                    if (!source.enabled) return;
	                    if (source.behavior === "browse") openDeckyNexusBrowser();
	                    if (source.behavior === "paste") {
	                      setTab("main");
	                      setImportResult(`Paste a ${source.title} URL in Mod Link for ${selectedDeckyGame?.name ?? "this game"}.`);
	                      void logFrontendEvent("decky source paste selected", { app_id: selectedDeckyGameID, source: source.id });
	                    }
	                  }}
	                  onClick={() => {
	                    if (!source.enabled) return;
	                    if (source.behavior === "browse") openDeckyNexusBrowser();
	                    if (source.behavior === "paste") {
	                      setTab("main");
	                      setImportResult(`Paste a ${source.title} URL in Mod Link for ${selectedDeckyGame?.name ?? "this game"}.`);
	                      void logFrontendEvent("decky source paste selected", { app_id: selectedDeckyGameID, source: source.id });
	                    }
	                  }}
	                  style={{
	                    ...deckyCompositeRowStyle(false, source.enabled),
	                    opacity: source.enabled ? 1 : 0.58,
	                    padding: "10px"
	                  }}
	                >
	                  <div style={{ alignItems: "flex-start", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
	                    <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", flex: "1 1 120px", fontWeight: 900 }}>{source.title}</div>
	                    <span style={deckySourcePillStyle(source.catalog)}>{sourceLabel(source.catalog)}</span>
	                  </div>
	                  <div style={{ color: "#d4d4d8", fontSize: "11px", lineHeight: 1.25, overflowWrap: "anywhere" }}>{source.detail}</div>
	                  <div style={{ color: source.enabled ? "#99f6e4" : "#a1a1aa", fontSize: "11px", fontWeight: 900, lineHeight: 1.25, overflowWrap: "anywhere" }}>
	                    {source.enabled ? `A ${source.action}` : source.action}
	                  </div>
	                </Focusable>
	              ))}
	            </div>
	          </PanelSectionRow>
	          {(deckyMods.length > 0 || deckyWorkshopItems.length > 0) && (
	            <PanelSectionRow>
	              <div className="dmm-sidebar-surface" style={deckySidebarSurfaceStyle}>
                <TextField label="Search Mods" value={modSearch} bShowClearAction onChange={(event) => setModSearch(event.currentTarget.value)} />
                <ButtonItem layout="below" onClick={cycleDeckyModSort}>
                  Sort: {deckyModSortLabel(effectiveModSort)}
                </ButtonItem>
                <ButtonItem layout="below" onClick={toggleDeckyModOrderMode}>
                  Order Mode: {modOrderMode ? "On" : "Off"}
                </ButtonItem>
                <ButtonItem layout="below" onClick={checkDeckyModUpdates} disabled={modUpdateBusy || deckyMods.length === 0}>
                  {modUpdateBusy ? "Checking Updates..." : "Check Updates"}
                </ButtonItem>
              </div>
            </PanelSectionRow>
          )}
          <PanelSectionRow>
            <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Toggling a mod applies the selected profile. Restart a running game to pick up changes.</div>
          </PanelSectionRow>
          {deckyDeploymentStatus?.restore_available && (
            <PanelSectionRow>
              <Focusable
                className="dmm-sidebar-surface dmm-sidebar-row"
                focusClassName="dmm-sidebar-row-focused"
                onActivate={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  askRestoreDeckyDeployment();
                }}
                onClick={askRestoreDeckyDeployment}
                style={{
                  ...deckyCompositeRowStyle(false),
                  padding: "10px"
                }}
              >
                <div style={{ alignItems: "center", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
                  <span
                    style={{
                      background: "#312e81",
                      border: "1px solid #818cf8",
                      borderRadius: "999px",
                      color: "#e0e7ff",
                      fontSize: "11px",
                      fontWeight: 800,
                      lineHeight: 1,
                      padding: "5px 8px"
                    }}
                  >
                    Recovery
                  </span>
                  <span style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                    {deckyDeploymentStatus.file_count} managed file{deckyDeploymentStatus.file_count === 1 ? "" : "s"} · {deckyDeploymentStatus.strategy || "managed"}
                  </span>
                </div>
                <div style={{ color: "#d4d4d8", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                  {deckyDeploymentStatus.restore_summary || "Restore the last DMM-applied state if deployed files drift."}
                </div>
                <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>
                  A Restore Last DMM State
                </div>
              </Focusable>
            </PanelSectionRow>
          )}
	          <PanelSectionRow>
	            <ButtonItem layout="below" onClick={clearSelectedDeckyGame}>
	              Change Game
	            </ButtonItem>
	          </PanelSectionRow>
	          {deckyInstallCandidates.length > 0 && (
            <PanelSectionRow>
              <div className="dmm-sidebar-surface" style={deckySidebarListStyle}>
                <div style={{ alignItems: "center", display: "flex", justifyContent: "space-between", minWidth: 0 }}>
                  <div style={{ fontWeight: 800 }}>Action Center</div>
                  <div style={{ color: "#fbbf24", fontSize: "11px", fontWeight: 800 }}>{deckyInstallCandidates.length} item{deckyInstallCandidates.length === 1 ? "" : "s"}</div>
                </div>
                {deckyInstallCandidates.map((candidate) => {
                  const focused = focusedCandidateID === candidate.id;
                  const installer = installerForCandidate(candidate);
                  return (
                    <Focusable
                      key={candidate.id}
                      className="dmm-sidebar-row"
                      focusClassName="dmm-sidebar-row-focused"
                      onActivate={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (installer) openDeckyInstallerChoice(candidate);
                      }}
                      onClick={() => {
                        if (installer) openDeckyInstallerChoice(candidate);
                      }}
                      onSecondaryActionDescription="Clear Items"
                      onSecondaryButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        askClearDeckyInstallCandidates();
                      }}
                      onGamepadFocus={() => setFocusedCandidateID(candidate.id)}
                      onFocus={() => setFocusedCandidateID(candidate.id)}
                      onMouseEnter={() => setFocusedCandidateID(candidate.id)}
                      preferredFocus={focused}
                      style={{
                        ...deckyCompositeRowStyle(focused),
                        padding: "10px"
                      }}
                    >
                      <div style={{ alignItems: "flex-start", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
                        <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", flex: "1 1 120px", fontWeight: 800 }}>{candidate.name}</div>
                        <span style={deckySourcePillStyle(sourceForInstallCandidate(candidate))}>{sourceLabel(sourceForInstallCandidate(candidate))}</span>
                      </div>
                      <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                        {candidate.status === "blocked" ? "Blocked installer" : "Installer choices"} · {candidate.source_game_domain}/mods/{candidate.source_mod_id}/files/{candidate.source_file_id}
                      </div>
                      {candidate.reason && (
                        <div style={{ color: candidate.status === "blocked" ? "#fca5a5" : "#d4d4d8", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                          {candidate.reason}
                        </div>
                      )}
                      <div style={{ color: installer ? "#99f6e4" : "#fca5a5", fontSize: "11px", fontWeight: 800, lineHeight: 1.25 }}>
                        {installer ? "A Open Choices" : "Review on phone/tablet"} · Y Clear Items
                      </div>
                    </Focusable>
                  );
                })}
              </div>
            </PanelSectionRow>
          )}
          {deckyProfiles.length > 1 && (
            <PanelSectionRow>
              <div className="dmm-sidebar-surface" style={deckySidebarListStyle}>
                <div style={{ fontWeight: 800, marginBottom: "8px" }}>Profile</div>
                {deckyProfiles.map((profile) => (
                  <Focusable
                    className="dmm-focus-card"
                    focusClassName="dmm-focus-card-focused"
                    key={profile.id}
                    onActivate={() => selectDeckyProfile(profile)}
                    onClick={() => selectDeckyProfile(profile)}
                    onGamepadFocus={() => setFocusedProfileID(profile.id)}
                    onFocus={() => setFocusedProfileID(profile.id)}
                    onMouseEnter={() => setFocusedProfileID(profile.id)}
                    style={{
                      ...deckyFocusableCardStyle(focusedProfileID === profile.id, profile.is_default),
                      fontWeight: 800,
                      marginBottom: "8px",
                      padding: "10px",
                    }}
                  >
                    <div style={{ minWidth: 0 }}>
                      <div style={deckyTwoLineTextStyle}>{profile.name}</div>
                      <div style={{ color: profile.is_default ? "#d1fae5" : "#a1a1aa", fontSize: "11px", lineHeight: 1.2 }}>
                        {profile.is_default ? "Active - " : ""}{profileCountText(profile)}
                      </div>
                    </div>
                  </Focusable>
                ))}
              </div>
            </PanelSectionRow>
          )}
          {deckyMods.length === 0 && deckyWorkshopItems.length === 0 && (
            <PanelSectionRow>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>No profile mods yet. Add a mod link from Decky or the phone/tablet UI.</div>
            </PanelSectionRow>
          )}
          {deckyMods.length > 0 && visibleDeckyMods.length === 0 && (
            <PanelSectionRow>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>No mods match this search.</div>
            </PanelSectionRow>
          )}
          {visibleDeckyMods.length > 0 && (
            <PanelSectionRow>
              <Focusable flow-children="down" navEntryPreferPosition={NavEntryPositionPreferences.FIRST} style={deckySidebarListStyle}>
                {visibleDeckyMods.map((mod, index) => {
                  const focused = focusedModID === mod.id;
                  const updateAvailable = mod.update?.status === "available";
                  return (
                    <Focusable
                      key={mod.id}
                      className="dmm-sidebar-surface dmm-sidebar-row"
                      data-dmm-mod-id={String(mod.id)}
                      focusClassName="dmm-sidebar-row-focused"
                      onActivate={() => {
                        if (modOrderMode) void moveDeckyModInProfile(mod, -1);
                        else void toggleDeckyMod(mod, !mod.enabled);
                      }}
                      onClick={() => {
                        if (modOrderMode) void moveDeckyModInProfile(mod, -1);
                        else void toggleDeckyMod(mod, !mod.enabled);
                      }}
                      onOKButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) void moveDeckyModInProfile(mod, -1);
                        else void toggleDeckyMod(mod, !mod.enabled);
                      }}
                      onSecondaryActionDescription={modOrderMode ? "Move Down" : updateAvailable ? "Install Update" : "Reinstall"}
                      onSecondaryButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) void moveDeckyModInProfile(mod, 1);
                        else if (updateAvailable) void updateDeckyMod(mod);
                        else void reinstallDeckyMod(mod);
                      }}
                      onOptionsActionDescription={modOrderMode ? "Done Ordering" : "Remove"}
                      onOptionsButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) setModOrderMode(false);
                        else askRemoveDeckyMod(mod);
                      }}
                      onMenuActionDescription={modOrderMode ? "Done Ordering" : updateAvailable ? "Reinstall" : "Remove"}
                      onMenuButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) setModOrderMode(false);
                        else if (updateAvailable) void reinstallDeckyMod(mod);
                        else askRemoveDeckyMod(mod);
                      }}
                      onGamepadFocus={() => {
                        setFocusedModID(mod.id);
                      }}
                      onFocus={() => {
                        setFocusedModID(mod.id);
                      }}
                      onMouseEnter={() => {
                        setFocusedModID(mod.id);
                      }}
                      preferredFocus={focused || (index === 0 && focusedModID === null)}
                      style={{
                        ...deckyCompositeRowStyle(focused, mod.enabled),
                        opacity: busyModID === mod.id ? 0.65 : 1,
                        padding: "10px"
                      }}
                    >
                      <div style={{ alignItems: "flex-start", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
                        <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", flex: "1 1 120px", fontWeight: 800 }}>{mod.name}</div>
                        <span style={deckySourcePillStyle(sourceForManagedMod(mod))}>{sourceLabel(sourceForManagedMod(mod))}</span>
                      </div>
                      <div style={{ alignItems: "center", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
                        <span
                          style={{
                            background: mod.enabled ? "#0f766e" : "#3f3f46",
                            border: `1px solid ${mod.enabled ? "#5eead4" : "#52525b"}`,
                            borderRadius: "999px",
                            color: "#f8fafc",
                            fontSize: "11px",
                            fontWeight: 800,
                            lineHeight: 1,
                            maxWidth: "100%",
                            overflow: "hidden",
                            padding: "5px 8px",
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap"
                          }}
                        >
                          {mod.enabled ? "Enabled" : "Disabled"}
                        </span>
                        <span style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                          Priority {mod.priority} · {deckyModStateLabel(mod)}
                        </span>
                      </div>
                      <div style={{ color: mod.update?.status === "available" ? "#99f6e4" : mod.update?.status === "error" ? "#fca5a5" : "#a1a1aa", fontSize: "11px", fontWeight: 700, lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                        {deckyModUpdateLabel(mod.update)} · {deckyModUpdateDetail(mod.update)}
                      </div>
                      <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>
                        {modOrderMode ? "A Move Up · Y Move Down · Options Done" : updateAvailable ? `A ${mod.enabled ? "Disable" : "Enable"} · Y Update · Options Remove · Menu Reinstall` : `A ${mod.enabled ? "Disable" : "Enable"} · Y Reinstall · Options Remove`}
                      </div>
                    </Focusable>
                  );
                })}
              </Focusable>
            </PanelSectionRow>
          )}
          {deckyLoadOrder?.supported && (
            <PanelSectionRow>
              <div className="dmm-sidebar-surface" style={{ ...deckySidebarSurfaceStyle, background: "#111827", border: "1px solid #303741", borderRadius: "6px", padding: "10px" }}>
                <div style={{ alignItems: "center", display: "flex", justifyContent: "space-between", minWidth: 0 }}>
                  <div style={{ fontWeight: 800 }}>Plugin Load Order</div>
                  <div style={{ color: "#7dd3fc", fontSize: "11px", fontWeight: 800 }}>{deckyLoadOrder.plugins.length} plugin{deckyLoadOrder.plugins.length === 1 ? "" : "s"}</div>
                </div>
                <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, overflowWrap: "anywhere" }}>
                  {deckyLoadOrder.name || "Extension activation"} · {deckyLoadOrder.plugins_file || "plugins.txt"} / {deckyLoadOrder.load_order_file || "loadorder.txt"}
                </div>
                {deckyLoadOrder.warnings?.map((warning) => (
                  <div key={warning} style={{ color: "#fbbf24", fontSize: "11px", lineHeight: 1.2, overflowWrap: "anywhere" }}>{warning}</div>
                ))}
                {deckyLoadOrder.plugins.length === 0 ? (
                  <div style={{ color: "#a1a1aa", fontSize: "11px" }}>No active plugin files are in this profile.</div>
                ) : (
                  <div style={{ display: "grid", gap: "6px", width: "100%" }}>
                    {deckyLoadOrder.plugins.slice(0, 8).map((plugin, index) => (
                      <div key={`${plugin.source}:${plugin.name}:${index}`} style={{ alignItems: "start", background: "#0b1220", border: "1px solid #263243", borderRadius: "6px", display: "grid", gap: "8px", gridTemplateColumns: "24px minmax(0, 1fr)", padding: "7px" }}>
                        <div style={{ color: "#7dd3fc", fontSize: "11px", fontWeight: 900, textAlign: "center" }}>{index + 1}</div>
                        <div style={{ minWidth: 0 }}>
                          <div style={{ alignItems: "flex-start", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
                            <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", flex: "1 1 120px", fontWeight: 800 }}>{plugin.name}</div>
                            <span style={deckySourcePillStyle(plugin.catalog || plugin.source)}>{sourceLabel(plugin.catalog || plugin.source)}</span>
                          </div>
                          <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, overflowWrap: "anywhere" }}>
                            {plugin.source === "native" ? "Native" : `DMM mod ${plugin.mod_id || plugin.installed_mod_id || ""}`} · Priority {plugin.priority}
                          </div>
                        </div>
                      </div>
                    ))}
                    {deckyLoadOrder.plugins.length > 8 && (
                      <div style={{ color: "#a1a1aa", fontSize: "11px" }}>{deckyLoadOrder.plugins.length - 8} more plugin{deckyLoadOrder.plugins.length - 8 === 1 ? "" : "s"} in phone/tablet advanced view.</div>
                    )}
                  </div>
                )}
              </div>
            </PanelSectionRow>
          )}
          {deckyConflictTargets.length > 0 && (
            <PanelSectionRow>
              <div className="dmm-sidebar-surface" style={deckySidebarListStyle}>
                <div style={{ alignItems: "center", display: "flex", justifyContent: "space-between", marginBottom: "8px", minWidth: 0 }}>
                  <div style={{ fontWeight: 800 }}>File Winners</div>
                  <div style={{ color: "#7dd3fc", fontSize: "11px", fontWeight: 800 }}>{deckyConflictTargets.length} target{deckyConflictTargets.length === 1 ? "" : "s"}</div>
                </div>
                {deckyConflictTargets.map((target) => {
                  const focused = focusedConflictTarget === target.target_path;
                  const next = nextDeckyConflictCandidate(target);
                  const busy = busyConflictTarget === target.target_path;
                  const current = target.candidates.find((candidate) => candidate.current);
                  return (
                    <Focusable
                      key={target.target_path}
                      className="dmm-sidebar-row"
                      focusClassName="dmm-sidebar-row-focused"
                      onActivate={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (next) void setDeckyFileConflictWinner(target, next.id);
                      }}
                      onClick={() => {
                        if (next) void setDeckyFileConflictWinner(target, next.id);
                      }}
                      onSecondaryActionDescription="Use Profile Order"
                      onSecondaryButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        void clearDeckyFileConflictWinner(target);
                      }}
                      onGamepadFocus={() => setFocusedConflictTarget(target.target_path)}
                      onFocus={() => setFocusedConflictTarget(target.target_path)}
                      onMouseEnter={() => setFocusedConflictTarget(target.target_path)}
                      style={{
                        ...deckyCompositeRowStyle(focused),
                        opacity: busy ? 0.65 : 1,
                        padding: "10px"
                      }}
                    >
                      <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", fontWeight: 800 }}>{target.target_relative}</div>
                      <div style={{ alignItems: "center", color: "#a1a1aa", display: "flex", flexWrap: "wrap", fontSize: "11px", gap: "6px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                        <span>Current: {target.current_winner_name}</span>
                        {current?.catalog && <span style={deckySourcePillStyle(current.catalog)}>{sourceLabel(current.catalog)}</span>}
                      </div>
                      <div style={{ display: "grid", gap: "4px", minWidth: 0 }}>
                        {target.candidates.map((candidate) => (
                          <div key={candidate.id} style={{ alignItems: "center", color: "#d4d4d8", display: "flex", flexWrap: "wrap", fontSize: "11px", gap: "6px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                            <span>{candidate.current ? "Using" : "Option"} {candidate.name}</span>
                            {candidate.catalog && <span style={deckySourcePillStyle(candidate.catalog)}>{sourceLabel(candidate.catalog)}</span>}
                          </div>
                        ))}
                      </div>
                      <div style={{ color: next ? "#99f6e4" : "#a1a1aa", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>
                        {next ? `A Use ${next.name}` : "Only one winner available"} · Y Use Profile Order
                      </div>
                    </Focusable>
                  );
                })}
              </div>
            </PanelSectionRow>
          )}
          {deckyWorkshopItems.length > 0 && visibleWorkshopItems.length === 0 && (
            <PanelSectionRow>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>No Steam Workshop items match this search.</div>
            </PanelSectionRow>
          )}
          {visibleWorkshopItems.length > 0 && (
            <PanelSectionRow>
              <div className="dmm-sidebar-surface" style={deckySidebarListStyle}>
                <div style={{ alignItems: "center", display: "flex", justifyContent: "space-between", marginBottom: "8px", minWidth: 0 }}>
                  <div style={{ fontWeight: 800 }}>Steam Workshop</div>
                  <div style={{ color: deckyWorkshopSupported ? "#72e0a2" : "#fbbf24", fontSize: "11px", fontWeight: 800 }}>{visibleWorkshopItems.length} item{visibleWorkshopItems.length === 1 ? "" : "s"}</div>
                </div>
                {!deckyWorkshopSupported && (
                  <div style={{ color: "#fbbf24", fontSize: "11px", marginBottom: "8px", overflowWrap: "anywhere" }}>This extension can coexist with Workshop content, but management actions are not enabled for this game.</div>
                )}
                {visibleWorkshopItems.map((item) => {
                  const disabled = item.disabled_known && item.disabled_locally;
                  const toggleKind = disabled ? "enable" : "disable";
                  const toggleKey = `${item.published_file_id}:${toggleKind}`;
                  const unsubscribeKey = `${item.published_file_id}:unsubscribe`;
                  const orderKey = `${item.published_file_id}:order`;
                  const busy = busyWorkshopKey === toggleKey || busyWorkshopKey === unsubscribeKey || busyWorkshopKey === orderKey;
                  const toggleSupported = deckyWorkshopSupported && item.disabled_known;
                  return (
                    <Focusable
                      key={item.published_file_id}
                      className="dmm-sidebar-row"
                      focusClassName="dmm-sidebar-row-focused"
                      onActivate={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) void moveDeckyWorkshopItem(item, -1);
                        else if (toggleSupported) void queueDeckyWorkshopAction(item, toggleKind);
                      }}
                      onClick={() => {
                        if (modOrderMode) void moveDeckyWorkshopItem(item, -1);
                        else if (toggleSupported) void queueDeckyWorkshopAction(item, toggleKind);
                      }}
                      onSecondaryActionDescription={modOrderMode ? "Move Down" : "Unsubscribe"}
                      onSecondaryButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) void moveDeckyWorkshopItem(item, 1);
                        else askUnsubscribeWorkshopItem(item);
                      }}
                      onOptionsActionDescription={modOrderMode ? "Done Ordering" : "Unsubscribe"}
                      onOptionsButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) setModOrderMode(false);
                        else askUnsubscribeWorkshopItem(item);
                      }}
                      style={{
                        ...deckyCompositeRowStyle(false, !disabled),
                        opacity: busy ? 0.65 : 1,
                        padding: "10px"
                      }}
                    >
                      <div style={{ alignItems: "flex-start", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
                        <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", flex: "1 1 120px", fontWeight: 800 }}>{workshopItemName(item)}</div>
                        <span style={deckySourcePillStyle(item.source_tag || item.catalog || "steam_workshop")}>{sourceLabel(item.source_tag || item.catalog || "steam_workshop")}</span>
                      </div>
                      <div style={{ alignItems: "center", display: "flex", flexWrap: "wrap", gap: "6px", minWidth: 0 }}>
                        <span
                          style={{
                            background: disabled ? "#3f3f46" : "#0f766e",
                            border: `1px solid ${disabled ? "#52525b" : "#5eead4"}`,
                            borderRadius: "999px",
                            color: "#f8fafc",
                            fontSize: "11px",
                            fontWeight: 800,
                            lineHeight: 1,
                            padding: "5px 8px"
                          }}
                        >
                          {item.disabled_known ? (disabled ? "Disabled" : "Enabled") : "Steam managed"}
                        </span>
                        <span style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                          {item.downloaded ? "Downloaded" : "Subscribed"} · {item.published_file_id}
                        </span>
                      </div>
                      <div style={{ color: "#99f6e4", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, overflowWrap: "anywhere" }}>
                        {modOrderMode
                          ? `A Move Up · Y Move Down · Options Done`
                          : `A ${busyWorkshopKey === toggleKey ? "Queueing" : toggleSupported ? (disabled ? "Enable" : "Disable") : "Sync Needed"} · Y ${busyWorkshopKey === unsubscribeKey ? "Queueing" : "Unsubscribe"}`}
                      </div>
                    </Focusable>
                  );
                })}
              </div>
            </PanelSectionRow>
          )}
          {modsResult && (
            <PanelSectionRow>
              <div style={{ color: "#72e0a2", overflowWrap: "anywhere" }}>{modsResult}</div>
            </PanelSectionRow>
          )}
          {error && (
            <PanelSectionRow>
              <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{error}</div>
            </PanelSectionRow>
          )}
        </>
      )}
    </>
  );

  const settingsContent = (
    <>
      <PanelSectionRow>
        <div>
          <div style={{ fontWeight: 800, marginBottom: "6px" }}>Server Access</div>
          <div>LAN only: {status?.backend?.lan_only ? "Enabled" : "Disabled"}</div>
          <div>Captured installs: {status?.backend?.install.auto_install_captured_downloads ? "Install automatically" : "Manual install"}</div>
          <div>New mod state: {status?.backend?.install.auto_enable_installed_mods ? "Enable automatically" : "Install disabled"}</div>
          <div>FOMOD installers: {status?.backend?.install.auto_show_fomod_installers ? "Auto display" : "Action Center"}</div>
          <div>Downloads: {status?.backend?.download?.active_captured_downloads ?? 0}/{status?.backend?.download?.max_concurrent_captured_downloads ?? 2} active</div>
          <div>NXM handler: {nxm?.registered ? "Registered" : "Not registered"}</div>
        </div>
      </PanelSectionRow>
      <PanelSectionRow>
        <ToggleField
          label="Auto-install captured downloads"
          description="NXM links always download immediately. This installs the cached archive without asking first."
          checked={status?.backend?.install.auto_install_captured_downloads ?? false}
          disabled={!status?.running}
          onChange={setAutoInstallCapturedDownloads}
        />
      </PanelSectionRow>
      <PanelSectionRow>
        <ToggleField
          label="Auto-enable installed mods"
          description="New installs are enabled and deployed automatically when no choices or conflicts block them."
          checked={status?.backend?.install.auto_enable_installed_mods ?? false}
          disabled={!status?.running}
          onChange={setAutoEnableInstalledMods}
        />
      </PanelSectionRow>
      <PanelSectionRow>
        <ToggleField
          label="Auto-display FOMOD installers"
          description="When installer choices are required, DMM opens a Decky modal automatically. Closing it leaves the item in Action Center."
          checked={status?.backend?.install.auto_show_fomod_installers ?? true}
          disabled={!status?.running}
          onChange={setAutoShowFOMODInstallers}
        />
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={cycleMaxConcurrentCapturedDownloads} disabled={!status?.running}>
          Concurrent Downloads: {status?.backend?.download?.max_concurrent_captured_downloads ?? 2}
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={cycleMaxConcurrentCapturedDownloadsPerGame} disabled={!status?.running}>
          Per-Game Downloads: {status?.backend?.download?.max_concurrent_captured_downloads_per_game ?? 1}
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setLanOnly(true)}>
          Enable LAN Only
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setLanOnly(false)}>
          Allow Trusted Tunnel
        </ButtonItem>
      </PanelSectionRow>
    </>
  );

  const debugContent = (
    <>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={refresh}>
          Refresh
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={askInstallLatestUpdate} disabled={updateBusy}>
          {updateBusy ? "Installing Update..." : "Install Latest Update"}
        </ButtonItem>
      </PanelSectionRow>
      {updateResult && (
        <PanelSectionRow>
          <div style={{ color: error ? "#fbbf24" : "#72e0a2", overflowWrap: "anywhere", whiteSpace: "pre-wrap" }}>{updateResult}</div>
        </PanelSectionRow>
      )}
      <PanelSectionRow>
        <div style={{ display: "grid", gap: "4px", paddingRight: "4px", width: "100%" }}>
          <div style={{ fontWeight: 800 }}>Build</div>
          <div>Channel: {status?.build?.channel || "Unknown"}</div>
          <div>Commit: {status?.build?.short_commit || status?.build?.commit?.slice(0, 12) || "Unknown"}</div>
          <div>Built: {status?.build?.built_at || "Unknown"}</div>
          <div
            style={{
              alignItems: "center",
              background: "#052e2b",
              border: "1px solid #14b8a6",
              borderRadius: "6px",
              color: "#ccfbf1",
              display: "flex",
              gap: "8px",
              justifyContent: "space-between",
              marginTop: "4px",
              padding: "6px 8px"
            }}
          >
            <span style={{ fontWeight: 900 }}>Build Fingerprint</span>
            <span style={{ fontFamily: "monospace", fontWeight: 800 }}>{status?.build?.short_commit || "unknown"}</span>
          </div>
          {status?.build?.version && <div>Plugin: {status.build.version}</div>}
          {status?.build?.error && <div style={{ color: "#fbbf24", overflowWrap: "anywhere" }}>{status.build.error}</div>}
        </div>
      </PanelSectionRow>
      <PanelSectionRow>
        <div style={{ display: "grid", gap: "8px", width: "100%" }}>
          <div style={{ alignItems: "center", display: "flex", justifyContent: "space-between", gap: "8px" }}>
            <strong>Live Logs</strong>
            <span style={{ color: "#72e0a2", fontSize: "11px", fontWeight: 800 }}>Auto</span>
          </div>
          <pre
            style={{
              background: "#020617",
              border: "1px solid #334155",
              borderRadius: "6px",
              color: "#d4d4d8",
              fontFamily: "monospace",
              fontSize: "10px",
              lineHeight: 1.35,
              margin: 0,
              maxHeight: "420px",
              overflow: "auto",
              padding: "10px",
              whiteSpace: "pre-wrap",
              width: "100%"
            }}
          >
            {diagnosticLogText}
          </pre>
        </div>
      </PanelSectionRow>
      <PanelSectionRow>
        <div style={{ paddingRight: "4px", width: "100%" }}>
          <div style={{ fontWeight: 800, marginBottom: "8px" }}>Dependencies</div>
          {dependencies.map((dep) => (
            <div key={dep.command} style={{ marginBottom: "10px", borderBottom: "1px solid #303741", paddingBottom: "8px" }}>
              <div style={{ color: dep.installed ? "#72e0a2" : "#f87171", fontWeight: 800 }}>
                {dep.name}: {dep.installed ? "Installed" : "Missing"}
              </div>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere", lineHeight: 1.25 }}>{dep.path ?? dep.command}</div>
            </div>
          ))}
        </div>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={registerNXM}>
          Register NXM Handler
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={testNXM}>
          Test Handler Direct
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={testNXMDispatch}>
          Test NXM Dispatch
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={probeSteamBrowserNXM}>
          Probe Steam Browser NXM
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={probeDMMBrowserViewNXM}>
          Probe DMM BrowserView NXM
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <div>
          <div>Registered: {nxm?.registered ? "Yes" : "No"}</div>
          <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Current: {nxm?.current_handler || "None"}</div>
          <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Protocol: {nxm?.protocol_handler || "None"}</div>
          <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>xdg-mime: {nxm?.xdg_handler || "Unknown"}</div>
          <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>File: {nxm?.desktop_path || "Unknown"}</div>
          {status?.logs && (
            <>
              <div style={{ color: "#a1a1aa", marginTop: "8px", overflowWrap: "anywhere" }}>Plugin log: {status.logs.plugin}</div>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Backend log: {status.logs.backend}</div>
            </>
          )}
        </div>
      </PanelSectionRow>
    </>
  );

  const tabItems: { id: Tab; title: string; content: ReactNode }[] = [
    { id: "main", title: deckyTabLabels.main, content: mainContent },
    { id: "games", title: deckyTabLabels.games, content: modsContent },
    { id: "settings", title: deckyTabLabels.settings, content: settingsContent },
    { id: "debug", title: deckyTabLabels.debug, content: debugContent }
  ];
  const activeTabItem = tabItems.find((item) => item.id === tab) ?? tabItems[0];

  function showDeckyRouteTab(next: Tab, source: string) {
    if (next === tab) return;
    setTab(next);
    void logFrontendEvent("decky route tab changed", { tab: next, source });
  }

  function cycleDeckyRouteTab(delta: -1 | 1, source: string) {
    const currentIndex = Math.max(0, deckyTabOrder.indexOf(tab));
    const nextIndex = (currentIndex + delta + deckyTabOrder.length) % deckyTabOrder.length;
    showDeckyRouteTab(deckyTabOrder[nextIndex], source);
  }

  function handleDeckyRouteButtonDown(event: GamepadEvent) {
    if (event.detail.button === GamepadButton.CANCEL && tab === "games" && selectedDeckyGameID) {
      event.preventDefault();
      event.stopPropagation();
      clearSelectedDeckyGame();
      void logFrontendEvent("decky selected game cleared", { source: "route-cancel" });
      return;
    }
    if (event.detail.button === GamepadButton.BUMPER_LEFT) {
      event.preventDefault();
      event.stopPropagation();
      cycleDeckyRouteTab(-1, "bumper-left");
      return;
    }
    if (event.detail.button === GamepadButton.BUMPER_RIGHT) {
      event.preventDefault();
      event.stopPropagation();
      cycleDeckyRouteTab(1, "bumper-right");
    }
  }

  return (
    <Focusable flow-children="down" onButtonDown={handleDeckyRouteButtonDown} style={deckyRouteShellStyle}>
      <style>{deckyRuntimeStyles}</style>
      <div style={deckyRouteContentStyle}>
        <Focusable flow-children="right" navEntryPreferPosition={NavEntryPositionPreferences.FIRST} style={deckyRouteTabBarStyle}>
          {tabItems.map((item) => {
            const active = item.id === tab;
            return (
              <Focusable
                key={item.id}
                className="dmm-focus-card"
                focusClassName="dmm-focus-card-focused"
                onActivate={() => showDeckyRouteTab(item.id, "tab-button")}
                onClick={() => showDeckyRouteTab(item.id, "tab-click")}
                onOKButton={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  showDeckyRouteTab(item.id, "tab-ok");
                }}
                preferredFocus={active}
                style={deckyRouteTabButtonStyle(active)}
              >
                {item.title}
              </Focusable>
            );
          })}
        </Focusable>
        {deckyTabBody(
          activeTabItem.id,
          activeTabItem.content,
          activeTabItem.id === "games" ? selectedDeckyGameID || "game-list" : activeTabItem.id,
          activeTabItem.id === "games" && selectedDeckyGameID ? handleDeckyTabCancel : undefined
        )}
      </div>
    </Focusable>
  );
}

function QuickAccessContent() {
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [error, setError] = useState("");

  async function refreshStatus() {
    try {
      setError("");
      setStatus(await call<[], BackendStatus>("status"));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function toggleServer() {
    try {
      setError("");
      const method = status?.running ? "stop_server" : "start_server";
      const next = await call<[], BackendStatus>(method);
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

  useEffect(() => {
    void refreshStatus();
    const listener = () => void refreshStatus();
    window.addEventListener(DMM_EVENT_NAME, listener);
    return () => window.removeEventListener(DMM_EVENT_NAME, listener);
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
          <div style={{ display: "grid", gap: "6px", width: "100%" }}>
            <div>Status: {status?.running ? "Running" : "Stopped"}</div>
            <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Phone URL: {status?.url ?? "Unavailable"}</div>
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
  routerHook.addRoute(DMM_DECKY_ROUTE, () => <DeckyModManagerRoute />);
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
