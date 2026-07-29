<script lang="ts">
  import { onMount } from "svelte";

  type Status = {
    listen_addr: string;
    lan_only: boolean;
    data_dir: string;
    nexus: { api_key_configured: boolean };
  };

  type Dependency = {
    name: string;
    command: string;
    installed: boolean;
    path?: string;
    description: string;
    install_hint?: string;
  };

  type Game = {
    app_id: string;
    name: string;
    path: string;
    library_path: string;
    state: string;
    markers?: string[];
  };

  type Profile = {
    id: number;
    game_id: number;
    name: string;
    is_default: boolean;
  };

  type Job = {
    id: string;
    type: string;
    title: string;
    status: string;
    message: string;
    created_at: string;
    updated_at: string;
  };

  type DownloadLink = {
    name: string;
    short_name: string;
    URI: string;
  };

  let status: Status | null = null;
  let dependencies: Dependency[] = [];
  let games: Game[] = [];
  let jobs: Job[] = [];
  let selectedGame: Game | null = null;
  let profiles: Profile[] = [];
  let profileName = "";
  let importURL = "";
  let resolvedImport = "";
  let downloadLinks: DownloadLink[] = [];
  let nexusUser = "";
  let nexusAPIKey = "";
  let loading = true;
  let error = "";
  let savingSecurity = false;

  async function getJSON<T>(url: string): Promise<T> {
    const response = await fetch(url);
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  async function refresh() {
    error = "";
    try {
      const [nextStatus, nextDeps, nextGames, nextJobs] = await Promise.all([
        getJSON<Status>("/api/status"),
        getJSON<Dependency[]>("/api/dependencies"),
        getJSON<Game[]>("/api/games"),
        getJSON<Job[]>("/api/jobs")
      ]);
      status = nextStatus;
      dependencies = nextDeps;
      games = nextGames;
      jobs = nextJobs;
      selectedGame = nextGames.find((game) => game.state === "clean_candidate") ?? nextGames[0] ?? null;
      if (selectedGame) await loadProfiles(selectedGame);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  async function selectGame(game: Game) {
    selectedGame = game;
    await loadProfiles(game);
  }

  async function loadProfiles(game: Game) {
    profiles = await getJSON<Profile[]>(`/api/games/${game.app_id}/profiles`);
  }

  async function createProfile() {
    if (!selectedGame || !profileName.trim()) return;
    error = "";
    const response = await fetch(`/api/games/${selectedGame.app_id}/profiles`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name: profileName })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    profileName = "";
    await loadProfiles(selectedGame);
  }

  async function setDefaultProfile(profile: Profile) {
    if (!selectedGame || profile.is_default) return;
    error = "";
    const response = await fetch(`/api/profiles/${profile.id}/default`, { method: "PUT" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    await loadProfiles(selectedGame);
  }

  async function resolveImport() {
    if (!importURL.trim()) return;
    error = "";
    const response = await fetch("/api/imports/resolve", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url: importURL })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    upsertJob(result.job);
    resolvedImport = `${result.resolved.catalog}:${result.resolved.game_domain}/mods/${result.resolved.mod_id}${result.resolved.file_id ? `/files/${result.resolved.file_id}` : ""}`;
    downloadLinks = result.download_links ?? [];
    importURL = "";
  }

  async function saveNexusKey() {
    error = "";
    const response = await fetch("/api/settings/nexus", {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ api_key: nexusAPIKey })
    });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    status = await response.json();
    nexusAPIKey = "";
  }

  async function validateNexusKey() {
    error = "";
    nexusUser = "";
    const response = await fetch("/api/nexus/validate", { method: "POST" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    const result = await response.json();
    nexusUser = `${result.name}${result.is_premium ? " · Premium" : ""}`;
  }

  async function saveSecurity(lanOnly: boolean) {
    error = "";
    savingSecurity = true;
    const previous = status;
    if (status) status = { ...status, lan_only: lanOnly };
    try {
      const response = await fetch("/api/settings/security", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ lan_only: lanOnly })
      });
      if (!response.ok) throw new Error(await response.text());
      status = await response.json();
    } catch (err) {
      status = previous;
      error = err instanceof Error ? err.message : String(err);
    } finally {
      savingSecurity = false;
    }
  }

  function upsertJob(job: Job) {
    jobs = [job, ...jobs.filter((item) => item.id !== job.id)];
  }

  onMount(() => {
    refresh();
    const events = new EventSource("/api/jobs/events");
    events.addEventListener("job", (event) => {
      upsertJob(JSON.parse((event as MessageEvent).data));
    });
    return () => events.close();
  });
</script>

