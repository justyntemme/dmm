import {
  ButtonItem,
  PanelSection,
  PanelSectionRow,
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
  warning: string;
  error?: string;
};

type Dependency = {
  name: string;
  command: string;
  installed: boolean;
  path?: string;
  description: string;
};

type Tab = "server" | "dependencies";

const tabButton = (active: boolean): React.CSSProperties => ({
  minHeight: "28px",
  border: "0",
  borderRadius: "6px",
  padding: "0 8px",
  color: active ? "#08110d" : "#d4d4d8",
  background: active ? "#72e0a2" : "#2d333c",
  fontSize: "12px",
  fontWeight: 800
});

function Content() {
  const [tab, setTab] = useState<Tab>("server");
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [dependencies, setDependencies] = useState<Dependency[]>([]);
  const [error, setError] = useState<string>("");

  async function refresh() {
    try {
      setError("");
      setStatus(await call<[], BackendStatus>("status"));
      setDependencies(await call<[], Dependency[]>("dependencies"));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function toggleServer() {
    try {
      setError("");
      const method = status?.running ? "stop_server" : "start_server";
      setStatus(await call<[], BackendStatus>(method));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function setLanOnly(lanOnly: boolean) {
    try {
      setError("");
      const result = await call<[boolean], { ok: boolean; error?: string }>("set_lan_only", lanOnly);
      if (!result.ok) {
        setError(result.error ?? "Unable to update server settings.");
      }
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
        <div style={{ display: "flex", gap: "6px", alignItems: "center" }}>
          <button type="button" style={tabButton(tab === "server")} onClick={() => setTab("server")}>
            Server
          </button>
          <button type="button" style={tabButton(tab === "dependencies")} onClick={() => setTab("dependencies")}>
            Deps
          </button>
        </div>
      </PanelSectionRow>

      {tab === "server" ? (
        <>
          <PanelSectionRow>
            <ButtonItem layout="below" onClick={toggleServer}>
              {status?.running ? "Stop Server" : "Start Server"}
            </ButtonItem>
          </PanelSectionRow>
          <PanelSectionRow>
            <div>
              <div>Status: {status?.running ? "Running" : "Stopped"}</div>
              {status?.pid && <div>PID: {status.pid}</div>}
              <div>URL: {status?.url ?? "Unavailable"}</div>
              {status?.backend && <div>Games: {status.backend.game_count}</div>}
              {status?.backend && (
                <div>Nexus: {status.backend.nexus.api_key_configured ? "Configured" : "Missing"}</div>
              )}
              {status?.backend && <div>LAN only: {status.backend.lan_only ? "Enabled" : "Disabled"}</div>}
              {status?.logs && (
                <div style={{ color: "#a1a1aa", marginTop: "8px" }}>Logs: {status.logs.plugin}</div>
              )}
              {error && <div style={{ color: "#f87171", marginTop: "8px" }}>{error}</div>}
              {status?.error && <div style={{ color: "#f87171", marginTop: "8px" }}>{status.error}</div>}
            </div>
          </PanelSectionRow>
          <PanelSectionRow>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "8px" }}>
              <ButtonItem layout="below" onClick={() => setLanOnly(true)}>
                LAN Only
              </ButtonItem>
              <ButtonItem layout="below" onClick={() => setLanOnly(false)}>
                Allow Tunnel
              </ButtonItem>
            </div>
          </PanelSectionRow>
        </>
      ) : (
        <PanelSectionRow>
          <div style={{ maxHeight: "360px", overflowY: "auto", paddingRight: "4px" }}>
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
