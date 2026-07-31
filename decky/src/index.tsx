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
  TextField,
  ToggleField,
  showModal,
  staticClasses,
  type GamepadEvent
} from "@decky/ui";
import { call, definePlugin, routerHook, toaster } from "@decky/api";
import { FaPowerOff } from "react-icons/fa";
import { CSSProperties, ReactNode, useEffect, useMemo, useRef, useState } from "react";

declare const SteamClient:
  | {
      Apps?: {
        SetAppLaunchOptions?: (appid: number, launchOptions: string) => void;
        GetSubscribedWorkshopItems?: (appid: number) => Promise<SteamWorkshopClientItem[]>;
        GetDownloadedWorkshopItems?: (appid: number) => Promise<SteamWorkshopClientItem[]>;
        SetWorkshopItemsDisabledLocally?: (appid: number, itemIds: string[], disabled: boolean) => void | Promise<unknown>;
        SetWorkshopItemsLoadOrder?: (appid: number, itemIds: string[]) => void | Promise<unknown>;
        SubscribeWorkshopItem?: (appid: number, itemId: string, subscribed: boolean) => void | Promise<unknown>;
        DownloadWorkshopItem?: (appid: number, itemId: string, highPriority: boolean) => void | Promise<unknown>;
      };
      GameSessions?: {
        RegisterForAppLifetimeNotifications?: (callback: (notification: { unAppID: number; bRunning: boolean }) => void) => { unregister?: () => void; Unregister?: () => void } | (() => void);
      };
    }
  | undefined;

type SteamWorkshopClientItem = {
  unAppID?: number;
  appid?: number;
  ulPublishedFileID?: string;
  publishedfileid?: string;
  published_file_id?: string;
  title?: string;
  strTitle?: string;
  bDisabledLocally?: boolean;
  bDisabled?: boolean;
  disabled?: boolean;
  disabled_locally?: boolean;
};

