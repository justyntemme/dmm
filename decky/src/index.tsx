import {
  ButtonItem,
  Navigation,
  PanelSection,
  PanelSectionRow,
  TextField,
  ToggleField,
  staticClasses
} from "@decky/ui";
import { call, definePlugin, toaster } from "@decky/api";
import { FaPowerOff } from "react-icons/fa";
import { useEffect, useState } from "react";

declare const SteamClient:
  | {
      Apps?: {
        SetAppLaunchOptions?: (appid: number, launchOptions: string) => void;
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
      auto_deploy: boolean;
      auto_approve_downloads: boolean;
    };
  } | null;
  logs?: {
    plugin: string;
    backend: string;
  };
  error?: string;
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

type ManagedGame = {
  app_id: string;
  name: string;
  state: string;
  nexus_domains?: string[];
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

function installToastBody(job: Job): string {
  if (job.status === "waiting") return job.message || "Open the phone or tablet UI to approve this download.";
  if (job.status === "running" || job.status === "queued") return job.message || "DMM is downloading and preparing this mod.";
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
const completedLaunchActions = new Set<string>();
const launchActionAttempts = new Map<string, number>();
let backgroundMonitorInterval: number | null = null;

type LaunchResultSink = (message: string) => void;

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

async function pollInstallJobs({ seed = false } = {}) {
  try {
    const result = await call<[], { ok: boolean; error?: string; jobs: Job[] }>("jobs");
    if (!result.ok) {
      await logFrontendEvent("install job poll returned not ok", { error: result.error || "" });
      return;
    }
    for (const job of result.jobs) {
      if (!isInstallNotificationJob(job)) continue;
      const stateKey = `${job.status}:${job.message || ""}`;
      const previous = notifiedInstallJobStates.get(job.id);
      notifiedInstallJobStates.set(job.id, stateKey);
      const updatedAt = Date.parse(job.updated_at || "");
      const recent = Number.isFinite(updatedAt) && Date.now() - updatedAt < 120_000;
      if (previous !== stateKey && (!seed || recent) && ["waiting", "running", "completed", "failed"].includes(job.status)) {
        await logFrontendEvent("install job toast shown", { job_id: job.id, status: job.status, seed, recent, type: job.type });
        showInstallToast(job);
      }
    }
  } catch (_err) {
    await logFrontendEvent("install job poll failed", { error: _err instanceof Error ? _err.message : String(_err) });
  }
}

async function applyLaunchActionThroughBackend(action: LaunchAction, source: string, sink?: LaunchResultSink): Promise<boolean> {
  try {
    await logFrontendEvent("backend launch action requested", { app_id: action.app_id, tool_id: action.tool_id, source });
    const result = await call<
      [string],
      { ok: boolean; error?: string; result?: { applied?: boolean; status?: LaunchStatus; job?: Job } }
    >("apply_launch_action", action.app_id);
    if (!result.ok) {
      const message = result.error || "Backend launch setup did not complete.";
      sink?.(message);
      showLaunchToast("DMM launch tool failed", message, true);
      await logFrontendEvent("backend launch action failed", { app_id: action.app_id, tool_id: action.tool_id, source, error: message });
      return false;
    }
    const configured = Boolean(result.result?.applied || result.result?.status?.configured);
    const message = configured ? "Launch tool configured." : "Launch setup ran, but DMM still sees it as pending.";
    sink?.(message);
    showLaunchToast(configured ? "DMM launch tool configured" : "DMM launch tool needs review", message, !configured);
    await logFrontendEvent("backend launch action completed", { app_id: action.app_id, tool_id: action.tool_id, source, configured });
    return configured;
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    sink?.(message);
    showLaunchToast("DMM launch tool failed", message, true);
    await logFrontendEvent("backend launch action threw", { app_id: action.app_id, tool_id: action.tool_id, source, error: message });
    return false;
  }
}

async function pollLaunchActions(options: { force?: boolean; sink?: LaunchResultSink } = {}) {
  try {
    const result = await call<[], { ok: boolean; error?: string; actions: LaunchStatus[] }>("launch_actions");
    if (!result.ok) {
      await logFrontendEvent("launch action poll returned not ok", { error: result.error || "" });
      return;
    }
    if (result.actions.length > 0) {
      await logFrontendEvent("launch action poll found actions", { count: result.actions.length });
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
        if (await applyLaunchActionThroughBackend(action, "steam-api-unavailable", options.sink)) {
          completedLaunchActions.add(actionKey);
        }
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
        if (await applyLaunchActionThroughBackend(action, "steam-api-not-verified", options.sink)) {
          completedLaunchActions.add(actionKey);
        }
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
    await logFrontendEvent("launch action poll failed", { error: _err instanceof Error ? _err.message : String(_err) });
  }
}

function startBackgroundMonitors() {
  if (backgroundMonitorInterval !== null) return;
  logFrontendEvent("background monitors started");
  pollInstallJobs({ seed: true });
  pollLaunchActions();
  backgroundMonitorInterval = window.setInterval(() => {
    pollInstallJobs();
    pollLaunchActions();
  }, 4000);
}

function stopBackgroundMonitors() {
  if (backgroundMonitorInterval === null) return;
  window.clearInterval(backgroundMonitorInterval);
  backgroundMonitorInterval = null;
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
  const [deckyProfiles, setDeckyProfiles] = useState<Profile[]>([]);
  const [deckyMods, setDeckyMods] = useState<ManagedMod[]>([]);
  const [modsResult, setModsResult] = useState<string>("");
  const [busyModID, setBusyModID] = useState<number | null>(null);

  async function refresh() {
    try {
      setError("");
      setStatus(await call<[], BackendStatus>("status"));
      setDependencies(await call<[], Dependency[]>("dependencies"));
      setNXM(await call<[], NXMStatus>("nxm_status"));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
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
      const selected = appID || selectedDeckyGameID;
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
      setSelectedDeckyGameID(appID);
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
      const result = await call<[number], { ok: boolean; error?: string }>("set_default_profile", profile.id);
      if (!result.ok) {
        setError(result.error ?? "Unable to select profile.");
        return;
      }
      await loadDeckyGameState(selectedDeckyGameID);
      setModsResult("Profile selected. Restart the game for changes to affect a running session.");
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
      const result = await call<[string, number, number, boolean], { ok: boolean; error?: string; mod?: ManagedMod; deploy?: { job?: Job; message?: string } }>(
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
      setModsResult(result.deploy?.job?.message || result.deploy?.message || "Profile changes applied. Restart the game if it is already running.");
      if (result.deploy?.job) showInstallToast(result.deploy.job);
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
        await pollInstallJobs({ seed: true });
        await pollLaunchActions();
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

  async function setAutoAcceptDownloads(autoAccept: boolean) {
    try {
      setError("");
      const result = await call<[boolean], { ok: boolean; error?: string; status?: unknown }>("set_auto_accept_downloads", autoAccept);
      if (!result.ok) setError(result.error ?? "Unable to update download settings.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function openNexus() {
    try {
      setError("");
      const result = await call<[string | null], { ok: boolean; error?: string; url?: string }>("open_nexus", null);
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
        setError(result.error ?? "Unable to add install request.");
        return;
      }
      setImportUrl("");
      const job = result.result?.job;
      setImportResult(job?.message || job?.title || "Install request added.");
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
    await pollLaunchActions({ force: true, sink: setLaunchResult });
    await refresh();
  }

  useEffect(() => {
    logFrontendEvent("content mounted");
    refresh();
    return () => {
      logFrontendEvent("content unmounted");
    };
  }, []);

  return (
    <PanelSection title="Decky Mod Manager">
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setTab("main")}>
          Manage
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem
          layout="below"
          onClick={() => {
            setTab("mods");
            refreshDeckyMods();
          }}
        >
          Mods
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setTab("settings")}>
          Settings
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setTab("debug")}>
          Debug
        </ButtonItem>
      </PanelSectionRow>

      {tab === "main" && (
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
              <TextField
                label="Nexus URL"
                value={importUrl}
                bShowClearAction
                description="Paste a Nexus mod page URL or nxm:// link."
                onChange={(event) => setImportUrl(event.currentTarget.value)}
              />
              <ButtonItem layout="below" onClick={addPendingImport}>
                Add Install Request
              </ButtonItem>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>
                Adds the URL to Install Requests for phone or tablet approval.
              </div>
              {importResult && <div style={{ color: "#72e0a2", overflowWrap: "anywhere" }}>{importResult}</div>}
            </div>
          </PanelSectionRow>
        </>
      )}

      {tab === "mods" && (
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
          {status?.running && !selectedDeckyGameID && (
            <PanelSectionRow>
              <div style={{ maxHeight: "360px", overflowY: "auto", paddingRight: "4px", width: "100%" }}>
                <div style={{ fontWeight: 800, marginBottom: "8px" }}>Select Game</div>
                {managedGames.length === 0 && <div style={{ color: "#a1a1aa" }}>No games loaded.</div>}
                {managedGames.map((game) => (
                  <button
                    key={game.app_id}
                    type="button"
                    onClick={() => selectDeckyGame(game.app_id)}
                    style={{
                      background: "#1f2937",
                      border: "1px solid #374151",
                      borderRadius: "6px",
                      color: "#f8fafc",
                      display: "block",
                      fontWeight: 800,
                      marginBottom: "8px",
                      padding: "10px",
                      textAlign: "left",
                      width: "100%"
                    }}
                  >
                    {game.name}
                  </button>
                ))}
              </div>
            </PanelSectionRow>
          )}
          {status?.running && selectedDeckyGameID && (
            <>
              <PanelSectionRow>
                <div>
                  <div style={{ fontWeight: 800 }}>{managedGames.find((game) => game.app_id === selectedDeckyGameID)?.name ?? selectedDeckyGameID}</div>
                  <div style={{ color: "#a1a1aa" }}>Changes apply to the selected profile. Restart the game for a running session to pick them up.</div>
                </div>
              </PanelSectionRow>
              <PanelSectionRow>
                <ButtonItem layout="below" onClick={() => setSelectedDeckyGameID("")}>
                  Change Game
                </ButtonItem>
              </PanelSectionRow>
              {deckyProfiles.length > 1 && (
                <PanelSectionRow>
                  <div style={{ maxHeight: "180px", overflowY: "auto", width: "100%" }}>
                    <div style={{ fontWeight: 800, marginBottom: "8px" }}>Profile</div>
                    {deckyProfiles.map((profile) => (
                      <button
                        key={profile.id}
                        type="button"
                        onClick={() => selectDeckyProfile(profile)}
                        style={{
                          background: profile.is_default ? "#0f766e" : "#1f2937",
                          border: "1px solid #374151",
                          borderRadius: "6px",
                          color: "#f8fafc",
                          display: "block",
                          fontWeight: 800,
                          marginBottom: "8px",
                          padding: "10px",
                          textAlign: "left",
                          width: "100%"
                        }}
                      >
                        {profile.name}
                      </button>
                    ))}
                  </div>
                </PanelSectionRow>
              )}
              {deckyMods.length === 0 && (
                <PanelSectionRow>
                  <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>No profile mods yet. Add and approve Nexus downloads from the phone/tablet UI or the Decky paste field.</div>
                </PanelSectionRow>
              )}
              {deckyMods.map((mod) => (
                <PanelSectionRow key={mod.id}>
                  <ToggleField
                    label={mod.name}
                    description={`${mod.enabled ? "Enabled" : "Disabled"} · Priority ${mod.priority} · ${mod.source_game_domain}/mods/${mod.source_mod_id}/files/${mod.source_file_id}`}
                    checked={mod.enabled}
                    disabled={busyModID === mod.id}
                    onChange={(checked) => toggleDeckyMod(mod, checked)}
                  />
                </PanelSectionRow>
              ))}
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
      )}

      {tab === "settings" && (
        <>
          <PanelSectionRow>
            <div>
              <div style={{ fontWeight: 800, marginBottom: "6px" }}>Server Access</div>
              <div>LAN only: {status?.backend?.lan_only ? "Enabled" : "Disabled"}</div>
              <div>Download requests: {status?.backend?.install.auto_approve_downloads ? "Auto-accepted" : "Approval required"}</div>
              <div>NXM handler: {nxm?.registered ? "Registered" : "Not registered"}</div>
            </div>
          </PanelSectionRow>
          <PanelSectionRow>
            <ToggleField
              label="Auto-accept download requests"
              description="Captured Nexus links start downloading immediately. Keep this off when you want phone approval."
              checked={status?.backend?.install.auto_approve_downloads ?? false}
              disabled={!status?.running}
              onChange={setAutoAcceptDownloads}
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
      )}

      {tab === "debug" && (
        <>
          <PanelSectionRow>
            <div style={{ maxHeight: "320px", overflowY: "auto", paddingRight: "4px", width: "100%" }}>
              <div style={{ fontWeight: 800, marginBottom: "8px" }}>Dependencies</div>
              {dependencies.map((dep) => (
                <div key={dep.command} style={{ marginBottom: "10px", borderBottom: "1px solid #303741", paddingBottom: "8px" }}>
                  <div style={{ color: dep.installed ? "#72e0a2" : "#f87171", fontWeight: 800 }}>
                    {dep.name}: {dep.installed ? "Installed" : "Missing"}
                  </div>
                  <div style={{ color: "#a1a1aa", overflowWrap: "anywhere", lineHeight: 1.25 }}>
                    {dep.path ?? dep.command}
                  </div>
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
                <div style={{ display: "grid", gap: "10px", marginTop: "10px", maxHeight: "420px", overflowY: "auto", width: "100%" }}>
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
      )}

      <PanelSectionRow>
        <ButtonItem layout="below" onClick={refresh}>
          Refresh
        </ButtonItem>
      </PanelSectionRow>
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
