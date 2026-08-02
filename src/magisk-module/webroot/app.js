(() => {
  "use strict";

  const state = { capabilities: null, status: null, config: null, logText: "", jobs: [], polling: null };
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
    actionCards: $("#actionCards"),
    jobLaunchers: $("#jobLaunchers"),
    jobList: $("#jobList"),
    inventoryLaunchers: $("#inventoryLaunchers"),
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

  function showFatal(error) {
    ui.connectionBadge.textContent = "disconnected";
    ui.connectionBadge.className = "badge danger";
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
      throw new Error(message);
    }
    const contentType = response.headers.get("content-type") || "";
    return contentType.includes("application/json") ? response.json() : response.text();
  }

  function activateTab(button) {
    if (!button || button.hidden) return;
    $$(".tab").forEach(item => item.classList.remove("active"));
    $$(".tab-panel").forEach(item => item.classList.remove("active"));
    button.classList.add("active");
    $(`#${button.dataset.panel}`)?.classList.add("active");
  }

  function wireTabs() {
    $$(".tab").forEach(button => button.addEventListener("click", () => activateTab(button)));
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
      details.append(element("dt", { text: key }), element("dd", { text: value }));
    });
    ui.statusDetails.replaceChildren(details);

    const safety = status.safety || {};
    ui.safetyCards.replaceChildren(...Object.entries(safety).map(([key, value]) =>
      card(key.replaceAll("_", " "), value === true ? "PASS" : String(value), value === true ? "good" : "caution")
    ));
  }

  function field(definition, value) {
    if (definition.type === "boolean") {
      const input = element("input", { type: "checkbox", checked: Boolean(value), name: definition.key });
      return element("label", { className: "field" }, [
        element("span", { text: definition.label }),
        element("span", { className: "toggle" }, [input, element("small", { text: definition.description || "" })]),
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
      element("small", { text: definition.description || "" }),
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
    renderConfig();
    await refreshStatus();
    showNotice("Settings saved.");
  }

  function renderActions() {
    const actions = state.capabilities?.actions || [];
    ui.actionCards.replaceChildren(...actions.map(definition => {
      const dryRun = element("input", { type: "checkbox", checked: Boolean(definition.supports_dry_run) });
      const confirmation = element("input", {
        type: "text",
        placeholder: definition.requires_confirmation ? `Type ${definition.confirmation_text}` : "",
      });
      const run = element("button", { type: "button", className: risk(definition.risk), text: definition.label });
      run.addEventListener("click", async () => {
        run.disabled = true;
        try {
          const result = await api("/api/v1/action", {
            method: "POST",
            body: JSON.stringify({
              name: definition.name,
              dry_run: definition.supports_dry_run ? dryRun.checked : false,
              confirmation: definition.requires_confirmation ? confirmation.value : "",
            }),
          });
          showNotice(result.message || `${definition.label} completed.`);
          await refreshAll();
        } catch (error) {
          showFatal(error);
        } finally {
          run.disabled = false;
        }
      });
      const controls = element("div", { className: "action-controls" });
      if (definition.supports_dry_run) controls.append(element("label", { className: "toggle" }, [
        dryRun, element("span", { text: "Dry-run" }),
      ]));
      if (definition.requires_confirmation) controls.append(element("label", { className: "field" }, [
        element("span", { text: "Confirmation" }), confirmation,
      ]));
      controls.append(run);
      return element("article", { className: "action-card" }, [
        element("header", {}, [
          element("div", {}, [
            element("h3", { text: definition.label }),
            element("p", { className: "muted", text: definition.description || "" }),
          ]),
          element("span", { className: `badge ${risk(definition.risk)}`, text: definition.risk }),
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
          showFatal(error);
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
    const response = await api("/api/v1/jobs");
    state.jobs = response.data || [];
    await renderJobs();
    if (!state.jobs.some(job => ["queued", "running"].includes(job.status))) stopJobPolling();
  }

  function startJobPolling() {
    if (!state.polling) state.polling = window.setInterval(() => refreshJobs().catch(showFatal), 1800);
  }

  function stopJobPolling() {
    if (state.polling) window.clearInterval(state.polling);
    state.polling = null;
  }

  function renderInventoryLaunchers() {
    const inventories = state.capabilities?.inventories || [];
    ui.inventoryLaunchers.replaceChildren(...inventories.map(definition => {
      const button = element("button", { type: "button", text: definition.label });
      button.addEventListener("click", async () => {
        button.disabled = true;
        try {
          renderInventory(await api(`/api/v1/inventory?name=${encodeURIComponent(definition.name)}`));
        } catch (error) {
          showFatal(error);
        } finally {
          button.disabled = false;
        }
      });
      return button;
    }));
  }

  function renderInventory(response) {
    const columns = response.columns || [];
    const items = response.items || [];
    if (!columns.length || !items.length) {
      ui.inventoryOutput.replaceChildren(element("p", { className: "muted", text: "Inventory is empty." }));
      return;
    }
    const table = element("table");
    table.append(element("thead", {}, [element("tr", {}, columns.map(column => element("th", { text: column })))]));
    table.append(element("tbody", {}, items.map(item =>
      element("tr", {}, columns.map(column => element("td", { text: String(item[column] ?? "") })))
    )));
    ui.inventoryOutput.replaceChildren(table);
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
  }

  async function refreshAll() {
    await Promise.all([
      refreshStatus(),
      state.capabilities?.features?.config
        ? api("/api/v1/config").then(value => { state.config = value; })
        : Promise.resolve(),
      state.capabilities?.features?.logs ? loadLog() : Promise.resolve(),
      state.capabilities?.features?.jobs ? refreshJobs() : Promise.resolve(),
    ]);
    renderConfig();
  }

  async function initialize() {
    wireTabs();
    $("#refreshButton").addEventListener("click", () => refreshAll().then(() => showNotice("Refreshed.")).catch(showFatal));
    ui.configForm.addEventListener("submit", event => saveConfig(event).catch(showFatal));
    $("#reloadLogButton").addEventListener("click", () => loadLog().catch(showFatal));
    $("#copyLogButton").addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(ui.logOutput.textContent);
        showNotice("Log copied.");
      } catch {
        showNotice("Clipboard access was denied.", "caution");
      }
    });
    ui.logFilter.addEventListener("input", renderLog);

    const response = await api("/api/v1/capabilities");
    state.capabilities = response.capabilities;
    applyFeatureVisibility();
    renderActions();
    renderJobLaunchers();
    renderInventoryLaunchers();
    await refreshAll();
    if (state.jobs.some(job => ["queued", "running"].includes(job.status))) startJobPolling();
  }

  initialize().catch(showFatal);
})();
