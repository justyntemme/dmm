import {
  ButtonItem,
  Navigation,
  PanelSection,
  PanelSectionRow,
  TextField,
  definePlugin,
  staticClasses
} from "@decky/ui";
import { call } from "@decky/api";
import { FaPowerOff } from "react-icons/fa";
import { useEffect, useState } from "react";

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

type Tab = "server" | "add" | "dependencies" | "nxm";

function Content() {
  const [tab, setTab] = useState<Tab>("server");
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [dependencies, setDependencies] = useState<Dependency[]>([]);
  const [nxm, setNXM] = useState<NXMStatus | null>(null);
  const [importUrl, setImportUrl] = useState<string>("");
  const [importResult, setImportResult] = useState<string>("");
  const [error, setError] = useState<string>("");

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

  async function openNexus() {
    const result = await call<[string | null], { ok: boolean; error?: string; url?: string }>("open_nexus", null);
    if (!result.ok) setError(result.error ?? "Unable to open Nexus.");
    if (result.url) Navigation.NavigateToExternalWeb(result.url);
  }

  async function registerNXM() {
    const result = await call<[], { ok: boolean; error?: string; status: NXMStatus }>("register_nxm_handler");
    setNXM(result.status);
    if (!result.ok) setError(result.error ?? "Unable to register NXM handler.");
  }

  async function testNXM() {
    const result = await call<[], { ok: boolean; error?: string }>("test_nxm_handler");
    if (!result.ok) setError(result.error ?? "Unable to run NXM handler.");
    await refresh();
  }

  async function testNXMDispatch() {
    const result = await call<[], { ok: boolean; error?: string }>("test_nxm_dispatch");
    if (!result.ok) setError(result.error ?? "Unable to dispatch test NXM link.");
    await refresh();
  }

  async function addPendingImport() {
    try {
      setError("");
      setImportResult("");
      const result = await call<[string], { ok: boolean; error?: string; result?: { job?: { title?: string; status?: string; message?: string } } }>("add_pending_import", importUrl);
      if (!result.ok) {
        setError(result.error ?? "Unable to add install request.");
        return;
      }
      setImportUrl("");
      const job = result.result?.job;
      setImportResult(job?.message || job?.title || "Install request added.");
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  return (
    <PanelSection title="Decky Mod Manager">
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setTab("server")}>
          Server
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setTab("add")}>
          Add Plugin
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setTab("dependencies")}>
          Dependencies
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={() => setTab("nxm")}>
          NXM Handler
        </ButtonItem>
      </PanelSectionRow>

      {tab === "server" && (
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
              {status?.backend && <div>LAN only: {status.backend.lan_only ? "Enabled" : "Disabled"}</div>}
              {status?.logs && <div style={{ color: "#a1a1aa", marginTop: "8px", overflowWrap: "anywhere" }}>Logs: {status.logs.plugin}</div>}
              {error && <div style={{ color: "#f87171", marginTop: "8px" }}>{error}</div>}
              {status?.error && <div style={{ color: "#f87171", marginTop: "8px" }}>{status.error}</div>}
            </div>
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

      {tab === "add" && (
        <>
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
              {error && <div style={{ color: "#f87171", overflowWrap: "anywhere" }}>{error}</div>}
            </div>
          </PanelSectionRow>
        </>
      )}

      {tab === "dependencies" && (
        <PanelSectionRow>
          <div style={{ maxHeight: "360px", overflowY: "auto", paddingRight: "4px", width: "100%" }}>
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
            {error && <div style={{ color: "#f87171", marginTop: "8px" }}>{error}</div>}
          </div>
        </PanelSectionRow>
      )}

      {tab === "nxm" && (
        <>
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
            <div>
              <div>Registered: {nxm?.registered ? "Yes" : "No"}</div>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Current: {nxm?.current_handler || "None"}</div>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>Protocol: {nxm?.protocol_handler || "None"}</div>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>xdg-mime: {nxm?.xdg_handler || "Unknown"}</div>
              <div style={{ color: "#a1a1aa", overflowWrap: "anywhere" }}>File: {nxm?.desktop_path || "Unknown"}</div>
              {error && <div style={{ color: "#f87171", marginTop: "8px" }}>{error}</div>}
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