<main class="shell">
  <section class="topbar">
    <div>
      <p class="eyebrow">Steam Deck</p>
      <h1>Decky Mod Manager</h1>
    </div>
    <button type="button" on:click={refresh}>Refresh</button>
  </section>

  {#if error}
    <section class="alert">{error}</section>
  {/if}

  {#if loading}
    <section class="panel">Loading...</section>
  {:else}
    <section class="grid">
      <article class="panel">
        <h2>Server</h2>
        {#if status}
          <dl>
            <div><dt>Listen</dt><dd>{status.listen_addr}</dd></div>
            <div><dt>LAN only</dt><dd>{status.lan_only ? "Enabled" : "Disabled"}</dd></div>
            <div><dt>Nexus API key</dt><dd>{status.nexus.api_key_configured ? "Configured" : "Missing"}</dd></div>
            <div><dt>Data</dt><dd>{status.data_dir}</dd></div>
          </dl>
        {/if}
        {#if status}
          <label class="toggle">
            <input
              type="checkbox"
              checked={status.lan_only}
              disabled={savingSecurity}
              on:change={(event) => saveSecurity((event.currentTarget as HTMLInputElement).checked)}
            />
            <span>Restrict access to LAN addresses</span>
          </label>
        {/if}
        <p class="warning">No app authentication is enabled for MVP. Keep LAN-only enabled unless using a trusted tunnel such as Tailscale.</p>
      </article>

      <article class="panel">
        <h2>Import</h2>
        <form on:submit|preventDefault={resolveImport}>
          <label for="import-url">Nexus URL or nxm:// link</label>
          <textarea id="import-url" bind:value={importURL} rows="4" placeholder="Paste a Nexus mod URL or nxm:// link"></textarea>
          <button type="submit">Resolve</button>
        </form>
        {#if resolvedImport}
          <p class="hint">Resolved {resolvedImport}</p>
        {/if}
        {#if downloadLinks.length > 0}
          <div class="link-list">
            {#each downloadLinks as link}
              <a href={link.URI}>{link.name || link.short_name || "Download link"}</a>
            {/each}
          </div>
        {/if}
      </article>

      <article class="panel">
        <h2>Nexus</h2>
        <form on:submit|preventDefault={saveNexusKey}>
          <label for="nexus-key">API key</label>
          <textarea id="nexus-key" bind:value={nexusAPIKey} rows="3" placeholder="Paste Nexus API key"></textarea>
          <button type="submit">Save Key</button>
        </form>
        <button type="button" class="secondary" on:click={validateNexusKey}>Validate Key</button>
        {#if nexusUser}
          <p class="hint">Signed in as {nexusUser}</p>
        {/if}
        <p class="hint">The key is stored in the Deck's local config file and is not echoed back to this UI.</p>
      </article>
    </section>

    <section class="panel">
      <h2>Dependencies</h2>
      <div class="dependency-list">
        {#each dependencies as dep}
          <div class:ok={dep.installed} class:missing={!dep.installed} class="dependency">
            <strong>{dep.name}</strong>
            <span>{dep.installed ? dep.path : dep.install_hint}</span>
          </div>
        {/each}
      </div>
    </section>

    <section class="panel">
      <h2>Games</h2>
      <p class="summary">{games.filter((game) => game.state === "clean_candidate").length} clean candidates · {games.filter((game) => game.state !== "clean_candidate").length} need review</p>
      {#if selectedGame}
        <div class="selected">
          <div>
            <strong>{selectedGame.name}</strong>
            <span>{selectedGame.state}</span>
          </div>
          <div>
            Active profile:
            <strong>{profiles.find((profile) => profile.is_default)?.name ?? "None"}</strong>
          </div>
        </div>
        {#if selectedGame.markers?.length}
          <div class="markers">
            {#each selectedGame.markers as marker}
              <span>{marker}</span>
            {/each}
          </div>
        {/if}
        <form class="inline-form" on:submit|preventDefault={createProfile}>
          <label for="profile-name">New profile</label>
          <input id="profile-name" bind:value={profileName} placeholder="Profile name" />
          <button type="submit">Create</button>
        </form>
        {#if profiles.length > 0}
          <div class="profile-list">
            {#each profiles as profile}
              <button type="button" class:active-profile={profile.is_default} on:click={() => setDefaultProfile(profile)}>
                <span>{profile.name}</span>
                <strong>{profile.is_default ? "Default" : "Make default"}</strong>
              </button>
            {/each}
          </div>
        {/if}
      {/if}
      <div class="games">
        {#each games as game}
          <button type="button" class:selected-game={selectedGame?.app_id === game.app_id} class="game" on:click={() => selectGame(game)}>
            <div>
              <h3>{game.name}</h3>
              <p>{game.app_id} · {game.path}</p>
            </div>
            <span class:review={game.state !== "clean_candidate"}>{game.state}</span>
          </button>
        {/each}
      </div>
    </section>

    <section class="panel">
      <h2>Jobs</h2>
      {#if jobs.length === 0}
        <p>No jobs yet.</p>
      {:else}
        <div class="jobs">
          {#each jobs as job}
            <article class="job">
              <strong>{job.title}</strong>
              <span>{job.status}</span>
              {#if job.message}<p>{job.message}</p>{/if}
            </article>
          {/each}
        </div>
      {/if}
    </section>
  {/if}
</main>
