import {
  ButtonItem,
  Navigation,
  PanelSection,
  PanelSectionRow,
  TextField,
  ToggleField,
  definePlugin,
  staticClasses
} from "@decky/ui";
import { call, toaster } from "@decky/api";
import { FaPowerOff } from "react-icons/fa";
import { useEffect, useRef, useState } from "react";

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

type Tab = "main" | "settings" | "debug";

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
  const seenJobStates = useRef<Map<string, string>>(new Map());
  const appliedLaunchActions = useRef<Map<string, string>>(new Map());

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

  async function toggleServer() {
    try {
      setError("");
      const method = status?.running ? "stop_server" : "start_server";
      setStatus(await call<[], BackendStatus>(method));
      await refresh();
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
        seenJobStates.current.set(job.id, job.status);
        showInstallToast(job as Job);
      }
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function pollInstallJobs({ seed = false } = {}) {
    try {
      const result = await call<[], { ok: boolean; error?: string; jobs: Job[] }>("jobs");
      if (!result.ok) return;
      for (const job of result.jobs) {
        if (job.type !== "pending-import") continue;
        const stateKey = `${job.status}:${job.message || ""}`;
        const previous = seenJobStates.current.get(job.id);
        seenJobStates.current.set(job.id, stateKey);
        if (!seed && previous !== stateKey && ["waiting", "running", "completed", "failed"].includes(job.status)) {
          showInstallToast(job);
        }
      }
    } catch (_err) {
      // Status polling is best-effort; the Debug view exposes detailed backend errors.
    }
  }

  async function pollLaunchActions() {
    try {
      if (!status?.running) return;
      const result = await call<[], { ok: boolean; error?: string; actions: LaunchStatus[] }>("launch_actions");
      if (!result.ok) return;
      for (const launchStatus of result.actions) {
        const action = launchStatus.action;
        if (!action || action.type !== "set-steam-launch-options") continue;
        if (!launchStatus.can_configure || launchStatus.configured) continue;
        const actionKey = `${action.app_id}:${action.tool_id}:${action.desired_options}`;
        if (appliedLaunchActions.current.get(actionKey) === "applied") continue;
        const appid = Number.parseInt(action.app_id, 10);
        const steamApps = typeof SteamClient !== "undefined" ? SteamClient?.Apps : undefined;
        if (!Number.isFinite(appid) || typeof steamApps?.SetAppLaunchOptions !== "function") {
          const message = "Steam launch-option API is unavailable in this Decky context.";
          appliedLaunchActions.current.set(actionKey, "failed");
          setLaunchResult(message);
          showLaunchToast("DMM launch tool failed", message, true);
          await call<[string, Record<string, string | boolean>], { ok: boolean }>("record_launch_action", action.app_id, {
            applied: false,
            error: message,
            source: "decky-auto"
          });
          continue;
        }
        try {
          steamApps.SetAppLaunchOptions(appid, action.desired_options);
          appliedLaunchActions.current.set(actionKey, "applied");
          const toolName = launchStatus.tool?.name || action.tool_id;
          const message = `${toolName} launch option configured for ${launchStatus.app_id}.`;
          setLaunchResult(message);
          showLaunchToast("DMM launch tool configured", message);
          await call<[string, Record<string, string | boolean>], { ok: boolean }>("record_launch_action", action.app_id, {
            applied: true,
            current_options: action.desired_options,
            source: "decky-auto"
          });
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          appliedLaunchActions.current.set(actionKey, "failed");
          setLaunchResult(message);
          showLaunchToast("DMM launch tool failed", message, true);
          await call<[string, Record<string, string | boolean>], { ok: boolean }>("record_launch_action", action.app_id, {
            applied: false,
            error: message,
            source: "decky-auto"
          });
        }
      }
    } catch (_err) {
      // Launch action polling is best-effort; backend diagnostics expose details.
    }
  }

  useEffect(() => {
    refresh();
    pollInstallJobs({ seed: true });
    const interval = window.setInterval(() => {
      pollInstallJobs();
      pollLaunchActions();
    }, 4000);
    return () => window.clearInterval(interval);
  }, [status?.running]);

  return (
    <PanelSection title="Decky Mod Manager">
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setTab("main")}>
          Manage
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
        </>
      )}

      {tab === "debug" && (
        <>
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
  return {
    name: "Decky Mod Manager",
    titleView: <div className={staticClasses.Title}>Decky Mod Manager</div>,
    alwaysRender: true,
    content: <Content />,
    icon: <FaPowerOff />,
    onDismount() {}
  };
});
