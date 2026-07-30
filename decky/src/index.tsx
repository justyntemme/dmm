import {
  ButtonItem,
  ConfirmModal,
  Focusable,
  GamepadButton,
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
import { call, definePlugin, toaster } from "@decky/api";
import { FaPowerOff } from "react-icons/fa";
import { CSSProperties, ReactNode, useEffect, useMemo, useState } from "react";

declare const SteamClient:
  | {
      Apps?: {
        SetAppLaunchOptions?: (appid: number, launchOptions: string) => void;
        GetSubscribedWorkshopItems?: (appid: number) => Promise<SteamWorkshopClientItem[]>;
        GetDownloadedWorkshopItems?: (appid: number) => Promise<SteamWorkshopClientItem[]>;
        SetWorkshopItemsDisabledLocally?: (appid: number, itemIds: string[], disabled: boolean) => void;
        SetWorkshopItemsLoadOrder?: (appid: number, itemIds: string[]) => void;
        SubscribeWorkshopItem?: (appid: number, itemId: string, subscribed: boolean) => void;
        DownloadWorkshopItem?: (appid: number, itemId: string, highPriority: boolean) => void;
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
};

type NexusSearchSort = "downloads" | "updated" | "endorsements" | "name" | "relevance";

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

type WorkshopActionJob = Job & {
  payload?: {
    app_id?: string;
    item_id?: string;
    kind?: "subscribe" | "unsubscribe" | "enable" | "disable" | string;
    action_name?: string;
  };
};

type Tab = "main" | "mods" | "settings" | "debug";
type GameSort = "recent" | "az" | "za";

const deckyTabOrder: Tab[] = ["main", "mods", "settings", "debug"];
const deckyGameListWindowSize = 18;
const deckyModListWindowSize = 12;

const deckyPanelFrameStyle: CSSProperties = {
  alignSelf: "stretch",
  boxSizing: "border-box",
  display: "flex",
  flexDirection: "column",
  gap: "8px",
  height: "calc(100vh - 112px)",
  maxHeight: "calc(100vh - 112px)",
  maxWidth: "100%",
  minHeight: 0,
  minWidth: 0,
  overflowX: "hidden",
  width: "100%"
};

const deckyTabBarStyle: CSSProperties = {
  alignSelf: "stretch",
  boxSizing: "border-box",
  display: "grid",
  gap: "4px",
  gridTemplateColumns: "repeat(4, minmax(0, 1fr))",
  minWidth: 0,
  width: "100%"
};

const deckyTabBodyStyle: CSSProperties = {
  boxSizing: "border-box",
  display: "block",
  flex: "1 1 auto",
  minHeight: 0,
  overflowY: "auto",
  paddingBottom: "16px",
  paddingRight: "4px",
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
.dmm-focus-card-focused {
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

function deckyTabButtonStyle(active: boolean): CSSProperties {
  return {
    alignItems: "center",
    background: active ? "#0f766e" : "#1f2937",
    border: `1px solid ${active ? "#5eead4" : "#374151"}`,
    borderRadius: "6px",
    boxSizing: "border-box",
    color: "#f8fafc",
    display: "flex",
    fontSize: "11px",
    fontWeight: 800,
    height: "34px",
    justifyContent: "center",
    lineHeight: 1,
    minWidth: 0,
    overflow: "hidden",
    padding: "0 4px",
    textAlign: "center",
    textOverflow: "ellipsis",
    textTransform: "uppercase",
    whiteSpace: "nowrap",
    width: "100%"
  };
}

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

function deckyActionGridStyle(columns: 1 | 2): CSSProperties {
  return {
    alignItems: "stretch",
    boxSizing: "border-box",
    display: "grid",
    gap: "6px",
    gridTemplateColumns: columns === 2 ? "minmax(0, 1fr) minmax(0, 1fr)" : "minmax(0, 1fr)",
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

function deckyTabBody(content: ReactNode, onCancelButton?: (event: GamepadEvent) => void) {
  return (
    <Focusable
      flow-children="column"
      navEntryPreferPosition={NavEntryPositionPreferences.FIRST}
      onCancelActionDescription={onCancelButton ? "Back" : undefined}
      onCancelButton={onCancelButton}
      style={deckyTabBodyStyle}
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

function deckyModStateLabel(mod: ManagedMod) {
  if (mod.status === "needs_recovery") return "Needs repair";
  if (mod.status === "staged") return mod.enabled ? "Enabled" : "Installed";
  return mod.status || (mod.enabled ? "Enabled" : "Installed");
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

function nextNexusSort(current: NexusSearchSort): NexusSearchSort {
  if (current === "downloads") return "updated";
  if (current === "updated") return "endorsements";
  if (current === "endorsements") return "name";
  if (current === "name") return "relevance";
  return "downloads";
}

function nexusSortLabel(sort: NexusSearchSort) {
  if (sort === "updated") return "Updated";
  if (sort === "endorsements") return "Endorsed";
  if (sort === "name") return "Name";
  if (sort === "relevance") return "Relevant";
  return "Downloads";
}

function compactNumber(value: number | undefined) {
  const normalized = Number(value ?? 0);
  if (!Number.isFinite(normalized)) return "0";
  return normalized.toLocaleString(undefined, { maximumFractionDigits: 0, notation: normalized >= 10_000 ? "compact" : "standard" });
}

function windowedList<T>(items: T[], focusedIndex: number, maxItems: number) {
  const total = items.length;
  if (total <= maxItems) {
    return { items, start: 0, end: total, total };
  }
  const safeIndex = Math.max(0, Math.min(total - 1, focusedIndex < 0 ? 0 : focusedIndex));
  const half = Math.floor(maxItems / 2);
  const start = Math.max(0, Math.min(total - maxItems, safeIndex - half));
  const end = Math.min(total, start + maxItems);
  return { items: items.slice(start, end), start, end, total };
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

function defaultFomodSelections(installer: FomodInstaller): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const step of installer.steps ?? []) {
    for (const group of step.groups ?? []) {
      const plugins = group.plugins ?? [];
      const type = fomodGroupType(group);
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

function storedFomodSelections(candidate: InstallCandidate): Record<string, string[]> | null {
  if (!candidate.choices_json) return null;
  try {
    const parsed = JSON.parse(candidate.choices_json) as Record<string, string[]>;
    return parsed && typeof parsed === "object" ? parsed : null;
  } catch (_err) {
    return null;
  }
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
    steamApps.SetWorkshopItemsDisabledLocally(appid, [itemID], false);
    return;
  }
  if (kind === "disable") {
    if (typeof steamApps.SetWorkshopItemsDisabledLocally !== "function") throw new Error("Steam Workshop disable API is unavailable.");
    steamApps.SetWorkshopItemsDisabledLocally(appid, [itemID], true);
    return;
  }
  if (kind === "subscribe") {
    if (typeof steamApps.SubscribeWorkshopItem !== "function") throw new Error("Steam Workshop subscribe API is unavailable.");
    steamApps.SubscribeWorkshopItem(appid, itemID, true);
    if (typeof steamApps.DownloadWorkshopItem === "function") {
      steamApps.DownloadWorkshopItem(appid, itemID, true);
    }
    return;
  }
  if (kind === "unsubscribe") {
    if (typeof steamApps.SubscribeWorkshopItem !== "function") throw new Error("Steam Workshop unsubscribe API is unavailable.");
    steamApps.SubscribeWorkshopItem(appid, itemID, false);
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
  const installer = installerForCandidate(props.candidate);
  const [selections, setSelections] = useState<Record<string, string[]>>(() => storedFomodSelections(props.candidate) ?? (installer ? defaultFomodSelections(installer) : {}));
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");

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
      const result = await call<[string, number, Record<string, string[]>], { ok: boolean; error?: string }>(
        "save_install_candidate_choices",
        props.appID,
        props.candidate.id,
        nextSelections
      );
      if (!result.ok) {
        await logFrontendEvent("installer choice modal save failed", { app_id: props.appID, candidate_id: props.candidate.id, error: result.error || "" });
      }
    } catch (err) {
      await logFrontendEvent("installer choice modal save threw", { app_id: props.appID, candidate_id: props.candidate.id, error: err instanceof Error ? err.message : String(err) });
    }
  }

  async function applyChoices() {
    if (!installer || busy) return;
    setBusy(true);
    setMessage("");
    try {
      const result = await call<[string, number, Record<string, string[]>], { ok: boolean; error?: string; result?: { job?: Job; mod?: ManagedMod } }>(
        "apply_install_candidate",
        props.appID,
        props.candidate.id,
        selections
      );
      if (!result.ok) {
        setMessage(result.error || "Unable to apply installer choices.");
        await logFrontendEvent("installer choice modal apply failed", { app_id: props.appID, candidate_id: props.candidate.id, error: result.error || "" });
        return;
      }
      if (result.result?.job) showJobToast(result.result.job);
      await logFrontendEvent("installer choice modal applied", { app_id: props.appID, candidate_id: props.candidate.id });
      props.onApplied();
      props.closeModal();
    } catch (err) {
      const error = err instanceof Error ? err.message : String(err);
      setMessage(error);
      await logFrontendEvent("installer choice modal apply threw", { app_id: props.appID, candidate_id: props.candidate.id, error });
    } finally {
      setBusy(false);
    }
  }

  return (
    <ConfirmModal
      strTitle={props.candidate.name}
      strDescription={
        <div style={{ display: "grid", gap: "12px", maxHeight: "62vh", overflowY: "auto", paddingRight: "4px" }}>
          <div style={{ color: "#a1a1aa" }}>{props.candidate.reason || "Choose installer options before DMM adds this mod to the profile."}</div>
          {!installer && <div style={{ color: "#f87171" }}>Installer choices are not available for this request.</div>}
          {installer?.steps?.map((step) => (
            <section key={step.id} style={{ display: "grid", gap: "8px" }}>
              <div style={{ fontWeight: 800 }}>{step.name}</div>
              {step.groups?.map((group) => (
                <fieldset key={group.id} style={{ border: "1px solid #303741", borderRadius: "6px", display: "grid", gap: "8px", margin: 0, padding: "10px" }}>
                  <legend style={{ color: "#7dd3fc", fontWeight: 800, padding: "0 4px" }}>{group.name}</legend>
                  {group.plugins?.map((plugin) => (
                    <label key={plugin.id} style={{ alignItems: "flex-start", display: "grid", gap: "8px", gridTemplateColumns: "22px minmax(0, 1fr)" }}>
                      <input
                        type={fomodGroupInputType(group)}
                        name={`candidate-${props.candidate.id}-${group.id}`}
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
  const [mods, setMods] = useState<NexusModResult[]>([]);
  const [totalCount, setTotalCount] = useState(0);
  const [selectedModID, setSelectedModID] = useState<number | null>(null);
  const [filesByMod, setFilesByMod] = useState<Record<number, NexusModFile[]>>({});
  const [busy, setBusy] = useState(false);
  const [busyFileKey, setBusyFileKey] = useState("");
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  async function searchMods(nextSort = sort) {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const result = await call<[string, string, string, number, number], { ok: boolean; error?: string; mods: NexusModResult[]; total_count: number }>(
        "nexus_mods",
        props.appID,
        query,
        nextSort,
        20,
        0
      );
      if (!result.ok) {
        setError(result.error || "Unable to search Nexus Mods.");
        setMods([]);
        setTotalCount(0);
        return;
      }
      setMods(result.mods ?? []);
      setTotalCount(result.total_count ?? result.mods?.length ?? 0);
      if ((result.mods ?? []).length === 0) setMessage("No Vortex-compatible mods matched this search.");
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
      const result = await call<[string], { ok: boolean; error?: string; result?: { job?: Job } }>("add_captured_install", url);
      if (!result.ok) {
        setError(result.error || "Unable to add this Nexus file.");
        return;
      }
      const job = result.result?.job;
      if (job) showJobToast(job);
      setMessage(job?.message || `${file.name || file.file_name || mod.name} was sent to DMM.`);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyFileKey("");
    }
  }

  function cycleSort() {
    const next = nextNexusSort(sort);
    setSort(next);
    void searchMods(next);
  }

  useEffect(() => {
    void searchMods("downloads");
  }, []);

  return (
    <ConfirmModal
      strTitle={`Browse ${props.gameName}`}
      strDescription={
        <div style={{ display: "grid", gap: "12px", maxHeight: "70vh", overflowX: "hidden", overflowY: "auto", paddingRight: "6px", width: "100%" }}>
          <div style={{ alignItems: "end", display: "grid", gap: "8px", gridTemplateColumns: "minmax(0, 1fr) 120px 104px", width: "100%" }}>
            <TextField label="Search Nexus" value={query} bShowClearAction onChange={(event) => setQuery(event.currentTarget.value)} />
            <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={cycleSort} onClick={cycleSort} style={deckyCompactActionStyle("neutral")}>
              {nexusSortLabel(sort)}
            </Focusable>
            <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => searchMods()} onClick={() => searchMods()} style={deckyCompactActionStyle("neutral", busy)}>
              {busy ? "Searching" : "Search"}
            </Focusable>
          </div>
          <div style={{ color: "#a1a1aa", fontSize: "12px" }}>
            Showing {mods.length} of {compactNumber(totalCount)} Vortex-compatible Nexus result{totalCount === 1 ? "" : "s"} for {props.gameDomain}.
          </div>
          {error && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{error}</div>}
          {message && <div style={{ color: "#72e0a2", overflowWrap: "anywhere" }}>{message}</div>}
          {mods.map((mod) => {
            const files = filesByMod[mod.mod_id] ?? [];
            const filesOpen = selectedModID === mod.mod_id;
            return (
              <div key={mod.mod_id} className="dmm-sidebar-row" style={{ ...deckyCompositeRowStyle(filesOpen), background: "#111827" }}>
                <div style={{ alignItems: "start", display: "grid", gap: "10px", gridTemplateColumns: "112px minmax(0, 1fr)", width: "100%" }}>
                  <div style={{ background: "#030712", border: "1px solid #303741", borderRadius: "6px", height: "63px", overflow: "hidden", width: "112px" }}>
                    {mod.thumbnail_url ? (
                      <img src={mod.thumbnail_url} style={{ height: "100%", objectFit: "cover", width: "100%" }} />
                    ) : (
                      <div style={{ alignItems: "center", color: "#71717a", display: "flex", fontSize: "11px", height: "100%", justifyContent: "center", textAlign: "center" }}>No image</div>
                    )}
                  </div>
                  <div style={{ display: "grid", gap: "5px", minWidth: 0 }}>
                    <div style={{ ...deckyTwoLineTextStyle, fontWeight: 800 }}>{mod.name}</div>
                    <div style={{ color: "#d4d4d8", fontSize: "12px", lineHeight: 1.25, maxHeight: "3.75em", overflow: "hidden", overflowWrap: "anywhere" }}>{mod.summary}</div>
                    <div style={{ color: "#a1a1aa", display: "flex", flexWrap: "wrap", fontSize: "11px", gap: "8px" }}>
                      <span>v{mod.version || "unknown"}</span>
                      <span>{compactNumber(mod.downloads)} downloads</span>
                      <span>{compactNumber(mod.endorsements)} endorsements</span>
                    </div>
                  </div>
                </div>
                <Focusable className="dmm-action-grid" flow-children="row" style={deckyActionGridStyle(2)}>
                  <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => Navigation.NavigateToExternalWeb(mod.url)} onClick={() => Navigation.NavigateToExternalWeb(mod.url)} style={deckyCompactActionStyle("neutral")}>
                    Open Page
                  </Focusable>
                  <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => loadFiles(mod)} onClick={() => loadFiles(mod)} style={deckyCompactActionStyle("neutral", filesOpen || busyFileKey === `files:${mod.mod_id}`)}>
                    {busyFileKey === `files:${mod.mod_id}` ? "Loading" : filesOpen ? "Refresh Files" : "Files"}
                  </Focusable>
                </Focusable>
                {filesOpen && (
                  <div style={{ display: "grid", gap: "8px", width: "100%" }}>
                    {files.length === 0 && busyFileKey !== `files:${mod.mod_id}` && <div style={{ color: "#a1a1aa" }}>No files loaded yet.</div>}
                    {files.map((file) => {
                      const fileKey = `${mod.mod_id}:${file.file_id}`;
                      return (
                        <div key={file.file_id} style={{ border: "1px solid #303741", borderRadius: "6px", display: "grid", gap: "6px", padding: "8px", width: "100%" }}>
                          <div style={{ ...deckyTwoLineTextStyle, fontWeight: 800 }}>{file.name || file.file_name || `File ${file.file_id}`}</div>
                          <div style={{ color: "#a1a1aa", fontSize: "11px", overflowWrap: "anywhere" }}>
                            {file.file_name || "Nexus file"} · {formatBytes(file.size)} · v{file.version || "unknown"}
                          </div>
                          <Focusable className="dmm-focus-card" focusClassName="dmm-focus-card-focused" onActivate={() => installFile(mod, file)} onClick={() => installFile(mod, file)} style={deckyCompactActionStyle("neutral", busyFileKey === fileKey)}>
                            {busyFileKey === fileKey ? "Adding" : "Install"}
                          </Focusable>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })}
        </div>
      }
      strOKButtonText="Close"
      strCancelButtonText="Close"
      onOK={props.closeModal}
      onCancel={props.closeModal}
      closeModal={props.closeModal}
    />
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
  if (["job.updated", "profile_mods.changed", "deployment.changed", "install.changed"].includes(event.type)) {
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

function Content() {
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
  const [modsResult, setModsResult] = useState<string>("");
  const [modSearch, setModSearch] = useState<string>("");
  const [gameSearch, setGameSearch] = useState<string>("");
  const [gameSort, setGameSortState] = useState<GameSort>("recent");
  const [favoriteGameIDs, setFavoriteGameIDs] = useState<Set<string>>(new Set());
  const [gameRecent, setGameRecent] = useState<Record<string, number>>({});
  const [busyModID, setBusyModID] = useState<number | null>(null);
  const [focusedModID, setFocusedModID] = useState<number | null>(null);
  const [focusedGameID, setFocusedGameID] = useState<string>("");
  const [focusedProfileID, setFocusedProfileID] = useState<number | null>(null);
  const [focusedCandidateID, setFocusedCandidateID] = useState<number | null>(null);
  const [focusedModAction, setFocusedModAction] = useState<string>("");

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

  function clearSelectedDeckyGame() {
    setSelectedDeckyGameID("");
    setFocusedModID(null);
    setFocusedModAction("");
    setFocusedCandidateID(null);
    setModSearch("");
  }

  function handleDeckyTabCancel(event: GamepadEvent) {
    if (tab !== "mods" || !selectedDeckyGameID) return;
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
      return;
    }
    const [profilesResult, modsResult, candidatesResult] = await Promise.all([
      call<[string], { ok: boolean; error?: string; profiles: Profile[] }>("game_profiles", appID),
      call<[string], { ok: boolean; error?: string; mods: ManagedMod[] }>("game_mods", appID),
      call<[string], { ok: boolean; error?: string; candidates: InstallCandidate[] }>("game_install_candidates", appID)
    ]);
    if (!profilesResult.ok) {
      setError(profilesResult.error ?? "Unable to load profiles.");
      return;
    }
    if (!modsResult.ok) {
      setError(modsResult.error ?? "Unable to load mods.");
      return;
    }
    setDeckyProfiles(profilesResult.profiles);
    setDeckyMods(modsResult.mods);
    if (candidatesResult.ok) {
      setDeckyInstallCandidates(candidatesResult.candidates);
    } else {
      setDeckyInstallCandidates([]);
      setError(candidatesResult.error ?? "Unable to load installer items.");
    }
    void syncWorkshopStateForApp(appID);
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
      setSelectedDeckyGameID(appID);
      markDeckyGameRecent(appID);
      await loadDeckyGameState(appID);
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
      const applyMessage = result.apply?.message || "Profile changes applied. Restart the game if it is already running.";
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
        strDescription="DMM will remove blocked installer items and choice requests for this game. Downloaded archives are kept in the cache."
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

  async function openNexus(gameDomain: string | null = null) {
    try {
      setError("");
      const result = await call<[string | null], { ok: boolean; error?: string; url?: string }>("open_nexus", gameDomain);
      if (!result.ok) setError(result.error ?? "Unable to open Nexus.");
      if (result.url) Navigation.NavigateToExternalWeb(result.url);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
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
      { strTitle: "Browse Nexus Mods", bNeverPopOut: true, popupWidth: 760, popupHeight: 800 }
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
      logFrontendEvent("content unmounted");
    };
  }, []);

  useEffect(() => {
    const listener = (rawEvent: Event) => {
      const event = (rawEvent as CustomEvent<DomainEvent>).detail;
      if (event?.type === "ui.changed") {
        if (isUISettings(event.payload)) applyDeckyUIPreferencesFromUI(event.payload);
        return;
      }
      if (!event || tab !== "mods" || !selectedDeckyGameID || !status?.running) return;
      if (["job.updated", "profile_mods.changed", "deployment.changed", "install.changed", "launch.changed"].includes(event.type) && eventMatchesAppID(event, selectedDeckyGameID)) {
        void loadDeckyGameState(selectedDeckyGameID);
      }
    };
    window.addEventListener(DMM_EVENT_NAME, listener);
    return () => window.removeEventListener(DMM_EVENT_NAME, listener);
  }, [tab, selectedDeckyGameID, status?.running]);

  useEffect(() => {
    const syncRunningGame = () => {
      const running = currentRunningGame();
      setRunningGame(running);
      if (!running || tab !== "mods" || !managedGames.some((game) => game.app_id === running.app_id) || selectedDeckyGameID === running.app_id) return;
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
    if (!normalizedModSearch) return deckyMods;
    return deckyMods.filter((mod) =>
        [mod.name, mod.status, mod.source_game_domain, mod.source_mod_id, mod.source_file_id]
          .some((value) => String(value ?? "").toLowerCase().includes(normalizedModSearch))
      );
  }, [deckyMods, modSearch]);
  const focusedGameIndex = visibleManagedGames.findIndex((game) => game.app_id === focusedGameID);
  const focusedModIndex = visibleDeckyMods.findIndex((mod) => mod.id === focusedModID);
  const renderedManagedGames = windowedList(visibleManagedGames, focusedGameIndex, deckyGameListWindowSize);
  const renderedDeckyMods = windowedList(visibleDeckyMods, focusedModIndex, deckyModListWindowSize);
  const visibleManagedGameIDs = visibleManagedGames.map((game) => game.app_id).join("|");
  const visibleDeckyModIDs = visibleDeckyMods.map((mod) => String(mod.id)).join("|");

  useEffect(() => {
    if (selectedDeckyGameID || visibleManagedGames.length === 0) {
      if (focusedGameID && !visibleManagedGames.some((game) => game.app_id === focusedGameID)) {
        setFocusedGameID("");
      }
      return;
    }
    if (!focusedGameID || !visibleManagedGames.some((game) => game.app_id === focusedGameID)) {
      setFocusedGameID(visibleManagedGames[0].app_id);
    }
  }, [selectedDeckyGameID, visibleManagedGameIDs, focusedGameID]);

  useEffect(() => {
    if (!selectedDeckyGameID || visibleDeckyMods.length === 0) {
      if (focusedModID !== null) setFocusedModID(null);
      setFocusedModAction("");
      return;
    }
    if (focusedModID === null || !visibleDeckyMods.some((mod) => mod.id === focusedModID)) {
      setFocusedModID(visibleDeckyMods[0].id);
      setFocusedModAction("");
    }
  }, [selectedDeckyGameID, visibleDeckyModIDs, focusedModID]);

  function nextDirectionalIndex(currentIndex: number, length: number, direction: -1 | 1) {
    if (length <= 0) return -1;
    if (currentIndex < 0) return direction > 0 ? 0 : length - 1;
    return Math.max(0, Math.min(length - 1, currentIndex + direction));
  }

  function handleDeckyGameListDirection(event: GamepadEvent) {
    const button = event.detail.button;
    if (button !== GamepadButton.DIR_DOWN && button !== GamepadButton.DIR_UP) return;
    event.preventDefault();
    event.stopPropagation();
    const direction = button === GamepadButton.DIR_DOWN ? 1 : -1;
    const nextIndex = nextDirectionalIndex(visibleManagedGames.findIndex((game) => game.app_id === focusedGameID), visibleManagedGames.length, direction);
    const nextGame = visibleManagedGames[nextIndex];
    if (nextGame) {
      setFocusedGameID(nextGame.app_id);
    }
  }

  function handleDeckyModListDirection(event: GamepadEvent) {
    const button = event.detail.button;
    if (button !== GamepadButton.DIR_DOWN && button !== GamepadButton.DIR_UP) return;
    event.preventDefault();
    event.stopPropagation();
    const direction = button === GamepadButton.DIR_DOWN ? 1 : -1;
    const nextIndex = nextDirectionalIndex(visibleDeckyMods.findIndex((mod) => mod.id === focusedModID), visibleDeckyMods.length, direction);
    const nextMod = visibleDeckyMods[nextIndex];
    if (nextMod) {
      setFocusedModID(nextMod.id);
      setFocusedModAction("");
    }
  }

  const mainContent = (
    <>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={toggleServer}>
          {status?.running ? "Stop Server" : "Start Server"}
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={retryLaunchSetup} disabled={!status?.running}>
          Retry Launch Setup
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <a
          href="https://www.nexusmods.com"
          onClick={(event) => {
            event.preventDefault();
            openNexus();
          }}
          style={{ color: "#7dd3fc", display: "block", fontWeight: 800, padding: "10px 0", textDecoration: "underline" }}
        >
          Open Nexus Mods
        </a>
      </PanelSectionRow>
      <PanelSectionRow>
        <div>
          <div>Status: {status?.running ? "Running" : "Stopped"}</div>
          {status?.pid && <div>PID: {status.pid}</div>}
          <div>URL: {status?.url ?? "Unavailable"}</div>
          {status?.backend && <div>Games: {status.backend.game_count}</div>}
          {status?.backend && <div>Nexus: {status.backend.nexus.api_key_configured ? "Configured" : "Missing"}</div>}
          {launchResult && <div style={{ color: "#72e0a2", marginTop: "8px", overflowWrap: "anywhere" }}>{launchResult}</div>}
          {error && <div style={{ color: "#f87171", marginTop: "8px", overflowWrap: "anywhere" }}>{error}</div>}
          {status?.error && <div style={{ color: "#f87171", marginTop: "8px", overflowWrap: "anywhere" }}>{status.error}</div>}
        </div>
      </PanelSectionRow>
      <PanelSectionRow>
        <div style={{ display: "grid", gap: "10px", width: "100%" }}>
          <TextField label="Nexus URL" value={importUrl} bShowClearAction description="Paste a Nexus mod page URL or nxm:// link." onChange={(event) => setImportUrl(event.currentTarget.value)} />
          <ButtonItem layout="below" onClick={addCapturedInstall}>
            Add Nexus Link
          </ButtonItem>
          <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Captures the URL and starts the download. Install actions and choices appear in Action Center.</div>
          {importResult && <div style={{ color: "#72e0a2", overflowWrap: "anywhere" }}>{importResult}</div>}
        </div>
      </PanelSectionRow>
    </>
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
            {visibleManagedGames.length > renderedManagedGames.items.length && (
              <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>
                Showing {renderedManagedGames.start + 1}-{renderedManagedGames.end} of {renderedManagedGames.total}
              </div>
            )}
            <Focusable flow-children="column" navEntryPreferPosition={NavEntryPositionPreferences.FIRST} onGamepadDirection={handleDeckyGameListDirection} style={deckySidebarListStyle}>
              {renderedManagedGames.items.map((game, index) => {
                const absoluteIndex = renderedManagedGames.start + index;
                const focused = focusedGameID === game.app_id;
                const favorite = favoriteGameIDs.has(game.app_id);
                return (
                  <Focusable
                    key={game.app_id}
                    className="dmm-sidebar-surface dmm-sidebar-row"
                    data-dmm-game-id={game.app_id}
                    focusClassName="dmm-sidebar-row-focused"
                    focusWithinClassName="dmm-sidebar-row-focused"
                    flow-children="column"
                    noFocusRing
                    onActivate={() => selectDeckyGame(game.app_id)}
                    onOKButton={(event) => {
                      event.preventDefault();
                      event.stopPropagation();
                      void selectDeckyGame(game.app_id);
                    }}
                    onGamepadFocus={() => setFocusedGameID(game.app_id)}
                    onFocus={() => setFocusedGameID(game.app_id)}
                    onMouseEnter={() => setFocusedGameID(game.app_id)}
                    preferredFocus={focused}
                    style={deckyCompositeRowStyle(focused, favorite)}
                  >
                    <Focusable
                      className="dmm-focus-card"
                      focusClassName="dmm-focus-card-focused"
                      onActivate={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        selectDeckyGame(game.app_id);
                      }}
                      onClick={() => {
                        void selectDeckyGame(game.app_id);
                      }}
                      onOKButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        void selectDeckyGame(game.app_id);
                      }}
                      onGamepadFocus={() => setFocusedGameID(game.app_id)}
                      onFocus={() => setFocusedGameID(game.app_id)}
                      onMouseEnter={() => setFocusedGameID(game.app_id)}
                      preferredFocus={focused || (absoluteIndex === 0 && !focusedGameID)}
                      style={{
                        ...deckyFocusableCardStyle(focused, favorite),
                        display: "grid",
                        gap: "4px",
                        minHeight: "48px",
                        padding: "10px",
                      }}
                    >
                      <div style={deckyTwoLineTextStyle}>{game.name}</div>
                      <div style={{ color: favorite ? "#99f6e4" : "#a1a1aa", fontSize: "11px", fontWeight: 800 }}>
                        {favorite ? "Favorite" : `App ${game.app_id}`}
                      </div>
                    </Focusable>
                    <Focusable
                      className="dmm-focus-card"
                      focusClassName="dmm-focus-card-focused"
                      onActivate={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        toggleDeckyFavoriteGame(game.app_id);
                      }}
                      onClick={() => toggleDeckyFavoriteGame(game.app_id)}
                      onOKButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        toggleDeckyFavoriteGame(game.app_id);
                      }}
                      onGamepadFocus={() => setFocusedGameID(game.app_id)}
                      onFocus={() => setFocusedGameID(game.app_id)}
                      onMouseEnter={() => setFocusedGameID(game.app_id)}
                      style={deckyCompactActionStyle("neutral")}
                    >
                      {favorite ? "Unfavorite" : "Favorite"}
                    </Focusable>
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
            <div style={{ alignItems: "center", display: "flex", gap: "10px", width: "100%" }}>
              <img src={steamHeaderImage(selectedDeckyGameID)} style={{ borderRadius: "5px", height: "42px", objectFit: "cover", width: "74px" }} />
              <div style={{ minWidth: 0 }}>
                <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800, textTransform: "uppercase" }}>{selectedProfile ? `Profile: ${selectedProfile.name}` : "No profile"}</div>
                <div style={{ fontWeight: 800, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{selectedDeckyGame?.name ?? selectedDeckyGameID}</div>
              </div>
            </div>
          </PanelSectionRow>
          <PanelSectionRow>
            <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Toggling a mod applies the selected profile. Restart a running game to pick up changes.</div>
          </PanelSectionRow>
          <PanelSectionRow>
            <ButtonItem layout="below" onClick={clearSelectedDeckyGame}>
              Change Game
            </ButtonItem>
          </PanelSectionRow>
          <PanelSectionRow>
            {selectedNexusDomain ? (
              <div style={{ display: "grid", gap: "8px", width: "100%" }}>
                <ButtonItem layout="below" onClick={openDeckyNexusBrowser}>
                  Browse Nexus Mods
                </ButtonItem>
                <ButtonItem layout="below" onClick={() => openNexus(selectedNexusDomain)}>
                  Open Nexus Page
                </ButtonItem>
              </div>
            ) : (
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>No Nexus page is registered for this game yet.</div>
            )}
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
                      focusWithinClassName="dmm-sidebar-row-focused"
                      flow-children="column"
                      noFocusRing
                      onGamepadFocus={() => setFocusedCandidateID(candidate.id)}
                      onFocus={() => setFocusedCandidateID(candidate.id)}
                      onMouseEnter={() => setFocusedCandidateID(candidate.id)}
                      preferredFocus={focused}
                      style={deckyCompositeRowStyle(focused)}
                    >
                      <Focusable
                        className="dmm-focus-card"
                        focusClassName="dmm-focus-card-focused"
                        onActivate={(event) => {
                          event.preventDefault();
                          event.stopPropagation();
                          if (installer) openDeckyInstallerChoice(candidate);
                        }}
                        onClick={() => {
                          if (installer) openDeckyInstallerChoice(candidate);
                        }}
                        onGamepadFocus={() => setFocusedCandidateID(candidate.id)}
                        onFocus={() => setFocusedCandidateID(candidate.id)}
                        onMouseEnter={() => setFocusedCandidateID(candidate.id)}
                        style={{
                          ...deckyFocusableCardStyle(focused),
                          display: "grid",
                          gap: "5px",
                          padding: "10px"
                        }}
                      >
                        <div style={{ ...deckyTwoLineTextStyle, fontWeight: 800 }}>{candidate.name}</div>
                        <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                          {candidate.status === "blocked" ? "Blocked installer" : "Installer choices"} · {candidate.source_game_domain}/mods/{candidate.source_mod_id}/files/{candidate.source_file_id}
                        </div>
                        {candidate.reason && (
                          <div style={{ color: candidate.status === "blocked" ? "#fca5a5" : "#d4d4d8", fontSize: "11px", lineHeight: 1.2, minWidth: 0, overflowWrap: "anywhere" }}>
                            {candidate.reason}
                          </div>
                        )}
                      </Focusable>
                      <Focusable className="dmm-action-grid" flow-children="column" style={deckyActionGridStyle(1)}>
                        {installer && (
                          <Focusable
                            className="dmm-focus-card"
                            focusClassName="dmm-focus-card-focused"
                            onActivate={(event) => {
                              event.preventDefault();
                              event.stopPropagation();
                              openDeckyInstallerChoice(candidate);
                            }}
                            onClick={() => openDeckyInstallerChoice(candidate)}
                            style={deckyCompactActionStyle("neutral", focused)}
                          >
                            Open Choices
                          </Focusable>
                        )}
                        <Focusable
                          className="dmm-focus-card"
                          focusClassName="dmm-focus-card-focused"
                          onActivate={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            askClearDeckyInstallCandidates();
                          }}
                          onClick={askClearDeckyInstallCandidates}
                          style={deckyCompactActionStyle("danger")}
                        >
                          Clear Items
                        </Focusable>
                      </Focusable>
                    </Focusable>
                  );
                })}
              </div>
            </PanelSectionRow>
          )}
          {deckyMods.length > 0 && (
            <PanelSectionRow>
              <TextField label="Search Mods" value={modSearch} bShowClearAction onChange={(event) => setModSearch(event.currentTarget.value)} />
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
          {deckyMods.length === 0 && (
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
              {visibleDeckyMods.length > renderedDeckyMods.items.length && (
                <div style={{ color: "#a1a1aa", fontSize: "11px", fontWeight: 800, marginBottom: "8px" }}>
                  Showing {renderedDeckyMods.start + 1}-{renderedDeckyMods.end} of {renderedDeckyMods.total}
                </div>
              )}
              <Focusable flow-children="column" navEntryPreferPosition={NavEntryPositionPreferences.FIRST} onGamepadDirection={handleDeckyModListDirection} style={deckySidebarListStyle}>
                {renderedDeckyMods.items.map((mod) => {
                  const focused = focusedModID === mod.id;
                  const toggleActionID = `${mod.id}:toggle`;
                  return (
                    <Focusable
                      key={mod.id}
                      className="dmm-sidebar-surface dmm-sidebar-row"
                      data-dmm-mod-id={String(mod.id)}
                      focusClassName="dmm-sidebar-row-focused"
                      focusWithinClassName="dmm-sidebar-row-focused"
                      flow-children="column"
                      noFocusRing
                      onActivate={() => toggleDeckyMod(mod, !mod.enabled)}
                      onOKButton={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        void toggleDeckyMod(mod, !mod.enabled);
                      }}
                      onGamepadFocus={() => {
                        setFocusedModID(mod.id);
                        setFocusedModAction("");
                      }}
                      onFocus={() => {
                        setFocusedModID(mod.id);
                        setFocusedModAction("");
                      }}
                      onMouseEnter={() => {
                        setFocusedModID(mod.id);
                        setFocusedModAction("");
                      }}
                      preferredFocus={focused}
                      style={deckyCompositeRowStyle(focused, mod.enabled)}
                    >
                      <Focusable
                        className="dmm-focus-card"
                        focusClassName="dmm-focus-card-focused"
                        onActivate={(event) => {
                          event.preventDefault();
                          event.stopPropagation();
                          void toggleDeckyMod(mod, !mod.enabled);
                        }}
                        onClick={() => {
                          void toggleDeckyMod(mod, !mod.enabled);
                        }}
                        onOKButton={(event) => {
                          event.preventDefault();
                          event.stopPropagation();
                          void toggleDeckyMod(mod, !mod.enabled);
                        }}
                        onGamepadBlur={() => {
                          if (focusedModID === mod.id) setFocusedModAction("");
                        }}
                        onGamepadFocus={() => setFocusedModID(mod.id)}
                        onFocus={() => setFocusedModID(mod.id)}
                        onMouseEnter={() => setFocusedModID(mod.id)}
                        style={{
                          ...deckyFocusableCardStyle(focused, mod.enabled),
                          display: "grid",
                          gap: "6px",
                          maxWidth: "100%",
                          minHeight: "58px",
                          opacity: busyModID === mod.id ? 0.65 : 1,
                          padding: "10px",
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
                      </Focusable>
                      <Focusable
                        className="dmm-focus-card"
                        focusClassName="dmm-focus-card-focused"
                        onActivate={(event) => {
                          event.preventDefault();
                          event.stopPropagation();
                          void toggleDeckyMod(mod, !mod.enabled);
                        }}
                        onClick={() => {
                          void toggleDeckyMod(mod, !mod.enabled);
                        }}
                        onOKButton={(event) => {
                          event.preventDefault();
                          event.stopPropagation();
                          void toggleDeckyMod(mod, !mod.enabled);
                        }}
                        onGamepadFocus={() => {
                          setFocusedModID(mod.id);
                          setFocusedModAction(toggleActionID);
                        }}
                        onFocus={() => {
                          setFocusedModID(mod.id);
                          setFocusedModAction(toggleActionID);
                        }}
                        onMouseEnter={() => {
                          setFocusedModID(mod.id);
                          setFocusedModAction(toggleActionID);
                        }}
                        style={deckyCompactActionStyle("neutral", focusedModAction === toggleActionID)}
                      >
                        {mod.enabled ? "Disable" : "Enable"}
                      </Focusable>
                      <Focusable className="dmm-action-grid" flow-children="column" style={deckyActionGridStyle(1)}>
                        <Focusable
                          className="dmm-focus-card"
                          focusClassName="dmm-focus-card-focused"
                          onActivate={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            void reinstallDeckyMod(mod);
                          }}
                          onClick={() => {
                            void reinstallDeckyMod(mod);
                          }}
                          onOKButton={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            void reinstallDeckyMod(mod);
                          }}
                          onGamepadFocus={() => {
                            setFocusedModID(mod.id);
                            setFocusedModAction(`${mod.id}:reinstall`);
                          }}
                          onFocus={() => {
                            setFocusedModID(mod.id);
                            setFocusedModAction(`${mod.id}:reinstall`);
                          }}
                          onMouseEnter={() => {
                            setFocusedModID(mod.id);
                            setFocusedModAction(`${mod.id}:reinstall`);
                          }}
                          style={deckyCompactActionStyle("neutral", focusedModAction === `${mod.id}:reinstall`)}
                        >
                          Reinstall
                        </Focusable>
                        <Focusable
                          className="dmm-focus-card"
                          focusClassName="dmm-focus-card-focused"
                          onActivate={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            askRemoveDeckyMod(mod);
                          }}
                          onClick={() => askRemoveDeckyMod(mod)}
                          onOKButton={(event) => {
                            event.preventDefault();
                            event.stopPropagation();
                            askRemoveDeckyMod(mod);
                          }}
                          onGamepadFocus={() => {
                            setFocusedModID(mod.id);
                            setFocusedModAction(`${mod.id}:remove`);
                          }}
                          onFocus={() => {
                            setFocusedModID(mod.id);
                            setFocusedModAction(`${mod.id}:remove`);
                          }}
                          onMouseEnter={() => {
                            setFocusedModID(mod.id);
                            setFocusedModAction(`${mod.id}:remove`);
                          }}
                          style={deckyCompactActionStyle("danger", focusedModAction === `${mod.id}:remove`)}
                        >
                          Remove
                        </Focusable>
                      </Focusable>
                    </Focusable>
                  );
                })}
              </Focusable>
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
    { id: "main", title: "Manage", content: mainContent },
    { id: "mods", title: "Mods", content: modsContent },
    { id: "settings", title: "Settings", content: settingsContent },
    { id: "debug", title: "Debug", content: debugContent }
  ];
  const activeTabContent = tabItems.find((item) => item.id === tab)?.content ?? mainContent;
  const showDeckyTab = (next: Tab) => {
    if (next === tab) return;
    setTab(next);
    if (next === "mods") refreshDeckyMods();
  };
  const shiftDeckyTab = (direction: -1 | 1) => {
    const index = deckyTabOrder.indexOf(tab);
    const current = index >= 0 ? index : 0;
    const next = deckyTabOrder[(current + direction + deckyTabOrder.length) % deckyTabOrder.length];
    showDeckyTab(next);
  };
  const handleDeckyShellButtonDown = (event: CustomEvent<{ button: number }>) => {
    if (event.detail.button === GamepadButton.BUMPER_LEFT) {
      event.preventDefault();
      event.stopPropagation();
      shiftDeckyTab(-1);
    } else if (event.detail.button === GamepadButton.BUMPER_RIGHT) {
      event.preventDefault();
      event.stopPropagation();
      shiftDeckyTab(1);
    }
  };

  return (
    <PanelSection title="Decky Mod Manager">
      <style>{deckyRuntimeStyles}</style>
      <Focusable
        flow-children="column"
        navEntryPreferPosition={NavEntryPositionPreferences.PREFERRED_CHILD}
        onButtonDown={handleDeckyShellButtonDown}
        actionDescriptionMap={{
          [GamepadButton.BUMPER_LEFT]: "Previous Tab",
          [GamepadButton.BUMPER_RIGHT]: "Next Tab"
        }}
        style={deckyPanelFrameStyle}
      >
        <Focusable flow-children="row" noFocusRing style={deckyTabBarStyle}>
          {tabItems.map((item) => (
            <Focusable
              key={item.id}
              onActivate={() => showDeckyTab(item.id)}
              onClick={() => showDeckyTab(item.id)}
              preferredFocus={item.id === tab}
              style={deckyTabButtonStyle(tab === item.id)}
            >
              {item.title}
            </Focusable>
          ))}
        </Focusable>
        {deckyTabBody(activeTabContent, tab === "mods" && selectedDeckyGameID ? handleDeckyTabCancel : undefined)}
      </Focusable>
    </PanelSection>
  );
}

export default definePlugin(() => {
  startBackgroundMonitors();
  return {
    name: "Decky Mod Manager",
    titleView: <div className={staticClasses.Title}>Decky Mod Manager</div>,
    alwaysRender: true,
    content: <Content />,
    icon: <FaPowerOff />,
    onDismount() {
      stopBackgroundMonitors();
    }
  };
});
