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

  type View = "games" | "jobs";

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
  let activeView: View = "games";
  let gameQuery = "";
  let gameSwitcherOpen = false;

  $: cleanCount = games.filter((game) => game.state === "clean_candidate").length;
  $: reviewCount = games.length - cleanCount;
  $: selectedProfile = profiles.find((profile) => profile.is_default) ?? profiles[0] ?? null;
  $: filteredGames = games.filter((game) => {
    const query = gameQuery.trim().toLowerCase();
    if (!query) return true;
    return game.name.toLowerCase().includes(query) || game.app_id.includes(query);
  });

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
    activeView = "games";
    gameSwitcherOpen = false;
    resolvedImport = "";
    nexusFiles = [];
    downloadLinks = [];
    await loadProfiles(game);
  }

  function backToLibrary() {
    selectedGame = null;
    profiles = [];
    profileName = "";
    importURL = "";
    lastImportURL = "";
    resolvedImport = "";
    nexusFiles = [];
    downloadLinks = [];
    gameSwitcherOpen = false;
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

  function upsertJob(job: Job) {
    jobs = [job, ...jobs.filter((item) => item.id !== job.id)];
  }

  function stateLabel(state: string) {
    return state === "clean_candidate" ? "Clean" : "Review";
  }

  function gameImage(appID: string) {
    return `https://cdn.cloudflare.steamstatic.com/steam/apps/${appID}/header.jpg`;
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
    <div>
      <p class="eyebrow">Decky Mod Manager</p>
      <h1>{activeView === "games" ? selectedGame?.name ?? "Games" : "Jobs"}</h1>
    </div>
    <button type="button" class="icon-button" aria-label="Refresh" on:click={refresh}>R</button>
  </header>

  <section class="status-strip" aria-label="App status">
    <span>{status?.nexus.api_key_configured ? "Nexus ready" : "Nexus missing"}</span>
    <span>{cleanCount} clean</span>
    <span>{reviewCount} review</span>
  </section>

  <nav class="view-tabs" aria-label="Primary">
    <button type="button" class:active={activeView === "games"} on:click={() => (activeView = "games")}>Games</button>
    <button type="button" class:active={activeView === "jobs"} on:click={() => (activeView = "jobs")}>Jobs</button>
  </nav>

  {#if error}
    <section class="alert">{error}</section>
  {/if}

  {#if loading}
    <section class="empty-state">Loading...</section>
  {:else if activeView === "games"}
    {#if selectedGame}
      <section class="game-workspace">
        <div class="game-nav">
          <button type="button" class="secondary" on:click={backToLibrary}>Games</button>
          <button type="button" class="secondary" on:click={() => (gameSwitcherOpen = !gameSwitcherOpen)}>
            Change Game
          </button>
        </div>

        {#if gameSwitcherOpen}
          <aside class="game-browser compact">
            <div class="browser-toolbar">
              <input bind:value={gameQuery} aria-label="Search games" placeholder="Search games" />
            </div>
            <div class="game-list compact-list">
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
          </aside>
        {/if}

          <article class="game-hero">
            <img src={gameImage(selectedGame.app_id)} alt="" />
            <div>
              <h2>{selectedGame.name}</h2>
              <span class:review-badge={selectedGame.state !== "clean_candidate"}>{stateLabel(selectedGame.state)}</span>
            </div>
          </article>

          {#if selectedGame.markers?.length}
            <article class="workspace-panel">
              <h2>Review</h2>
              <div class="markers">
                {#each selectedGame.markers as marker}
                  <span>{marker}</span>
                {/each}
              </div>
            </article>
          {/if}

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

          <article class="workspace-panel">
            <div class="panel-heading">
              <h2>Mods</h2>
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

          <article class="workspace-panel path-panel">
            <h2>Install Path</h2>
            <p>{selectedGame.path}</p>
          </article>
      </section>
    {:else}
      <section class="library-view">
        <div class="browser-toolbar">
          <input bind:value={gameQuery} aria-label="Search games" placeholder="Search games" />
        </div>

        <div class="library-grid">
          {#each filteredGames as game}
            <button
              type="button"
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
      </section>
    {/if}
  {:else}
    <section class="workspace-panel">
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
    </section>
  {/if}
</main>
