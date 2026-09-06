(() => {
  "use strict";

  const state = {
    capabilities: null,
    status: null,
    config: null,
    logText: "",
    jobs: [],
    polling: null,
    jobsRefreshPromise: null,
    actionJobSyncers: new Set(),
    inventoryCache: new Map(),
    inventoryCurrent: "",
    inventorySequence: 0,
  };
  const $ = selector => document.querySelector(selector);
  const $$ = selector => [...document.querySelectorAll(selector)];

  const ui = {
    moduleName: $("#moduleName"),
    moduleVersion: $("#moduleVersion"),
    connectionBadge: $("#connectionBadge"),
    notice: $("#notice"),
    statusCards: $("#statusCards"),
    statusDetails: $("#statusDetails"),
    configForm: $("#configForm"),
    dirtyBadge: $("#dirtyBadge"),
    saveConfigButton: $("#saveConfigButton"),
    actionStateSummary: $("#actionStateSummary"),
    actionCards: $("#actionCards"),
    jobLaunchers: $("#jobLaunchers"),
    jobList: $("#jobList"),
    inventoryLaunchers: $("#inventoryLaunchers"),
    inventoryRefreshButton: $("#inventoryRefreshButton"),
    inventoryMeta: $("#inventoryMeta"),
    inventoryOutput: $("#inventoryOutput"),
    logFilter: $("#logFilter"),
    logOutput: $("#logOutput"),
    safetyCards: $("#safetyCards"),
  };

  function element(tag, options = {}, children = []) {
    const node = document.createElement(tag);
    Object.entries(options).forEach(([key, value]) => {
      if (key === "className") node.className = value;
      else if (key === "text") node.textContent = value;
      else if (key === "attributes") Object.entries(value).forEach(([name, item]) => node.setAttribute(name, item));
      else node[key] = value;
    });
    children.forEach(child => node.append(child));
    return node;
  }

  function showNotice(message, level = "good") {
    ui.notice.textContent = message;
    ui.notice.className = `notice ${level}`;
    window.clearTimeout(showNotice.timer);
    showNotice.timer = window.setTimeout(() => ui.notice.classList.add("hidden"), 4500);
  }

  function showError(error, connectionFatal = false) {
    if (connectionFatal || error?.status === 401) {
      ui.connectionBadge.textContent = "disconnected";
      ui.connectionBadge.className = "badge danger";
    }
    showNotice(error instanceof Error ? error.message : String(error), "danger");
  }

  async function api(path, options = {}) {
    const headers = new Headers(options.headers || {});
    if (options.body) {
      headers.set("Content-Type", "application/json");
      headers.set("X-WebUI-Request", "1");
    }
    const response = await fetch(path, {
      ...options,
      headers,
      credentials: "same-origin",
      cache: "no-store",
    });
    if (!response.ok) {
      let message = `${response.status} ${response.statusText}`;
      try {
        const body = await response.json();
        message = body.error || message;
      } catch {
        // Keep the HTTP status when the body is not JSON.
      }
      const error = new Error(message);
      error.status = response.status;
      throw error;
    }
    const contentType = response.headers.get("content-type") || "";
    return contentType.includes("application/json") ? response.json() : response.text();
  }

  function visibleTabs() {
    return $$(".tab").filter(item => !item.hidden);
  }

  function activateTab(button, focus = false) {
    if (!button || button.hidden) return;
    $$(".tab").forEach(item => {
      item.classList.remove("active");
      item.setAttribute("aria-selected", "false");
      item.tabIndex = -1;
    });
    $$(".tab-panel").forEach(item => item.classList.remove("active"));
    button.classList.add("active");
    button.setAttribute("aria-selected", "true");
    button.tabIndex = 0;
    const panel = $(`#${button.dataset.panel}`);
    panel?.classList.add("active");
    if (focus) button.focus({ preventScroll: true });
    window.requestAnimationFrame(() => {
      button.scrollIntoView({ behavior: "auto", block: "nearest", inline: "nearest" });
    });

    if (button.dataset.panel === "inventoryPanel" && !state.inventoryCurrent) {
      const first = state.capabilities?.inventories?.[0];
      if (first) loadInventory(first.name).catch(error => showError(error));
    }
  }

  function wireTabs() {
    const tabs = $$(".tab");
    tabs.forEach(button => {
      button.setAttribute("role", "tab");
      button.setAttribute("aria-controls", button.dataset.panel);
      button.setAttribute("aria-selected", button.classList.contains("active") ? "true" : "false");
      button.tabIndex = button.classList.contains("active") ? 0 : -1;
      $(`#${button.dataset.panel}`)?.setAttribute("role", "tabpanel");
      button.addEventListener("click", () => activateTab(button));
      button.addEventListener("keydown", event => {
        if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
        const available = visibleTabs();
        const current = available.indexOf(button);
        if (current < 0) return;
        event.preventDefault();
        const target = event.key === "Home"
          ? available[0]
          : event.key === "End"
            ? available[available.length - 1]
            : available[(current + (event.key === "ArrowRight" ? 1 : -1) + available.length) % available.length];
        activateTab(target, true);
      });
    });
    $(".tabs")?.setAttribute("role", "tablist");
  }

  function applyFeatureVisibility() {
    const features = state.capabilities?.features || {};
    $$("[data-feature]").forEach(node => {
      const enabled = features[node.dataset.feature] === true;
      node.hidden = !enabled;
      if (!enabled) node.classList.remove("active");
    });
    if (!$(".tab.active:not([hidden])")) activateTab($(".tab:not([hidden])"));
  }

  function level(value) {
    return ["good", "caution", "danger", "muted"].includes(value) ? value : "";
  }

  function risk(value) {
    return value === "danger" ? "danger" : value === "caution" ? "caution" : "good";
  }

  function card(label, value, cardLevel = "") {
    return element("div", { className: "card" }, [
      element("div", { className: "label", text: label }),
      element("div", { className: `value ${cardLevel}`, text: value ?? "—" }),
    ]);
  }

  function flatten(value, prefix = "") {
    if (!value || typeof value !== "object" || Array.isArray(value)) return [];
    return Object.entries(value).flatMap(([key, item]) => {
      const name = prefix ? `${prefix}.${key}` : key;
      return item && typeof item === "object" && !Array.isArray(item)
        ? flatten(item, name)
        : [[name, Array.isArray(item) ? JSON.stringify(item) : String(item)]];
    });
  }

  function humanizeKey(value) {
    return String(value)
      .replaceAll(".", " · ")
      .replaceAll("_", " ")
      .replace(/\b\w/g, match => match.toUpperCase());
  }

  function renderStatus() {
    const status = state.status || {};
    const module = status.module || state.capabilities?.module || {};
    ui.moduleName.textContent = module.name || module.id || "Root Module WebUI";
    ui.moduleVersion.textContent = `${module.version || "unknown"} · standalone browser session`;
    ui.connectionBadge.textContent = "local · connected";
    ui.connectionBadge.className = "badge good";

    const config = status.config || {};
    const declared = Array.isArray(status.summary)
      ? status.summary.filter(item => item && typeof item.label === "string")
      : [];
    const fallback = [
      { label: "Module", value: module.id },
      { label: "Version", value: module.version },
      { label: "Enabled", value: config.enabled },
      { label: "Mode", value: config.mode },
      { label: "Log level", value: config.log_level },
      { label: "Interval", value: config.interval_seconds ? `${config.interval_seconds}s` : "—" },
    ];
    const summary = declared.length ? declared : fallback;
    ui.statusCards.replaceChildren(...summary.map(item => card(item.label, String(item.value ?? "—"), level(item.level))));

    const details = element("dl", { className: "details-grid" });
    flatten(status.runtime || {}).forEach(([key, value]) => {
      details.append(element("dt", { text: humanizeKey(key) }), element("dd", { text: value }));
    });
    ui.statusDetails.replaceChildren(details);

    const safety = status.safety || {};
    ui.safetyCards.replaceChildren(...Object.entries(safety).map(([key, value]) =>
      card(humanizeKey(key), value === true ? "PASS" : String(value), value === true ? "good" : "caution")
    ));
  }

  function configuredState(definition) {
    if (!definition.secret) return "";
    const runtime = state.status?.runtime;
    const key = `${definition.key}_configured`;
    if (!runtime || !Object.prototype.hasOwnProperty.call(runtime, key)) return "";
    const raw = runtime[key];
    const normalized = typeof raw === "string" ? raw.toLowerCase() : raw;
    const configured = normalized === true || normalized === 1 || normalized === "1" || normalized === "yes" || normalized === "true" || normalized === "configured";
    return configured ? "Configured · leave blank to preserve." : "Not configured.";
  }

  function fieldDescription(definition) {
    return [configuredState(definition), definition.description || ""].filter(Boolean).join(" ");
  }

  function field(definition, value) {
    if (definition.type === "boolean") {
      const input = element("input", { type: "checkbox", checked: Boolean(value), name: definition.key });
      return element("label", { className: "field" }, [
        element("span", { text: definition.label }),
        element("span", { className: "toggle" }, [input, element("small", { text: fieldDescription(definition) })]),
      ]);
    }
    let input;
    if (definition.type === "enum") {
      input = element("select", { name: definition.key });
      (definition.options || []).forEach(option => input.append(
        element("option", { value: option.value, text: option.label, selected: option.value === value })
      ));
    } else {
      input = element("input", {
        name: definition.key,
        type: definition.secret ? "password" : definition.type === "integer" ? "number" : "text",
        value: value ?? "",
      });
      if (definition.min !== undefined) input.min = definition.min;
      if (definition.max !== undefined) input.max = definition.max;
      if (definition.max_length) input.maxLength = definition.max_length;
      if (definition.pattern) input.pattern = definition.pattern;
    }
    return element("label", { className: "field" }, [
      element("span", { text: definition.label }),
      input,
      element("small", { text: fieldDescription(definition) }),
    ]);
  }

  function renderConfig() {
    if (!state.capabilities?.features?.config) return;
    const fields = state.capabilities.config_fields || [];
    ui.configForm.replaceChildren(...fields.map(definition => field(definition, state.config?.[definition.key])));
    ui.configForm.querySelectorAll("input,select").forEach(input =>
      input.addEventListener("input", () => ui.dirtyBadge.classList.remove("hidden"))
    );
    ui.saveConfigButton.disabled = false;
  }

  function readConfig() {
    const value = {};
    (state.capabilities?.config_fields || []).forEach(definition => {
      const input = ui.configForm.elements.namedItem(definition.key);
      if (!input) return;
      value[definition.key] = definition.type === "boolean"
        ? input.checked
        : definition.type === "integer" ? Number(input.value) : input.value;
    });
    return value;
  }

  async function saveConfig(event) {
    event.preventDefault();
    const response = await api("/api/v1/config", { method: "POST", body: JSON.stringify(readConfig()) });
    state.config = response.config || await api("/api/v1/config");
    ui.dirtyBadge.classList.add("hidden");
    state.inventoryCache.clear();
    await refreshStatus();
    renderConfig();
    showNotice("Settings saved.");
  }

  function actionState() {
    const reported = state.status?.action_state || {};
    return {
      active: new Set(Array.isArray(reported.active) ? reported.active.map(String) : []),
      blocked: reported.blocked && typeof reported.blocked === "object" && !Array.isArray(reported.blocked)
        ? reported.blocked
        : {},
    };
  }

  function renderActionSummary(actions, current) {
    if (!ui.actionStateSummary) return;
    const active = actions.filter(definition => current.active.has(definition.name));
    if (!active.length) {
      ui.actionStateSummary.replaceChildren(
        element("span", { className: "state-summary-label", text: "Current state" }),
        element("span", { className: "muted", text: state.status ? "No stateful action reported as active." : "Loading…" }),
      );
      return;
    }
    ui.actionStateSummary.replaceChildren(
      element("span", { className: "state-summary-label", text: "Currently active" }),
      element("div", { className: "state-chips" }, active.map(definition =>
        element("span", { className: "state-chip", text: definition.label })
      )),
    );
  }

  function renderActions() {
    const actions = state.capabilities?.actions || [];
    const current = actionState();
    renderActionSummary(actions, current);
    state.actionJobSyncers.clear();

    ui.actionCards.replaceChildren(...actions.map(definition => {
      const active = current.active.has(definition.name);
      const blockedReason = current.blocked[definition.name] ? String(current.blocked[definition.name]) : "";
      const applyJob = typeof definition.apply_job === "string" ? definition.apply_job : "";
      const dryRun = element("input", {
        type: "checkbox",
        checked: Boolean(definition.supports_dry_run),
        attributes: { "aria-label": `${definition.label}: preview only` },
      });
      const confirmation = element("input", {
        type: "text",
        placeholder: definition.requires_confirmation ? `Type ${definition.confirmation_text}` : "",
        autocomplete: "off",
      });
      const run = element("button", { type: "button", className: risk(definition.risk) });

      function syncRunState() {
        const confirmationMissing = definition.requires_confirmation && confirmation.value !== definition.confirmation_text;
        const jobRunning = Boolean(applyJob) && state.jobs.some(job =>
          job.name === applyJob && ["queued", "running"].includes(job.status)
        );
        run.disabled = Boolean(blockedReason) || confirmationMissing || jobRunning;
        const stateful = current.active.has(definition.name);
        const verb = jobRunning
          ? "Running…"
          : definition.supports_dry_run && dryRun.checked
            ? stateful ? "Preview current setting" : "Preview change"
            : stateful ? "Reapply current setting" : "Apply change";
        run.textContent = verb;
        run.setAttribute("aria-label", `${verb}: ${definition.label}`);
        if (jobRunning) run.title = `${definition.label} is running as a background job.`;
        else if (blockedReason) run.title = blockedReason;
        else run.removeAttribute("title");
      }

      confirmation.addEventListener("input", syncRunState);
      dryRun.addEventListener("change", syncRunState);
      if (applyJob) state.actionJobSyncers.add(syncRunState);
      syncRunState();

      run.addEventListener("click", async () => {
        run.disabled = true;
        try {
          const previewOnly = definition.supports_dry_run && dryRun.checked;
          if (!previewOnly && applyJob) {
            await api("/api/v1/jobs", {
              method: "POST",
              body: JSON.stringify({ name: applyJob }),
            });
            state.inventoryCache.clear();
            await refreshJobs();
            startJobPolling();
            showNotice(`${definition.label} started in Jobs.`);
            return;
          }
          const result = await api("/api/v1/action", {
            method: "POST",
            body: JSON.stringify({
              name: definition.name,
              dry_run: definition.supports_dry_run ? dryRun.checked : false,
              confirmation: definition.requires_confirmation ? confirmation.value : "",
            }),
          });
          if (!previewOnly) state.inventoryCache.clear();
          await refreshStatus();
          showNotice(result.message || `${definition.label} completed.`);
        } catch (error) {
          showError(error);
        } finally {
          syncRunState();
        }
      });

      const controls = element("div", { className: "action-controls" });
      if (definition.supports_dry_run) controls.append(element("label", { className: "toggle preview-toggle" }, [
        dryRun,
        element("span", {}, [
          element("strong", { text: "Preview only" }),
          element("small", { text: "Validate the request without changing runtime or persistent state." }),
        ]),
      ]));
      if (definition.requires_confirmation) controls.append(element("label", { className: "field" }, [
        element("span", { text: "Confirmation" }), confirmation,
      ]));
      if (blockedReason) controls.append(element("p", { className: "blocked-reason", text: blockedReason }));
      controls.append(run);

      const badges = element("div", { className: "action-badges" });
      if (active) badges.append(element("span", { className: "badge good active-badge", text: "ACTIVE" }));
      badges.append(element("span", { className: `badge ${risk(definition.risk)}`, text: definition.risk }));

      return element("article", {
        className: `action-card${active ? " active-state" : ""}${blockedReason ? " unavailable-state" : ""}`,
        attributes: active ? { "aria-current": "true" } : {},
      }, [
        element("header", {}, [
          element("div", {}, [
            element("h3", { text: definition.label }),
            element("p", { className: "muted", text: definition.description || "" }),
          ]),
          badges,
        ]),
        controls,
      ]);
    }));
  }

  function renderJobLaunchers() {
    const jobs = state.capabilities?.jobs || [];
    ui.jobLaunchers.replaceChildren(...jobs.map(definition => {
      const button = element("button", { type: "button", className: risk(definition.risk), text: definition.label });
      button.addEventListener("click", async () => {
        button.disabled = true;
        try {
          await api("/api/v1/jobs", { method: "POST", body: JSON.stringify({ name: definition.name }) });
          showNotice(`${definition.label} started.`);
          await refreshJobs();
          startJobPolling();
        } catch (error) {
          showError(error);
        } finally {
          button.disabled = false;
        }
      });
      const launcher = card(definition.risk, definition.label, risk(definition.risk));
      launcher.append(
        element("p", { className: "muted", text: definition.description || "" }),
        element("div", { className: "actions-row" }, [button])
      );
      return launcher;
    }));
  }

  async function outputFor(job, stream) {
    const response = await api(`/api/v1/jobs/${job.id}/output?stream=${stream}&offset=0&limit=65536`);
    return response.data?.text || "";
  }

  async function renderJobs() {
    const cards = state.jobs.map(job => {
      const output = element("pre", { className: "job-output", text: "Output not loaded." });
      const load = element("button", { type: "button", text: "Load output" });
      load.addEventListener("click", async () => {
        load.disabled = true;
        try {
          const stdout = await outputFor(job, "stdout");
          const stderr = await outputFor(job, "stderr");
          output.textContent = `${stdout}${stderr ? `\n--- stderr ---\n${stderr}` : ""}` || "(no output)";
        } catch (error) {
          output.textContent = error.message;
        } finally {
          load.disabled = false;
        }
      });
      return element("article", { className: "job-card" }, [
        element("header", {}, [
          element("div", {}, [element("h3", { text: job.name }), element("code", { text: job.id })]),
          element("span", {
            className: `badge ${job.status === "success" ? "good" : job.status === "failed" ? "danger" : "caution"}`,
            text: job.status,
          }),
        ]),
        element("div", { className: "job-meta" }, [
          element("span", { text: `stdout ${job.stdout_bytes} B` }),
          element("span", { text: `stderr ${job.stderr_bytes} B` }),
          element("span", { text: job.duration_seconds ? `${job.duration_seconds.toFixed(1)}s` : "pending" }),
          element("span", { text: job.truncated ? "output truncated" : "bounded output" }),
        ]),
        element("div", { className: "actions-row" }, [load]),
        output,
      ]);
    });
    ui.jobList.replaceChildren(...(cards.length ? cards : [
      element("p", { className: "muted", text: "No jobs in this WebUI session." }),
    ]));
  }

  async function refreshJobs() {
    if (state.jobsRefreshPromise) return state.jobsRefreshPromise;
    state.jobsRefreshPromise = (async () => {
      const response = await api("/api/v1/jobs");
      state.jobs = response.data || [];
      await renderJobs();
      state.actionJobSyncers.forEach(sync => sync());
      if (!state.jobs.some(job => ["queued", "running"].includes(job.status))) stopJobPolling();
    })();
    try {
      await state.jobsRefreshPromise;
    } finally {
      state.jobsRefreshPromise = null;
    }
  }

  function startJobPolling() {
    if (document.hidden || state.polling) return;
    state.polling = window.setInterval(() => refreshJobs().catch(error => showError(error)), 1800);
  }

  function stopJobPolling() {
    if (state.polling) window.clearInterval(state.polling);
    state.polling = null;
  }

  function inventoryDefinition(name) {
    return (state.capabilities?.inventories || []).find(definition => definition.name === name);
  }

  function syncInventoryLaunchers() {
    ui.inventoryLaunchers.querySelectorAll("button[data-inventory-name]").forEach(button => {
      const selected = button.dataset.inventoryName === state.inventoryCurrent;
      button.classList.toggle("active", selected);
      button.setAttribute("aria-pressed", selected ? "true" : "false");
    });
    if (ui.inventoryRefreshButton) ui.inventoryRefreshButton.disabled = !state.inventoryCurrent;
  }

  function renderInventoryLaunchers() {
    const inventories = state.capabilities?.inventories || [];
    ui.inventoryLaunchers.replaceChildren(...inventories.map(definition => {
      const button = element("button", {
        type: "button",
        text: definition.label,
        className: "inventory-launcher",
        attributes: {
          "data-inventory-name": definition.name,
          "aria-pressed": "false",
          title: definition.description || definition.label,
        },
      });
      button.addEventListener("click", () => loadInventory(definition.name).catch(error => showError(error)));
      return button;
    }));
    syncInventoryLaunchers();
  }

  function renderInventory(response, definition = null) {
    const columns = response.columns || [];
    const items = response.items || [];
    if (!columns.length || !items.length) {
      ui.inventoryOutput.replaceChildren(element("p", { className: "muted", text: "Inventory is empty." }));
      return;
    }
    const table = element("table", {
      className: "inventory-table",
      attributes: { "aria-label": definition?.label || response.name || "Inventory" },
    });
    table.append(element("thead", {}, [element("tr", {}, columns.map(column => element("th", { text: humanizeKey(column) })))]));
    table.append(element("tbody", {}, items.map(item =>
      element("tr", {}, columns.map(column => element("td", {
        text: String(item[column] ?? ""),
        attributes: { "data-label": humanizeKey(column) },
      })))
    )));
    ui.inventoryOutput.replaceChildren(table);
  }

  async function loadInventory(name, { force = false } = {}) {
    const definition = inventoryDefinition(name);
    if (!definition) throw new Error(`Unknown inventory: ${name}`);
    state.inventoryCurrent = name;
    const sequence = ++state.inventorySequence;
    syncInventoryLaunchers();

    const cached = state.inventoryCache.get(name);
    if (cached && !force) {
      renderInventory(cached.response, definition);
      if (ui.inventoryMeta) ui.inventoryMeta.textContent = "Cached for this browser session · Refresh view for a live read.";
      return;
    }

    if (ui.inventoryMeta) ui.inventoryMeta.textContent = force ? "Refreshing live inventory…" : "Loading inventory…";
    if (!cached) ui.inventoryOutput.replaceChildren(element("p", { className: "muted", text: "Loading…" }));

    const button = ui.inventoryLaunchers.querySelector(`button[data-inventory-name="${CSS.escape(name)}"]`);
    button?.classList.add("loading");
    button?.setAttribute("aria-busy", "true");

    try {
      const response = await api(`/api/v1/inventory?name=${encodeURIComponent(name)}`);
      state.inventoryCache.set(name, { response, loadedAt: Date.now() });
      if (sequence === state.inventorySequence && state.inventoryCurrent === name) {
        renderInventory(response, definition);
        if (ui.inventoryMeta) ui.inventoryMeta.textContent = "Live inventory loaded · switching views is now instant.";
      }
    } finally {
      button?.classList.remove("loading");
      button?.removeAttribute("aria-busy");
      syncInventoryLaunchers();
    }
  }

  function renderLog() {
    const query = ui.logFilter.value.toLowerCase().trim();
    ui.logOutput.textContent = query
      ? state.logText.split("\n").filter(line => line.toLowerCase().includes(query)).join("\n") || "No matching log lines."
      : state.logText || "Log is empty.";
  }

  async function loadLog() {
    state.logText = await api("/api/v1/log?lines=300");
    renderLog();
  }

  async function refreshStatus() {
    state.status = await api("/api/v1/status");
    renderStatus();
    renderActions();
  }

  async function refreshAll() {
    state.inventoryCache.clear();
    await Promise.all([
      refreshStatus(),
      state.capabilities?.features?.config
        ? api("/api/v1/config").then(value => { state.config = value; })
        : Promise.resolve(),
      state.capabilities?.features?.logs ? loadLog() : Promise.resolve(),
      state.capabilities?.features?.jobs ? refreshJobs() : Promise.resolve(),
      state.inventoryCurrent ? loadInventory(state.inventoryCurrent, { force: true }) : Promise.resolve(),
    ]);
    renderConfig();
  }

  async function initialize() {
    wireTabs();
    $("#refreshButton").addEventListener("click", () => refreshAll().then(() => showNotice("Refreshed.")).catch(error => showError(error, true)));
    ui.configForm.addEventListener("submit", event => saveConfig(event).catch(error => showError(error)));
    $("#reloadLogButton").addEventListener("click", () => loadLog().catch(error => showError(error)));
    $("#copyLogButton").addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(ui.logOutput.textContent);
        showNotice("Log copied.");
      } catch {
        showNotice("Clipboard access was denied.", "caution");
      }
    });
    ui.logFilter.addEventListener("input", renderLog);
    ui.inventoryRefreshButton?.addEventListener("click", () => {
      if (!state.inventoryCurrent) return;
      loadInventory(state.inventoryCurrent, { force: true })
        .then(() => showNotice("Inventory refreshed."))
        .catch(error => showError(error));
    });
    document.addEventListener("visibilitychange", () => {
      if (document.hidden) {
        stopJobPolling();
      } else if (state.jobs.some(job => ["queued", "running"].includes(job.status))) {
        refreshJobs().catch(error => showError(error));
        startJobPolling();
      }
    });

    const response = await api("/api/v1/capabilities");
    state.capabilities = response.capabilities;
    applyFeatureVisibility();
    renderActions();
    renderJobLaunchers();
    renderInventoryLaunchers();
    await refreshAll();
    if (state.jobs.some(job => ["queued", "running"].includes(job.status))) startJobPolling();
  }

  initialize().catch(error => showError(error, true));
})();
