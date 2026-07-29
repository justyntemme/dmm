<script lang="ts">
  import { onMount } from "svelte";

  type Status = {
    game_count: number;
    nexus: { api_key_configured: boolean };
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

  type NexusFile = {
    file_id: number;
    name: string;
    version: string;
    file_name: string;
    size: number;
  };

  type Drawer = "games" | "settings" | null;
  type Surface = "requests" | "game" | "settings";
  type GameModule = "plugins" | "requests" | "profiles" | "review" | "paths";
  type SettingsPage = "overview" | "jobs" | "server" | "nexus";

  let status: Status | null = null;
  let games: Game[] = [];
  let jobs: Job[] = [];
  let selectedGame: Game | null = null;
  let profiles: Profile[] = [];
  let profileName = "";
  let importURL = "";
  let lastImportURL = "";
  let resolvedImport = "";
  let nexusFiles: NexusFile[] = [];
  let downloadLinks: DownloadLink[] = [];
  let loading = true;
  let error = "";
  let drawer: Drawer = null;
  let surface: Surface = "requests";
  let activeGameModule: GameModule = "plugins";
  let activeSettingsPage: SettingsPage = "overview";
  let gameQuery = "";

  $: cleanCount = games.filter((game) => game.state === "clean_candidate").length;
  $: reviewCount = games.length - cleanCount;
  $: selectedProfile = profiles.find((profile) => profile.is_default) ?? profiles[0] ?? null;
  $: installRequests = jobs.filter((job) => job.type === "pending-import");
  $: selectedGameRequests = selectedGame ? installRequests.filter((job) => requestMatchesGame(job, selectedGame)) : installRequests;
  $: filteredGames = games.filter((game) => {
    const query = gameQuery.trim().toLowerCase();
    if (!query) return true;
    return game.name.toLowerCase().includes(query) || game.app_id.includes(query);
  });
  $: title = surface === "settings" ? settingsTitle(activeSettingsPage) : surface === "requests" ? "Install Requests" : selectedGame?.name ?? "Select a Game";

  async function getJSON<T>(url: string): Promise<T> {
    const response = await fetch(url);
    if (!response.ok) throw new Error(await response.text());
    return response.json();
  }

  async function refresh() {
    error = "";
    try {
      const [nextStatus, nextGames, nextJobs] = await Promise.all([
        getJSON<Status>("/api/status"),
        getJSON<Game[]>("/api/games"),
        getJSON<Job[]>("/api/jobs")
      ]);
      status = nextStatus;
      games = nextGames;
      jobs = nextJobs;
      const previousSelection = selectedGame?.app_id;
      selectedGame = nextGames.find((game) => game.app_id === previousSelection) ?? null;
      if (selectedGame) await loadProfiles(selectedGame);
    } catch (err) {
      error = err instanceof Error ? err.message : String(err);
    } finally {
      loading = false;
    }
  }

  async function selectGame(game: Game) {
    selectedGame = game;
    surface = "game";
    activeGameModule = "plugins";
    drawer = null;
    resolvedImport = "";
    nexusFiles = [];
    downloadLinks = [];
    await loadProfiles(game);
  }

  function openSettings(page: SettingsPage) {
    activeSettingsPage = page;
    surface = "settings";
    drawer = null;
  }

  function openRequests() {
    surface = "requests";
    drawer = null;
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
    if (!selectedGame || !importURL.trim()) return;
    error = "";
    lastImportURL = importURL;
    nexusFiles = [];
    downloadLinks = [];
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
    nexusFiles = result.files ?? [];
    downloadLinks = result.download_links ?? [];
    importURL = "";
  }

  async function resolveFile(file: NexusFile) {
    if (!lastImportURL) return;
    const nextURL = new URL(lastImportURL);
    nextURL.searchParams.set("file_id", String(file.file_id));
    importURL = nextURL.toString();
    await resolveImport();
  }

  async function clearInstallRequests() {
    error = "";
    const response = await fetch("/api/imports/pending", { method: "DELETE" });
    if (!response.ok) {
      error = await response.text();
      return;
    }
    jobs = jobs.filter((job) => job.type !== "pending-import");
  }

  function upsertJob(job: Job) {
    if (job.type === "pending-import" && job.status === "canceled" && job.message === "Cleared") {
      jobs = jobs.filter((item) => item.id !== job.id);
      return;
    }
    jobs = [job, ...jobs.filter((item) => item.id !== job.id)];
  }

  function stateLabel(state: string) {
    return state === "clean_candidate" ? "Clean" : "Review";
  }

  function gameImage(appID: string) {
    return `https://cdn.cloudflare.steamstatic.com/steam/apps/${appID}/header.jpg`;
  }

  function settingsTitle(page: SettingsPage) {
    if (page === "jobs") return "Jobs";
    if (page === "server") return "Server";
    if (page === "nexus") return "Nexus";
    return "Settings";
  }

  function requestMatchesGame(job: Job, game: Game) {
    const haystack = `${job.title} ${job.message}`.toLowerCase().replace(/[^a-z0-9]/g, "");
    const gameName = game.name.toLowerCase().replace(/[^a-z0-9]/g, "");
    if (gameName && haystack.includes(gameName)) return true;
    const aliases: Record<string, string[]> = {
      "413150": ["stardewvalley", "stardew"],
      "489830": ["skyrimspecialedition", "skyrim"],
      "292030": ["witcher3", "thewitcher3"],
      "377160": ["fallout4"],
      "287700": ["metalgearsolidvtpp", "mgsv"]
    };
    return (aliases[game.app_id] ?? []).some((alias) => haystack.includes(alias));
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

<main class="app-shell">
  <header class="app-header">
    <button type="button" class="icon-button" aria-label="Open games" on:click={() => (drawer = "games")}>☰</button>
    <div class="title-block">
      {#if surface === "game" && selectedGame}
        <img src={gameImage(selectedGame.app_id)} alt="" />
      {/if}
      <div>
        <p class="eyebrow">Decky Mod Manager</p>
        <h1>{title}</h1>
      </div>
    </div>
    <button type="button" class="icon-button" aria-label="Open settings" on:click={() => (drawer = "settings")}>⚙</button>
  </header>

  {#if drawer}
    <button type="button" class="scrim" aria-label="Close menu" on:click={() => (drawer = null)}></button>
    <aside class="drawer">
      {#if drawer === "games"}
        <div class="drawer-heading">
          <h2>Games</h2>
          <button type="button" class="icon-button small" aria-label="Refresh games" on:click={refresh}>R</button>
        </div>
        <input bind:value={gameQuery} aria-label="Search games" placeholder="Search games" />
        <div class="drawer-list game-list">
          {#each filteredGames as game}
            <button
              type="button"
              class:selected={selectedGame?.app_id === game.app_id}
              class:needs-review={game.state !== "clean_candidate"}
              on:click={() => selectGame(game)}
            >
              <img src={gameImage(game.app_id)} alt="" loading="lazy" />
              <span>
                <strong>{game.name}</strong>
                <small>{game.app_id} · {stateLabel(game.state)}</small>
              </span>
            </button>
          {/each}
        </div>
      {:else}
        <div class="drawer-heading">
          <h2>Settings</h2>
          <button type="button" class="icon-button small" aria-label="Close settings" on:click={() => (drawer = null)}>×</button>
        </div>
        <nav class="settings-nav" aria-label="Settings">
          <button type="button" class:active={activeSettingsPage === "overview"} on:click={() => openSettings("overview")}>Overview</button>
          <button type="button" on:click={openRequests}>Install Requests</button>
          <button type="button" class:active={activeSettingsPage === "jobs"} on:click={() => openSettings("jobs")}>Jobs</button>
          <button type="button" class:active={activeSettingsPage === "server"} on:click={() => openSettings("server")}>Server</button>
          <button type="button" class:active={activeSettingsPage === "nexus"} on:click={() => openSettings("nexus")}>Nexus</button>
        </nav>
      {/if}
    </aside>
  {/if}

  <section class="status-strip" aria-label="App status">
    <span>{status?.nexus.api_key_configured ? "Nexus ready" : "Nexus missing"}</span>
    <span>{cleanCount} clean</span>
    <span>{reviewCount} review</span>
  </section>

  {#if error}
    <section class="alert">{error}</section>
  {/if}

  {#if loading}
    <section class="empty-state">Loading...</section>
  {:else if surface === "requests"}
    <section class="settings-screen">
      <article class="workspace-panel">
        <div class="panel-heading">
          <h2>Install Requests</h2>
          <span>{installRequests.length} pending</span>
        </div>
        {#if installRequests.length > 0}
          <button type="button" class="secondary-action" on:click={clearInstallRequests}>Clear Requests</button>
        {/if}
        {#if installRequests.length === 0}
          <div class="empty-state inline-empty">
            <h2>No Install Requests</h2>
            <p class="hint">Add a Nexus URL from the Decky plugin or paste one from a selected game's Plugins tab.</p>
          </div>
        {:else}
          <div class="request-list">
            {#each installRequests as request}
              <article class:failed-request={request.status === "failed"}>
                <div>
                  <strong>{request.title}</strong>
                  {#if request.message}<p>{request.message}</p>{/if}
                  <small>{new Date(request.updated_at).toLocaleString()}</small>
                </div>
                <span>{request.status}</span>
              </article>
            {/each}
          </div>
        {/if}
      </article>
    </section>
  {:else if surface === "settings"}
    <section class="settings-screen">
      {#if activeSettingsPage === "overview"}
        <article class="workspace-panel">
          <h2>Overview</h2>
          <dl class="settings-list">
            <div><dt>Games</dt><dd>{status?.game_count ?? games.length}</dd></div>
            <div><dt>Clean</dt><dd>{cleanCount}</dd></div>
            <div><dt>Review</dt><dd>{reviewCount}</dd></div>
            <div><dt>Nexus</dt><dd>{status?.nexus.api_key_configured ? "Configured" : "Missing API key"}</dd></div>
          </dl>
        </article>
      {:else if activeSettingsPage === "jobs"}
        <article class="workspace-panel">
          <h2>Jobs</h2>
          {#if jobs.length === 0}
            <p class="hint">No jobs yet.</p>
          {:else}
            <div class="jobs">
              {#each jobs as job}
                <article class="job">
                  <div>
                    <strong>{job.title}</strong>
                    {#if job.message}<p>{job.message}</p>{/if}
                  </div>
                  <span>{job.status}</span>
                </article>
              {/each}
            </div>
          {/if}
        </article>
      {:else if activeSettingsPage === "server"}
        <article class="workspace-panel">
          <h2>Server</h2>
          <dl class="settings-list">
            <div><dt>Status</dt><dd>Managed from the Decky plugin</dd></div>
            <div><dt>LAN</dt><dd>LAN-only access is configured from the Decky plugin</dd></div>
          </dl>
        </article>
      {:else}
        <article class="workspace-panel">
          <h2>Nexus</h2>
          <dl class="settings-list">
            <div><dt>API key</dt><dd>{status?.nexus.api_key_configured ? "Configured" : "Missing"}</dd></div>
          </dl>
        </article>
      {/if}
    </section>
  {:else if selectedGame}
    <section class="game-workspace">
      <article class="game-hero">
        <img src={gameImage(selectedGame.app_id)} alt="" />
        <div>
          <h2>{selectedGame.name}</h2>
          <span class:review-badge={selectedGame.state !== "clean_candidate"}>{stateLabel(selectedGame.state)}</span>
        </div>
      </article>

      <nav class="module-tabs" aria-label="Game modules">
        <button type="button" class:active={activeGameModule === "plugins"} on:click={() => (activeGameModule = "plugins")}>Plugins</button>
        <button type="button" class:active={activeGameModule === "requests"} on:click={() => (activeGameModule = "requests")}>Requests</button>
        <button type="button" class:active={activeGameModule === "profiles"} on:click={() => (activeGameModule = "profiles")}>Profiles</button>
        <button type="button" class:active={activeGameModule === "review"} on:click={() => (activeGameModule = "review")}>Review</button>
        <button type="button" class:active={activeGameModule === "paths"} on:click={() => (activeGameModule = "paths")}>Paths</button>
      </nav>

      {#if activeGameModule === "plugins"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Plugins</h2>
            <span>0 installed</span>
          </div>
          <form class="stacked-form" on:submit|preventDefault={resolveImport}>
            <textarea bind:value={importURL} rows="4" aria-label="Nexus URL" placeholder="Nexus mod URL or nxm:// Mod Manager Download link"></textarea>
            <button type="submit">Add Mod</button>
          </form>
          <p class="hint">Use a Nexus mod page to list files. Use the Mod Manager Download nxm:// link to resolve download mirrors.</p>
          {#if resolvedImport}
            <p class="hint">Resolved {resolvedImport}</p>
          {/if}
          {#if nexusFiles.length > 0}
            <div class="file-list">
              {#each nexusFiles as file}
                <button type="button" on:click={() => resolveFile(file)}>
                  <span>
                    <strong>{file.name || file.file_name}</strong>
                    <small>{file.file_name} · v{file.version || "unknown"}</small>
                  </span>
                  <em>Use</em>
                </button>
              {/each}
            </div>
          {/if}
          {#if downloadLinks.length > 0}
            <div class="link-list">
              {#each downloadLinks as link}
                <a href={link.URI}>{link.name || link.short_name || "Download link"}</a>
              {/each}
            </div>
          {/if}
        </article>
      {:else if activeGameModule === "requests"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Install Requests</h2>
            <span>{selectedGameRequests.length} shown</span>
          </div>
          {#if selectedGameRequests.length > 0}
            <button type="button" class="secondary-action" on:click={clearInstallRequests}>Clear Requests</button>
          {/if}
          {#if selectedGameRequests.length === 0}
            <p class="hint">No install requests matched this game.</p>
          {:else}
            <div class="request-list">
              {#each selectedGameRequests as request}
                <article class:failed-request={request.status === "failed"}>
                  <div>
                    <strong>{request.title}</strong>
                    {#if request.message}<p>{request.message}</p>{/if}
                    <small>{new Date(request.updated_at).toLocaleString()}</small>
                  </div>
                  <span>{request.status}</span>
                </article>
              {/each}
            </div>
          {/if}
        </article>
      {:else if activeGameModule === "profiles"}
        <article class="workspace-panel">
          <div class="panel-heading">
            <h2>Profiles</h2>
            <span>{selectedProfile?.name ?? "None"}</span>
          </div>
          <form class="inline-form" on:submit|preventDefault={createProfile}>
            <input bind:value={profileName} aria-label="Profile name" placeholder="Profile name" />
            <button type="submit">Create</button>
          </form>
          <div class="profile-list">
            {#each profiles as profile}
              <button type="button" class:active-profile={profile.is_default} on:click={() => setDefaultProfile(profile)}>
                <span>{profile.name}</span>
                <strong>{profile.is_default ? "Default" : "Set default"}</strong>
              </button>
            {/each}
          </div>
        </article>
      {:else if activeGameModule === "review"}
        <article class="workspace-panel">
          <h2>Review</h2>
          {#if selectedGame.markers?.length}
            <div class="markers">
              {#each selectedGame.markers as marker}
                <span>{marker}</span>
              {/each}
            </div>
          {:else}
            <p class="hint">No review markers for this game.</p>
          {/if}
        </article>
      {:else}
        <article class="workspace-panel path-panel">
          <h2>Paths</h2>
          <dl class="settings-list">
            <div><dt>Install</dt><dd>{selectedGame.path}</dd></div>
            <div><dt>Library</dt><dd>{selectedGame.library_path}</dd></div>
          </dl>
        </article>
      {/if}
    </section>
  {:else}
    <section class="empty-state select-game">
      <h2>Select a Game</h2>
      <p class="hint">Open the games menu to choose a Steam game before importing Nexus mods or managing profiles.</p>
      <button type="button" on:click={() => (drawer = "games")}>Open Games</button>
    </section>
  {/if}
</main>
