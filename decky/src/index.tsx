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
import { CSSProperties, ReactNode, useEffect, useState } from "react";

declare const SteamClient:
  | {
      Apps?: {
        SetAppLaunchOptions?: (appid: number, launchOptions: string) => void;
      };
      GameSessions?: {
        RegisterForAppLifetimeNotifications?: (callback: (notification: { unAppID: number; bRunning: boolean }) => void) => { unregister?: () => void; Unregister?: () => void } | (() => void);
      };
    }
  | undefined;

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

type Tab = "main" | "mods" | "settings" | "debug";
type GameSort = "recent" | "az" | "za";

const deckyTabOrder: Tab[] = ["main", "mods", "settings", "debug"];

const deckyPanelFrameStyle: CSSProperties = {
  alignSelf: "stretch",
  boxSizing: "border-box",
  display: "flex",
  flexDirection: "column",
  gap: "8px",
  height: "calc(100vh - 112px)",
  maxHeight: "calc(100vh - 112px)",
  minHeight: 0,
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
    padding: "8px 6px",
    textAlign: "center",
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

function installToastBody(job: Job): string {
  if (job.status === "waiting") return job.message || "Open the phone or tablet UI to approve this install.";
  if (job.status === "running" || job.status === "queued") return job.message || "DMM is downloading or installing this mod.";
  if (job.status === "completed") return job.message || "The mod is ready in its profile.";
  if (job.status === "failed") return job.message || "Open DMM to review the error.";
  return job.message || job.title;
}

function showInstallToast(job: Job) {
  toaster.toast({
    title: job.status === "failed" ? "DMM install failed" : "Decky Mod Manager",
    body: installToastBody(job),
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

const notifiedInstallJobStates = new Map<string, string>();
const shownInstallerChoiceModals = new Set<string>();
const completedLaunchActions = new Set<string>();
const launchActionAttempts = new Map<string, number>();
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

function isInstallNotificationJob(job: Job) {
  return job.type === "pending-import" || job.type === "installer-choice";
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

async function maybeShowInstallToast(job: Job, { seed = false, source = "event" } = {}) {
  if (!isInstallNotificationJob(job)) return;
  const stateKey = `${job.status}:${job.message || ""}`;
  const previous = notifiedInstallJobStates.get(job.id);
  notifiedInstallJobStates.set(job.id, stateKey);
  const updatedAt = Date.parse(job.updated_at || "");
  const recent = Number.isFinite(updatedAt) && Date.now() - updatedAt < 120_000;
  if (previous !== stateKey && (!seed || recent) && ["waiting", "running", "completed", "failed"].includes(job.status)) {
    await logFrontendEvent("install job toast shown", { job_id: job.id, status: job.status, seed, recent, type: job.type, source });
    showInstallToast(job);
  }
}

async function seedInstallNotifications({ seed = false } = {}) {
  try {
    const result = await call<[], { ok: boolean; error?: string; jobs: Job[] }>("jobs");
    if (!result.ok) {
      await logFrontendEvent("install job seed returned not ok", { error: result.error || "" });
      return;
    }
    for (const job of result.jobs) {
      await maybeShowInstallToast(job, { seed, source: "seed" });
    }
  } catch (_err) {
    await logFrontendEvent("install job seed failed", { error: _err instanceof Error ? _err.message : String(_err) });
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
      if (result.result?.job) showInstallToast(result.result.job);
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
  if (!status?.backend?.install.auto_enable_installed_mods) {
    await logFrontendEvent("installer choice modal skipped because auto enable is off", { job_id: job.id });
    return;
  }
  const appID = String(job.payload?.app_id ?? "").trim();
  const candidateID = Number.parseInt(String(job.payload?.candidate_id ?? ""), 10);
  if (!appID || !Number.isFinite(candidateID)) return;
  const key = String(candidateID);
  if (shownInstallerChoiceModals.has(key)) return;
  shownInstallerChoiceModals.add(key);
  try {
    const result = await call<[string], { ok: boolean; error?: string; candidates: InstallCandidate[] }>("game_install_candidates", appID);
    if (!result.ok) {
      shownInstallerChoiceModals.delete(key);
      await logFrontendEvent("installer choice candidates load failed", { app_id: appID, candidate_id: candidateID, error: result.error || "" });
      showLaunchToast("DMM installer choices needed", "Open Decky Mod Manager or the phone UI to finish this installer.", false);
      return;
    }
    const candidate = result.candidates.find((item) => item.id === candidateID);
    if (!candidate || !installerForCandidate(candidate)) {
      shownInstallerChoiceModals.delete(key);
      await logFrontendEvent("installer choice candidate missing installer json", { app_id: appID, candidate_id: candidateID });
      showLaunchToast("DMM installer choices needed", "Open Decky Mod Manager or the phone UI to finish this installer.", false);
      return;
    }
    let modal: { Close: () => void } | null = null;
    const closeModal = () => {
      modal?.Close();
    };
    modal = showModal(
      <InstallerChoiceModal
        appID={appID}
        candidate={candidate}
        closeModal={closeModal}
        onApplied={() => {
          shownInstallerChoiceModals.delete(key);
          void seedInstallNotifications({ seed: true });
          void syncLaunchActions();
        }}
      />,
      window,
      { strTitle: "DMM Installer Choices", bNeverPopOut: true, popupWidth: 520, popupHeight: 720 }
    );
    await logFrontendEvent("installer choice modal opened", { app_id: appID, candidate_id: candidateID });
  } catch (err) {
    shownInstallerChoiceModals.delete(key);
    await logFrontendEvent("installer choice modal open failed", { app_id: appID, candidate_id: candidateID, error: err instanceof Error ? err.message : String(err) });
    showLaunchToast("DMM installer choices needed", "Open Decky Mod Manager or the phone UI to finish this installer.", false);
  }
}

async function handleDeckyDomainEvent(event: DomainEvent) {
  if (event.id > eventMonitorLastID) eventMonitorLastID = event.id;
  if (event.type === "jobs.snapshot" && Array.isArray(event.payload)) {
    for (const item of event.payload) {
      if (!isJob(item)) continue;
      await maybeShowInstallToast(item, { seed: true, source: "event-snapshot" });
      await maybeShowInstallerChoiceModal(item);
    }
  }
  if (event.type === "job.updated" && isJob(event.payload)) {
    await maybeShowInstallToast(event.payload, { source: "event" });
    if (event.payload.type === "installer-choice" && event.payload.status !== "waiting" && event.payload.payload?.candidate_id) {
      shownInstallerChoiceModals.delete(event.payload.payload.candidate_id);
    }
    await maybeShowInstallerChoiceModal(event.payload);
  }
  if (["job.updated", "profile_mods.changed", "deployment.changed", "install.changed"].includes(event.type)) {
    await syncLaunchActions();
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
  seedInstallNotifications({ seed: true });
  syncLaunchActions();
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
      return;
    }
    const [profilesResult, modsResult] = await Promise.all([
      call<[string], { ok: boolean; error?: string; profiles: Profile[] }>("game_profiles", appID),
      call<[string], { ok: boolean; error?: string; mods: ManagedMod[] }>("game_mods", appID)
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
      if (result.apply?.job) showInstallToast(result.apply.job);
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
      if (result.result?.job) showInstallToast(result.result.job);
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
        showInstallToast(job);
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
        await seedInstallNotifications({ seed: true });
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

  async function addPendingImport() {
    try {
      setError("");
      setImportResult("");
      const result = await call<[string], { ok: boolean; error?: string; result?: { job?: Job } }>("add_pending_import", importUrl);
      if (!result.ok) {
        setError(result.error ?? "Unable to capture Nexus link.");
        return;
      }
      setImportUrl("");
      const job = result.result?.job;
      setImportResult(job?.message || job?.title || "Nexus link captured.");
      if (job) {
        const stateKey = `${job.status}:${job.message || ""}`;
        notifiedInstallJobStates.set(job.id, stateKey);
        await logFrontendEvent("install job toast shown", { job_id: job.id, status: job.status, source: "decky-add-import" });
        showInstallToast(job as Job);
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
  const normalizedGameSearch = gameSearch.trim().toLowerCase();
  const visibleManagedGames = [...managedGames]
    .filter((game) => {
      if (!normalizedGameSearch) return true;
      return game.name.toLowerCase().includes(normalizedGameSearch) || game.app_id.includes(normalizedGameSearch);
    })
    .sort((a, b) => {
      const favoriteDelta = Number(favoriteGameIDs.has(b.app_id)) - Number(favoriteGameIDs.has(a.app_id));
      if (favoriteDelta !== 0) return favoriteDelta;
      if (gameSort === "az") return a.name.localeCompare(b.name);
      if (gameSort === "za") return b.name.localeCompare(a.name);
      const recentDelta = (gameRecent[b.app_id] ?? 0) - (gameRecent[a.app_id] ?? 0);
      if (recentDelta !== 0) return recentDelta;
      return a.name.localeCompare(b.name);
    });
  const normalizedModSearch = modSearch.trim().toLowerCase();
  const visibleDeckyMods = normalizedModSearch
    ? deckyMods.filter((mod) =>
        [mod.name, mod.status, mod.source_game_domain, mod.source_mod_id, mod.source_file_id]
          .some((value) => String(value ?? "").toLowerCase().includes(normalizedModSearch))
      )
    : deckyMods;

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
          <ButtonItem layout="below" onClick={addPendingImport}>
            Add Nexus Link
          </ButtonItem>
          <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Captures the URL and starts the download. Install approvals and choices appear in Action Center.</div>
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
          <div style={{ paddingRight: "4px", width: "100%" }}>
            <div style={{ fontWeight: 800, marginBottom: "8px" }}>Select Game</div>
            <TextField label="Search Games" value={gameSearch} bShowClearAction onChange={(event) => setGameSearch(event.currentTarget.value)} />
            <ButtonItem layout="below" onClick={cycleDeckyGameSort}>
              Sort: {gameSortLabel(gameSort)}
            </ButtonItem>
            {managedGames.length === 0 && <div style={{ color: "#a1a1aa" }}>No games loaded.</div>}
            {managedGames.length > 0 && visibleManagedGames.length === 0 && <div style={{ color: "#a1a1aa" }}>No games match this search.</div>}
            {visibleManagedGames.map((game) => (
              <div key={game.app_id} style={{ boxSizing: "border-box", display: "grid", gap: "6px", gridTemplateColumns: "minmax(0, 1fr) 40px", marginBottom: "8px", minWidth: 0, width: "100%" }}>
                <Focusable
                  onActivate={() => selectDeckyGame(game.app_id)}
                  onGamepadFocus={() => setFocusedGameID(game.app_id)}
                  onFocus={() => setFocusedGameID(game.app_id)}
                  onMouseEnter={() => setFocusedGameID(game.app_id)}
                  style={{
                    ...deckyFocusableCardStyle(focusedGameID === game.app_id),
                    fontWeight: 800,
                    padding: "10px",
                  }}
                >
                  <div style={deckyTwoLineTextStyle}>{game.name}</div>
                </Focusable>
                <Focusable
                  onActivate={() => toggleDeckyFavoriteGame(game.app_id)}
                  onGamepadFocus={() => setFocusedGameID(game.app_id)}
                  onFocus={() => setFocusedGameID(game.app_id)}
                  onMouseEnter={() => setFocusedGameID(game.app_id)}
                  style={{
                    alignItems: "center",
                    background: favoriteGameIDs.has(game.app_id) ? "#0f766e" : "#1f2937",
                    border: `1px solid ${focusedGameID === game.app_id ? "#7dd3fc" : "#374151"}`,
                    borderRadius: "6px",
                    boxSizing: "border-box",
                    color: "#f8fafc",
                    display: "flex",
                    fontSize: "18px",
                    fontWeight: 900,
                    justifyContent: "center",
                    minHeight: "38px",
                    minWidth: 0,
                    overflow: "hidden",
                    width: "40px"
                  }}
                >
                  {favoriteGameIDs.has(game.app_id) ? "★" : "☆"}
                </Focusable>
              </div>
            ))}
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
              <ButtonItem layout="below" onClick={() => openNexus(selectedNexusDomain)}>
                Open Nexus Mods
              </ButtonItem>
            ) : (
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>No Nexus page is registered for this game yet.</div>
            )}
          </PanelSectionRow>
          {deckyMods.length > 0 && (
            <PanelSectionRow>
              <TextField label="Search Mods" value={modSearch} bShowClearAction onChange={(event) => setModSearch(event.currentTarget.value)} />
            </PanelSectionRow>
          )}
          {deckyProfiles.length > 1 && (
            <PanelSectionRow>
              <div style={{ width: "100%" }}>
                <div style={{ fontWeight: 800, marginBottom: "8px" }}>Profile</div>
                {deckyProfiles.map((profile) => (
                  <Focusable
                    key={profile.id}
                    onActivate={() => selectDeckyProfile(profile)}
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
              <Focusable flow-children="column" style={{ boxSizing: "border-box", display: "grid", gap: "8px", paddingRight: "4px", width: "100%" }}>
                {visibleDeckyMods.map((mod) => {
                  const focused = focusedModID === mod.id;
                  return (
                    <div key={mod.id} style={{ boxSizing: "border-box", display: "grid", gap: "6px", minWidth: 0, width: "100%" }}>
                      <Focusable
                        onActivate={() => toggleDeckyMod(mod, !mod.enabled)}
                        onGamepadFocus={() => setFocusedModID(mod.id)}
                        onFocus={() => setFocusedModID(mod.id)}
                        onMouseEnter={() => setFocusedModID(mod.id)}
                        style={{
                          ...deckyFocusableCardStyle(focused),
                          alignItems: "start",
                          display: "grid",
                          gap: "10px",
                          gridTemplateColumns: "minmax(0, 1fr) 44px",
                          minHeight: "58px",
                          opacity: busyModID === mod.id ? 0.65 : 1,
                          padding: "10px",
                        }}
                      >
                        <div style={{ flex: 1, minWidth: 0 }}>
                          <div style={{ ...deckyTwoLineTextStyle, color: "#f8fafc", fontWeight: 800 }}>{mod.name}</div>
                          <div style={{ color: "#a1a1aa", fontSize: "11px", lineHeight: 1.2, marginTop: "4px", minWidth: 0, overflowWrap: "anywhere" }}>
                            {mod.enabled ? "Enabled" : "Disabled"} · Priority {mod.priority} · {deckyModStateLabel(mod)}
                          </div>
                        </div>
                        <div
                          style={{
                            alignItems: "center",
                            background: mod.enabled ? "#0f766e" : "#3f3f46",
                            borderRadius: "999px",
                            display: "flex",
                            height: "22px",
                            justifyContent: mod.enabled ? "flex-end" : "flex-start",
                            justifySelf: "end",
                            padding: "2px",
                            width: "42px"
                          }}
                        >
                          <div style={{ background: "#f8fafc", borderRadius: "999px", height: "18px", width: "18px" }} />
                        </div>
                      </Focusable>
                      {focused && (
                        <Focusable flow-children="row" style={{ boxSizing: "border-box", display: "grid", gap: "6px", gridTemplateColumns: "minmax(0, 1fr) minmax(0, 1fr)", minWidth: 0, width: "100%" }}>
                          <Focusable
                            onActivate={() => reinstallDeckyMod(mod)}
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
                            onActivate={() => askRemoveDeckyMod(mod)}
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
                      )}
                    </div>
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
          <div>Install captured downloads: {status?.backend?.install.auto_install_captured_downloads ? "Automatic" : "Approval required"}</div>
          <div>Enable installed mods: {status?.backend?.install.auto_enable_installed_mods ? "Automatic" : "Manual"}</div>
          <div>NXM handler: {nxm?.registered ? "Registered" : "Not registered"}</div>
        </div>
      </PanelSectionRow>
      <PanelSectionRow>
        <ToggleField
          label="Auto-install captured downloads"
          description="NXM links always download immediately. This skips phone approval for the local install step."
          checked={status?.backend?.install.auto_install_captured_downloads ?? false}
          disabled={!status?.running}
          onChange={setAutoInstallCapturedDownloads}
        />
      </PanelSectionRow>
      <PanelSectionRow>
        <ToggleField
          label="Auto-enable installed mods"
          description="New installs are enabled and deployed automatically when there are no conflicts."
          checked={status?.backend?.install.auto_enable_installed_mods ?? false}
          disabled={!status?.running}
          onChange={setAutoEnableInstalledMods}
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