type BackendStatus = {
  running: boolean;
  ip?: string;
  port: number;
  url?: string;
  pid?: number;
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

type Job = {
  id: string;
  type: string;
  title: string;
  status: string;
  message?: string;
  payload?: Record<string, string>;
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

type ManagedGame = {
  app_id: string;
  name: string;
  state: string;
  nexus_domains?: string[];
};

type RunningGame = {
  app_id: string;
  name: string;
};

type Profile = {
  id: number;
  name: string;
  is_default: boolean;
};

type ManagedMod = {
  id: number;
  name: string;
  enabled: boolean;
  priority: number;
  status: string;
  source_game_domain: string;
  source_mod_id: string;
  source_file_id: string;
  update?: ModUpdate;
};

type ModUpdate = {
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

type NexusModFile = {
  file_id: number;
  name: string;
  version: string;
  category_id: number;
  file_name: string;
  size: number;
  uploaded_timestamp: number;
};

type ProfileApplyResult = {
  status: string;
  message?: string;
  job?: Job;
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
    priority?: number;
    current: boolean;
  }>;
};

type InstallCandidate = {
  id: number;
  steam_app_id: string;
  name: string;
  status: string;
  reason: string;
  installer_json?: string;
  choices_json?: string;
  source_game_domain: string;
  source_mod_id: string;
  source_file_id: string;
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
type DeckyModSort = "profile" | "az" | "za" | "enabled";

const DMM_DECKY_ROUTE = "/decky-mod-manager";
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
const completedLaunchActions = new Set<string>();
const launchActionAttempts = new Map<string, number>();
const completedWorkshopActions = new Set<string>();
const workshopActionAttempts = new Map<string, number>();
const DMM_EVENT_NAME = "dmm-domain-event";
const DMM_BACKEND_WS_URL = "ws://127.0.0.1:17942/api/events/ws";
let eventMonitorSocket: WebSocket | null = null;
let eventMonitorReconnectTimer: number | null = null;
let eventMonitorReconnectDelay = 1000;
let eventMonitorLastID = 0;
let backgroundMonitorsStarted = false;

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

function deckyModStateLabel(mod: ManagedMod) {
  if (mod.status === "needs_recovery") return "Needs repair";
  if (mod.status === "staged") return mod.enabled ? "Enabled" : "Installed";
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
  if (!update) return "Use Check Updates to query Nexus.";
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

function nextDeckyModSort(current: DeckyModSort): DeckyModSort {
  if (current === "profile") return "az";
  if (current === "az") return "za";
  if (current === "za") return "enabled";
  return "profile";
}

function deckyModSortLabel(sort: DeckyModSort) {
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
  if (!Number.isFinite(bytes) || bytes <= 0) return "Unknown size";
  const units = ["B", "KB", "MB", "GB"];
  let amount = bytes;
  let unit = 0;
  while (amount >= 1024 && unit < units.length - 1) {
    amount /= 1024;
    unit++;
  }
  return `${amount.toFixed(unit === 0 ? 0 : 1)} ${units[unit]}`;
}

function nexusFileURL(gameDomain: string, modID: number, fileID: number) {
  return `https://www.nexusmods.com/${encodeURIComponent(gameDomain)}/mods/${modID}?file_id=${fileID}`;
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

async function logSteamClientCapabilities() {
  const root = typeof SteamClient !== "undefined" ? (SteamClient as unknown as Record<string, unknown>) : null;
  if (!root) {
    await logFrontendEvent("steam client capabilities unavailable");
    return;
  }
  const detail: Record<string, string | number | boolean> = {
    top_level: compactLogValue(loggedObjectKeys(root))
  };
  for (const section of ["Apps", "GameSessions", "Workshop", "UGC", "RemoteStorage", "Cloud"]) {
    detail[section] = compactLogValue(loggedObjectKeys(root[section]));
  }
  const appWorkshopMethods = loggedObjectKeyList(root.Apps).filter((key) => /workshop|subscrib|ugc/i.test(key));
  detail.AppsWorkshop = compactLogValue(appWorkshopMethods.join(","));
  await logFrontendEvent("steam client capabilities", detail);
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
  const previous = notifiedJobStates.get(job.id);
  notifiedJobStates.set(job.id, stateKey);
  const updatedAt = Date.parse(job.updated_at || "");
  const recent = Number.isFinite(updatedAt) && Date.now() - updatedAt < 120_000;
  const requireRecent = seed || source === "event" || source === "event-snapshot";
  if (previous !== stateKey && (!requireRecent || recent) && ["waiting", "running", "completed", "failed"].includes(job.status)) {
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
  const raw = item.ulPublishedFileID ?? item.publishedfileid ?? item.published_file_id ?? "";
  return String(raw).trim();
}

function workshopTitleFromClientItem(item: SteamWorkshopClientItem): string {
  const title = item.title ?? item.strTitle ?? "";
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
  for (const key of ["unAppID", "appid", "ulPublishedFileID", "publishedfileid", "published_file_id", "title", "strTitle", "bDisabledLocally", "bDisabled", "disabled", "disabled_locally"] as const) {
    const value = item[key];
    if (typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
      raw[key] = value;
    }
  }
  return JSON.stringify(raw);
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

async function syncWorkshopStateForApp(appID: string) {
  const appid = Number.parseInt(appID, 10);
  const steamApps = typeof SteamClient !== "undefined" ? SteamClient?.Apps : undefined;
  if (!Number.isFinite(appid) || !steamApps) return false;
  if (typeof steamApps.GetSubscribedWorkshopItems !== "function" && typeof steamApps.GetDownloadedWorkshopItems !== "function") {
    await logFrontendEvent("workshop sync steam api unavailable", { app_id: appID });
    return false;
  }

  let subscribed: SteamWorkshopClientItem[] = [];
  let downloaded: SteamWorkshopClientItem[] = [];
  try {
    if (typeof steamApps.GetSubscribedWorkshopItems === "function") {
      subscribed = await steamApps.GetSubscribedWorkshopItems(appid);
    }
  } catch (err) {
    await logFrontendEvent("workshop subscribed sync failed", { app_id: appID, error: err instanceof Error ? err.message : String(err) });
  }
  try {
    if (typeof steamApps.GetDownloadedWorkshopItems === "function") {
      downloaded = await steamApps.GetDownloadedWorkshopItems(appid);
    }
  } catch (err) {
    await logFrontendEvent("workshop downloaded sync failed", { app_id: appID, error: err instanceof Error ? err.message : String(err) });
  }

  const items = mergeWorkshopItems(appID, subscribed ?? [], downloaded ?? []);
  const result = await call<[string, WorkshopItem[]], { ok: boolean; error?: string }>("sync_workshop", appID, items);
  if (!result.ok) {
    await logFrontendEvent("workshop sync backend rejected", { app_id: appID, error: result.error || "" });
    return false;
  }
  await logFrontendEvent("workshop state synced", { app_id: appID, subscribed: subscribed.length, downloaded: downloaded.length, items: items.length });
  return true;
}

async function executeWorkshopAction(job: WorkshopActionJob) {
  const appID = String(job.payload?.app_id ?? "").trim();
  const itemID = String(job.payload?.item_id ?? "").trim();
  const kind = String(job.payload?.kind ?? "").trim();
  const appid = Number.parseInt(appID, 10);
  const steamApps = typeof SteamClient !== "undefined" ? SteamClient?.Apps : undefined;
  if (!Number.isFinite(appid) || !itemID || !steamApps) {
    throw new Error("Steam Workshop API is unavailable in this Decky context.");
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
          await syncWorkshopStateForApp(action.payload.app_id);
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

function InstallerChoiceModal(props: { appID: string; candidate: InstallCandidate; closeModal: () => void; onApplied: () => void }) {
  const [candidate, setCandidate] = useState<InstallCandidate>(props.candidate);
  const installer = installerForCandidate(candidate);
  const [selections, setSelections] = useState<Record<string, string[]>>(() => storedFomodSelections(props.candidate) ?? {});
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const selectedChoices = selectionCount(selections);

  useEffect(() => {
    setCandidate(props.candidate);
    setSelections(storedFomodSelections(props.candidate) ?? {});
  }, [props.candidate.id]);

  function pluginSelected(group: FomodGroup, plugin: FomodPlugin) {
    return (selections[group.id] ?? []).includes(plugin.id);
  }

  function setPluginSelection(group: FomodGroup, plugin: FomodPlugin, checked: boolean) {
    const type = fomodGroupType(group);
    if (type === "selectall") return;
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
    setBusy(true);
    setMessage("");
    try {
      const result = await call<[string, number, Record<string, string[]>], { ok: boolean; error?: string; result?: { job?: Job; mod?: ManagedMod } }>(
        "apply_install_candidate",
        props.appID,
        candidate.id,
        selections
      );
      if (!result.ok) {
        setMessage(result.error || "Unable to apply installer choices.");
        await logFrontendEvent("installer choice modal apply failed", { app_id: props.appID, candidate_id: candidate.id, error: result.error || "" });
        return;
      }
      if (result.result?.job) showJobToast(result.result.job);
      await logFrontendEvent("installer choice modal applied", { app_id: props.appID, candidate_id: candidate.id });
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
          {installer && visibleFomodSteps(installer).map((step) => (
            <section key={step.id} style={{ display: "grid", gap: "8px" }}>
              <div style={{ fontWeight: 800 }}>{step.name}</div>
              {step.groups?.map((group) => (
                <fieldset key={group.id} style={{ border: "1px solid #303741", borderRadius: "6px", display: "grid", gap: "8px", margin: 0, padding: "10px" }}>
                  <legend style={{ color: "#7dd3fc", fontWeight: 800, padding: "0 4px" }}>{group.name}</legend>
                  {group.plugins?.map((plugin) => (
                    <label key={plugin.id} style={{ alignItems: "flex-start", display: "grid", gap: "8px", gridTemplateColumns: "22px minmax(0, 1fr)" }}>
                      <input
                        type={fomodGroupInputType(group)}
                        name={`candidate-${candidate.id}-${group.id}`}
                        checked={pluginSelected(group, plugin)}
                        disabled={busy || fomodGroupType(group) === "selectall"}
                        onChange={(event) => setPluginSelection(group, plugin, event.currentTarget.checked)}
                      />
                      <span style={{ display: "grid", gap: "3px", minWidth: 0 }}>
                        <strong>{plugin.name}</strong>
                        {plugin.type && <small style={{ color: "#a1a1aa" }}>{plugin.type}</small>}
                        {plugin.description && <em style={{ color: "#d4d4d8", fontStyle: "normal", overflowWrap: "anywhere" }}>{plugin.description}</em>}
                      </span>
                    </label>
                  ))}
                </fieldset>
              ))}
            </section>
          ))}
          {message && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{message}</div>}
        </div>
      }
      strOKButtonText={busy ? "Applying..." : "Apply Choices"}
      strCancelButtonText="Later"
      bOKDisabled={busy || !installer}
      onOK={() => void applyChoices()}
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

async function openInstallerChoiceModalForCandidate(appID: string, candidate: InstallCandidate, source: string, onApplied?: () => void) {
  const key = String(candidate.id);
  if (shownInstallerChoiceModals.has(key)) return;
  if (!installerForCandidate(candidate)) {
    await logFrontendEvent("installer choice modal skipped for candidate without installer", { app_id: appID, candidate_id: candidate.id, source });
    showLaunchToast("DMM installer choices unavailable", "Open the phone UI to review this installer item.", false);
    return;
  }
  shownInstallerChoiceModals.add(key);
  try {
    let modal: { Close: () => void } | null = null;
    const closeModal = () => {
      shownInstallerChoiceModals.delete(key);
      modal?.Close();
    };
    modal = showModal(
      <InstallerChoiceModal
        appID={appID}
        candidate={candidate}
        closeModal={closeModal}
        onApplied={() => {
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

function NexusBrowserModal(props: { appID: string; gameName: string; gameDomain: string; closeModal: () => void }) {
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<NexusSearchSort>("downloads");
  const [timeWindow, setTimeWindow] = useState<NexusTimeWindow>("all");
  const [vortexOnly, setVortexOnly] = useState(true);
  const [mods, setMods] = useState<NexusModResult[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [offset, setOffset] = useState(0);
  const [selectedModID, setSelectedModID] = useState<number | null>(null);
  const [filesByMod, setFilesByMod] = useState<Record<number, NexusModFile[]>>({});
  const [busy, setBusy] = useState(false);
  const [busyFileKey, setBusyFileKey] = useState("");
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

  async function loadFiles(mod: NexusModResult) {
    setSelectedModID(mod.mod_id);
    setError("");
    setMessage("");
    if (filesByMod[mod.mod_id]) return;
    setBusyFileKey(`files:${mod.mod_id}`);
    try {
      const result = await call<[string, string], { ok: boolean; error?: string; files: NexusModFile[] }>("nexus_mod_files", props.appID, String(mod.mod_id));
      if (!result.ok) {
        setError(result.error || "Unable to load Nexus files. Check the Nexus API key in DMM settings.");
        return;
      }
      setFilesByMod((current) => ({ ...current, [mod.mod_id]: result.files ?? [] }));
      if ((result.files ?? []).length === 0) setMessage("This mod did not return installable files from Nexus.");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyFileKey("");
    }
  }

  async function installFile(mod: NexusModResult, file: NexusModFile) {
    const key = `${mod.mod_id}:${file.file_id}`;
    setBusyFileKey(key);
    setError("");
    setMessage("");
    try {
      const url = nexusFileURL(props.gameDomain, mod.mod_id, file.file_id);
      await logFrontendEvent("nexus browser install requested", {
        app_id: props.appID,
        game_domain: props.gameDomain,
        mod_id: mod.mod_id,
        file_id: file.file_id
      });
      const result = await call<[string], { ok: boolean; error?: string; result?: { job?: Job } }>("add_captured_install", url);
      if (!result.ok) {
        setError(result.error || "Unable to add this Nexus file.");
        await logFrontendEvent("nexus browser install failed", {
          app_id: props.appID,
          game_domain: props.gameDomain,
          mod_id: mod.mod_id,
          file_id: file.file_id,
          error: result.error || ""
        });
        return;
      }
      const job = result.result?.job;
      if (job) showJobToast(job);
      await logFrontendEvent("nexus browser install queued", {
        app_id: props.appID,
        game_domain: props.gameDomain,
        mod_id: mod.mod_id,
        file_id: file.file_id,
        job_id: job?.id || "",
        status: job?.status || ""
      });
      if (job?.status === "failed") {
        setError(job.message || "DMM could not start this Nexus download.");
        return;
      }
      setMessage(job?.message || `${file.name || file.file_name || mod.name} was sent to DMM.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      await logFrontendEvent("nexus browser install threw", {
        app_id: props.appID,
        game_domain: props.gameDomain,
        mod_id: mod.mod_id,
        file_id: file.file_id,
        error: err instanceof Error ? err.message : String(err)
      });
    } finally {
      setBusyFileKey("");
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
            const files = filesByMod[mod.mod_id] ?? [];
            const filesOpen = selectedModID === mod.mod_id;
            return (
              <div
                key={mod.mod_id}
                className="dmm-sidebar-row"
                style={{
                  ...deckyCompositeRowStyle(filesOpen),
                  alignSelf: "start",
                  background: "#111827",
                  flexShrink: 0,
                  minHeight: filesOpen ? "auto" : "146px",
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
                    <div style={{ ...deckyTwoLineTextStyle, fontWeight: 900 }}>{mod.name}</div>
                    <div style={{ color: "#d4d4d8", fontSize: "11px", lineHeight: 1.25, maxHeight: "3.75em", overflow: "hidden", overflowWrap: "anywhere" }}>{mod.summary}</div>
                    <div style={{ color: "#a1a1aa", display: "flex", flexWrap: "wrap", fontSize: "11px", gap: "10px" }}>
                      <span>v{mod.version || "unknown"}</span>
                      <span>{compactNumber(mod.downloads)} downloads</span>
                      <span>{compactNumber(mod.endorsements)} endorsements</span>
                      <span>{mod.updated_at ? `Updated ${new Date(mod.updated_at).toLocaleDateString()}` : "Updated unknown"}</span>
                    </div>
                  </div>
                </div>
                <Focusable className="dmm-action-grid" flow-children="right" style={deckyActionGridStyle(2)}>
                  <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => Navigation.NavigateToExternalWeb(mod.url)} onClick={() => Navigation.NavigateToExternalWeb(mod.url)} style={deckyCompactActionStyle("neutral")}>
                    Open Page
                  </Focusable>
                  <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => loadFiles(mod)} onClick={() => loadFiles(mod)} style={deckyCompactActionStyle("neutral", filesOpen || busyFileKey === `files:${mod.mod_id}`)}>
                    {busyFileKey === `files:${mod.mod_id}` ? "Loading Files" : filesOpen ? "Refresh Files" : "Show Files"}
                  </Focusable>
                </Focusable>
                {filesOpen && (
                  <div style={{ display: "grid", gap: "8px", width: "100%" }}>
                    {files.length === 0 && busyFileKey !== `files:${mod.mod_id}` && <div style={{ color: "#a1a1aa" }}>No files loaded yet.</div>}
                    {files.map((file) => {
                      const fileKey = `${mod.mod_id}:${file.file_id}`;
                      return (
                        <div key={file.file_id} style={{ background: "#0b1220", border: "1px solid #303741", borderRadius: "6px", boxSizing: "border-box", display: "grid", gap: "8px", padding: "10px", width: "100%" }}>
                          <div style={{ ...deckyTwoLineTextStyle, fontWeight: 800 }}>{file.name || file.file_name || `File ${file.file_id}`}</div>
                          <div style={{ color: "#a1a1aa", fontSize: "11px", overflowWrap: "anywhere" }}>
                            {file.file_name || "Nexus file"} · {formatBytes(file.size)} · v{file.version || "unknown"}
                          </div>
                          <Focusable className="dmm-action-grid" flow-children="right" style={deckyActionGridStyle(2)}>
                            <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => Navigation.NavigateToExternalWeb(nexusFileURL(props.gameDomain, mod.mod_id, file.file_id))} onClick={() => Navigation.NavigateToExternalWeb(nexusFileURL(props.gameDomain, mod.mod_id, file.file_id))} style={deckyCompactActionStyle("neutral")}>
                              Open File Page
                            </Focusable>
                            <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => installFile(mod, file)} onClick={() => installFile(mod, file)} style={deckyCompactActionStyle("neutral", busyFileKey === fileKey)}>
                              {busyFileKey === fileKey ? "Adding To DMM" : "Install With DMM"}
                            </Focusable>
                          </Focusable>
                        </div>
                      );
                    })}
                  </div>
                )}
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
      shownInstallerChoiceModals.delete(event.payload.payload.candidate_id);
    }
    await maybeShowInstallerChoiceModal(event.payload);
  }
  if (["job.updated", "profile_mods.changed", "deployment.changed", "install.changed", "mod_updates.changed"].includes(event.type)) {
    await syncLaunchActions();
  }
  if (event.type === "job.updated") {
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
  connectEventMonitor();
}

function stopBackgroundMonitors() {
  if (!backgroundMonitorsStarted) return;
  backgroundMonitorsStarted = false;
  closeEventMonitor();
  logFrontendEvent("background monitors stopped");
}

function openDeckyModManagerRoute() {
  Router.CloseSideMenus();
  Router.Navigate(DMM_DECKY_ROUTE);
}

function DeckyModManagerRoute() {
  const selectedDeckyGameRef = useRef<HTMLDivElement | null>(null);
  const [tab, setTab] = useState<Tab>("main");
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [dependencies, setDependencies] = useState<Dependency[]>([]);
  const [nxm, setNXM] = useState<NXMStatus | null>(null);
  const [importUrl, setImportUrl] = useState<string>("");
  const [importResult, setImportResult] = useState<string>("");
  const [launchResult, setLaunchResult] = useState<string>("");
  const [diagnostics, setDiagnostics] = useState<Diagnostics | null>(null);
  const [error, setError] = useState<string>("");
  const [managedGames, setManagedGames] = useState<ManagedGame[]>([]);
  const [selectedDeckyGameID, setSelectedDeckyGameID] = useState<string>("");
  const [runningGame, setRunningGame] = useState<RunningGame | null>(null);
  const [deckyProfiles, setDeckyProfiles] = useState<Profile[]>([]);
  const [deckyMods, setDeckyMods] = useState<ManagedMod[]>([]);
  const [deckyInstallCandidates, setDeckyInstallCandidates] = useState<InstallCandidate[]>([]);
  const [deckyWorkshopItems, setDeckyWorkshopItems] = useState<WorkshopItem[]>([]);
  const [deckyWorkshopSupported, setDeckyWorkshopSupported] = useState<boolean>(false);
  const [deckyLoadOrder, setDeckyLoadOrder] = useState<PluginLoadOrder | null>(null);
  const [deckyDeployPlan, setDeckyDeployPlan] = useState<DeployPlan | null>(null);
  const [modsResult, setModsResult] = useState<string>("");
  const [modSearch, setModSearch] = useState<string>("");
  const [modSort, setModSort] = useState<DeckyModSort>("profile");
  const [modOrderMode, setModOrderMode] = useState<boolean>(false);
  const [gameSearch, setGameSearch] = useState<string>("");
  const [gameSort, setGameSortState] = useState<GameSort>("recent");
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

  async function refresh() {
    try {
      setError("");
      const nextStatus = await call<[], BackendStatus>("status");
      setStatus(nextStatus);
      applyDeckyUIPreferences(nextStatus);
      setDependencies(await call<[], Dependency[]>("dependencies"));
      setNXM(await call<[], NXMStatus>("nxm_status"));
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
      setDeckyDeployPlan(null);
      return null;
    }
    const [profilesResult, modsResult, candidatesResult, workshopResult, loadOrderResult, deployPreviewResult] = await Promise.all([
      call<[string], { ok: boolean; error?: string; profiles: Profile[] }>("game_profiles", appID),
      call<[string], { ok: boolean; error?: string; mods: ManagedMod[] }>("game_mods", appID),
      call<[string], { ok: boolean; error?: string; candidates: InstallCandidate[] }>("game_install_candidates", appID),
      call<[string], { ok: boolean; error?: string; state?: WorkshopState; items: WorkshopItem[] }>("game_workshop", appID),
      call<[string], { ok: boolean; error?: string; load_order?: PluginLoadOrder }>("game_load_order", appID),
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
    if (deployPreviewResult.ok && deployPreviewResult.plan) {
      setDeckyDeployPlan(deployPreviewResult.plan);
    } else {
      setDeckyDeployPlan(null);
    }
    void syncWorkshopStateForApp(appID).then((synced) => {
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
      const selected = runningSupported ? running.app_id : appID || selectedDeckyGameID;
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
    });
  }

  async function removeDeckyMod(mod: ManagedMod) {
    if (!selectedDeckyGameID) return;
    try {
      setError("");
      setModsResult("");
      setBusyModID(mod.id);
      const result = await call<[string, number], { ok: boolean; error?: string; result?: { job?: Job; apply?: ProfileApplyResult } }>("remove_game_mod", selectedDeckyGameID, mod.id);
      if (!result.ok) {
        setError(result.error ?? "Unable to remove mod.");
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      const applyMessage = result.result?.apply?.message || "Mod removed. Restart the game if it is already running.";
      if (result.result?.apply?.status === "blocked" || result.result?.apply?.status === "failed") setError(applyMessage);
      else setModsResult(applyMessage);
      if (result.result?.job) showJobToast(result.result.job);
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
      setModsResult(available > 0 ? `${available} update${available === 1 ? "" : "s"} available.` : `Checked ${checked} Nexus mod${checked === 1 ? "" : "s"}.`);
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

  function askUnsubscribeWorkshopItem(item: WorkshopItem) {
    let modal: { Close: () => void } | null = null;
    const closeModal = () => modal?.Close();
    modal = showModal(
      <ConfirmModal
        strTitle={`Unsubscribe ${workshopItemName(item)}`}
        strDescription="Steam will remove this Workshop subscription for the selected game. DMM-managed Nexus mods are not changed."
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
        closeModal={closeModal}
      />,
      window,
      { strTitle: "Browse Nexus Mods", bNeverPopOut: true, bHideActionIcons: true, popupWidth: 760, popupHeight: 820 }
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

  async function loadDiagnostics() {
    try {
      setError("");
      setDiagnostics(await call<[], Diagnostics>("diagnostics"));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function addCapturedInstall() {
    try {
      setError("");
      setImportResult("");
      const result = await call<[string], { ok: boolean; error?: string; result?: { job?: Job } }>("add_captured_install", importUrl);
      if (!result.ok) {
        setError(result.error ?? "Unable to capture Nexus link.");
        return;
      }
      setImportUrl("");
      const job = result.result?.job;
      setImportResult(job?.message || job?.title || "Nexus link captured.");
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
      if (!running || tab !== "games" || !managedGames.some((game) => game.app_id === running.app_id) || selectedDeckyGameID === running.app_id) return;
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
  const selectedProfile = deckyProfiles.find((item) => item.is_default) ?? deckyProfiles[0] ?? null;
  const runningSupported = Boolean(runningGame && managedGames.some((game) => game.app_id === runningGame.app_id));
  const favoriteGameKey = [...favoriteGameIDs].sort().join("|");
  const effectiveModSort = modOrderMode ? "profile" : modSort;
  const visibleManagedGames = useMemo(() => {
    const normalizedGameSearch = gameSearch.trim().toLowerCase();
    const favoriteIDs = new Set(favoriteGameKey ? favoriteGameKey.split("|") : []);
    return [...managedGames].filter((game) => {
      if (!normalizedGameSearch) return true;
      return game.name.toLowerCase().includes(normalizedGameSearch) || game.app_id.includes(normalizedGameSearch);
    })
    .sort((a, b) => {
      const favoriteDelta = Number(favoriteIDs.has(b.app_id)) - Number(favoriteIDs.has(a.app_id));
      if (favoriteDelta !== 0) return favoriteDelta;
      if (gameSort === "az") return a.name.localeCompare(b.name);
      if (gameSort === "za") return b.name.localeCompare(a.name);
      const recentDelta = (gameRecent[b.app_id] ?? 0) - (gameRecent[a.app_id] ?? 0);
      if (recentDelta !== 0) return recentDelta;
      return a.name.localeCompare(b.name);
    });
  }, [managedGames, gameSearch, favoriteGameKey, gameSort, gameRecent]);
  const visibleDeckyMods = useMemo(() => {
    const normalizedModSearch = modSearch.trim().toLowerCase();
    const filtered = deckyMods.filter((mod) =>
      !normalizedModSearch ||
        [mod.name, mod.status, mod.source_game_domain, mod.source_mod_id, mod.source_file_id]
          .some((value) => String(value ?? "").toLowerCase().includes(normalizedModSearch))
      );
    return [...filtered].sort((a, b) => {
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
          <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800, lineHeight: 1, textTransform: "uppercase" }}>Nexus Link</div>
          <input
            aria-label="Nexus URL"
            placeholder="Paste Nexus URL or nxm:// link"
            style={{ ...deckyCompactInputStyle, minHeight: "34px", padding: "6px 9px" }}
            value={importUrl}
            onChange={(event) => setImportUrl(event.currentTarget.value)}
          />
          <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={addCapturedInstall} onClick={addCapturedInstall} style={{ ...deckyCompactActionStyle("neutral"), minHeight: "34px", padding: "7px 6px" }}>
            Add Nexus Link
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
            {managedGames.length === 0 && <div style={{ color: "#a1a1aa" }}>No games loaded.</div>}
            {managedGames.length > 0 && visibleManagedGames.length === 0 && <div style={{ color: "#a1a1aa" }}>No games match this search.</div>}
            <Focusable flow-children="down" navEntryPreferPosition={NavEntryPositionPreferences.FIRST} style={deckySidebarListStyle}>
              {visibleManagedGames.map((game) => {
                const focused = focusedGameID === game.app_id;
                const favorite = favoriteGameIDs.has(game.app_id);
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
              onActivate={() => {
                if (selectedNexusDomain) openDeckyNexusBrowser();
              }}
              onClick={() => {
                if (selectedNexusDomain) openDeckyNexusBrowser();
              }}
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
                <div style={{ color: selectedNexusDomain ? "#99f6e4" : "#a1a1aa", fontSize: "11px", fontWeight: 800, lineHeight: 1.25, marginTop: "4px", overflowWrap: "anywhere" }}>
                  {selectedNexusDomain ? "A Browse Nexus · B Change Game" : "B Change Game"}
                </div>
              </div>
            </Focusable>
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
          <PanelSectionRow>
            <ButtonItem layout="below" onClick={clearSelectedDeckyGame}>
              Change Game
            </ButtonItem>
          </PanelSectionRow>
          {!selectedNexusDomain && (
            <PanelSectionRow>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>No Nexus page is registered for this game yet.</div>
            </PanelSectionRow>
          )}
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
                      <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", fontWeight: 800 }}>{candidate.name}</div>
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
                    <div style={deckyTwoLineTextStyle}>{profile.name}</div>
                  </Focusable>
                ))}
              </div>
            </PanelSectionRow>
          )}
          {deckyMods.length === 0 && deckyWorkshopItems.length === 0 && (
            <PanelSectionRow>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>No profile mods yet. Add Nexus downloads from the Decky paste field or phone/tablet UI.</div>
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
                      onSecondaryActionDescription={modOrderMode ? "Move Down" : "Reinstall"}
                      onSecondaryButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) void moveDeckyModInProfile(mod, 1);
                        else void reinstallDeckyMod(mod);
                      }}
                      onOptionsActionDescription={modOrderMode ? "Done Ordering" : "Remove"}
                      onOptionsButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) setModOrderMode(false);
                        else askRemoveDeckyMod(mod);
                      }}
                      onMenuActionDescription={modOrderMode ? "Done Ordering" : "Remove"}
                      onMenuButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (modOrderMode) setModOrderMode(false);
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
                      <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", fontWeight: 800 }}>{mod.name}</div>
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
                        {modOrderMode ? "A Move Up · Y Move Down · Options Done" : `A ${mod.enabled ? "Disable" : "Enable"} · Y Reinstall · Options Remove`}
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
                          <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", fontWeight: 800 }}>{plugin.name}</div>
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
                      <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                        Current: {target.current_winner_name}
                      </div>
                      <div style={{ color: "#d4d4d8", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                        {target.candidates.map((candidate) => `${candidate.current ? "Using" : "Option"} ${candidate.name}`).join(" · ")}
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
                  const busy = busyWorkshopKey === toggleKey || busyWorkshopKey === unsubscribeKey;
                  const toggleSupported = deckyWorkshopSupported && item.disabled_known;
                  return (
                    <Focusable
                      key={item.published_file_id}
                      className="dmm-sidebar-row"
                      focusClassName="dmm-sidebar-row-focused"
                      onActivate={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        if (toggleSupported) void queueDeckyWorkshopAction(item, toggleKind);
                      }}
                      onClick={() => {
                        if (toggleSupported) void queueDeckyWorkshopAction(item, toggleKind);
                      }}
                      onSecondaryActionDescription="Unsubscribe"
                      onSecondaryButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        askUnsubscribeWorkshopItem(item);
                      }}
                      style={{
                        ...deckyCompositeRowStyle(false, !disabled),
                        opacity: busy ? 0.65 : 1,
                        padding: "10px"
                      }}
                    >
                      <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", fontWeight: 800 }}>{workshopItemName(item)}</div>
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
                        A {busyWorkshopKey === toggleKey ? "Queueing" : toggleSupported ? (disabled ? "Enable" : "Disable") : "Sync Needed"} · Y {busyWorkshopKey === unsubscribeKey ? "Queueing" : "Unsubscribe"}
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
        <ButtonItem layout="below" onClick={loadDiagnostics}>
          Load Diagnostics
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
          {diagnostics && (
            <div style={{ display: "grid", gap: "10px", marginTop: "10px", width: "100%" }}>
              {Object.entries(diagnostics.logs).map(([name, log]) => (
                <div key={name} style={{ borderTop: "1px solid #303741", paddingTop: "8px" }}>
                  <div style={{ fontWeight: 800 }}>{name}</div>
                  <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>{log.path}</div>
                  <pre style={{ color: "#d4d4d8", fontSize: "10px", maxWidth: "100%", overflowX: "auto", whiteSpace: "pre-wrap" }}>{log.tail || "No log entries."}</pre>
                </div>
              ))}
            </div>
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
  routerHook.addRoute(DMM_DECKY_ROUTE, DeckyModManagerRoute);
  return {
    name: "Decky Mod Manager",
    title: <div className={staticClasses.Title}>Decky Mod Manager</div>,
    alwaysRender: true,
    content: <QuickAccessContent />,
    icon: <FaPowerOff />,
    onDismount() {
      routerHook.removeRoute(DMM_DECKY_ROUTE);
      stopBackgroundMonitors();
    }
  };
});
