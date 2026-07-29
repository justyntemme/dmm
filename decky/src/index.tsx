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
  logs?: {
    plugin: string;
    backend: string;
  };
  warning: string;
  error?: string;
};

function Content() {
  const [status, setStatus] = useState<BackendStatus | null>(null);
  const [error, setError] = useState<string>("");

  async function refresh() {
    try {
      setError("");
      setStatus(await call<[], BackendStatus>("status"));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }

  async function toggle() {
    try {
      setError("");
      const method = status?.running ? "stop_server" : "start_server";
      setStatus(await call<[], BackendStatus>(method));
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
        <ButtonItem layout="below" onClick={toggle}>
          {status?.running ? "Stop Server" : "Start Server"}
        </ButtonItem>
      </PanelSectionRow>
      <PanelSectionRow>
          <div>
            <div>Status: {status?.running ? "Running" : "Stopped"}</div>
            {status?.pid && <div>PID: {status.pid}</div>}
            <div>URL: {status?.url ?? "Unavailable"}</div>
            {status?.logs && (
              <div style={{ color: "#a1a1aa", marginTop: "8px" }}>
                Logs: {status.logs.plugin}
              </div>
            )}
            {error && <div style={{ color: "#f87171", marginTop: "8px" }}>{error}</div>}
          {status?.warning && <div style={{ color: "#facc15", marginTop: "8px" }}>{status.warning}</div>}
          {status?.error && <div style={{ color: "#f87171", marginTop: "8px" }}>{status.error}</div>}
        </div>
      </PanelSectionRow>
      <PanelSectionRow>
        <ButtonItem layout="below" onClick={refresh}>Refresh</ButtonItem>
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
