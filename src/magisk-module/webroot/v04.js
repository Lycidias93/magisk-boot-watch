(() => {
  "use strict";

  const $ = selector => document.querySelector(selector);
  const state = { extensions: null, activeJobs: new Map(), referenceCache: new Map() };

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

  async function api(path, options = {}) {
    const headers = new Headers(options.headers || {});
    if (options.body !== undefined) {
      headers.set("Content-Type", "application/json");
      headers.set("X-WebUI-Request", "1");
    }
    const response = await fetch(path, { ...options, headers, credentials: "same-origin", cache: "no-store" });
    if (!response.ok) {
      let message = `${response.status} ${response.statusText}`;
      try {
        const body = await response.json();
        message = body.error || message;
      } catch {
        // Keep HTTP status.
      }
      const error = new Error(message);
      error.status = response.status;
      throw error;
    }
    return response.json();
  }

  function activateTab(button) {
    document.querySelectorAll(".tab").forEach(item => item.classList.remove("active"));
    document.querySelectorAll(".tab-panel").forEach(item => item.classList.remove("active"));
    button.classList.add("active");
    $(`#${button.dataset.panel}`)?.classList.add("active");
    button.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
  }

  function addTab() {
    const tabs = $(".tabs");
    const shell = $(".shell");
    const safety = $("#safetyPanel");
    if (!tabs || !shell || $("#v04WorkflowsPanel")) return $("#v04WorkflowsPanel");
    const button = element("button", { className: "tab", text: "Workflows", type: "button" });
    button.dataset.panel = "v04WorkflowsPanel";
    button.addEventListener("click", () => activateTab(button));
    tabs.append(button);
    const panel = element("section", { className: "panel tab-panel", attributes: { id: "v04WorkflowsPanel" } }, [
      element("div", { className: "panel-heading" }, [
        element("div", {}, [
          element("h2", { text: "Typed workflows" }),
          element("p", { text: "Bounded background jobs and inventory-bound operations. No arbitrary command or path input." }),
        ]),
      ]),
    ]);
    shell.insertBefore(panel, safety || null);
    return panel;
  }

  function riskClass(value) {
    return value === "danger" ? "danger" : value === "caution" ? "caution" : "good";
  }

  async function referenceValues(name) {
    if (state.referenceCache.has(name)) return state.referenceCache.get(name);
    const response = await api(`/api/v1/v04/reference?name=${encodeURIComponent(name)}`);
    const values = Array.isArray(response.items) ? response.items : [];
    state.referenceCache.set(name, values);
    return values;
  }

  async function parameterInput(definition) {
    let input;
    if (definition.type === "boolean") {
      input = element("input", { type: "checkbox" });
    } else if (definition.type === "enum") {
      input = element("select");
      (definition.options || []).forEach(option => input.append(element("option", { value: option.value, text: option.label })));
    } else if (definition.type === "reference") {
      input = element("select");
      const values = await referenceValues(definition.reference);
      values.forEach(value => input.append(element("option", { value, text: value })));
    } else {
      input = element("input", { type: definition.type === "integer" ? "number" : "text", value: "" });
      if (definition.min !== undefined) input.min = definition.min;
      if (definition.max !== undefined) input.max = definition.max;
      if (definition.max_length) input.maxLength = definition.max_length;
      if (definition.pattern) input.pattern = definition.pattern;
    }
    input.dataset.parameterKey = definition.key;
    input.dataset.parameterType = definition.type;
    if (definition.required) input.required = true;
    return element("label", { className: "field" }, [
      element("span", { text: definition.label || definition.key }),
      input,
      element("small", { text: definition.description || "" }),
    ]);
  }

  function readParameters(root, definition) {
    const parameters = {};
    for (const parameter of definition.parameters || []) {
      const input = root.querySelector(`[data-parameter-key="${CSS.escape(parameter.key)}"]`);
      if (!input) continue;
      if (parameter.type === "boolean") parameters[parameter.key] = input.checked;
      else if (parameter.type === "integer") parameters[parameter.key] = Number(input.value);
      else parameters[parameter.key] = input.value;
    }
    return parameters;
  }

  function jobStatusCard(job, definition, reused, digest) {
    const badge = element("span", { className: `badge ${job.status === "success" ? "good" : job.status === "failed" ? "danger" : "caution"}`, text: job.status });
    const meta = element("div", { className: "job-meta" }, [
      element("span", { text: reused ? "dedupe: reused active job" : "dedupe: new job" }),
      element("span", { text: `request ${String(digest || "").slice(0, 12)}…` }),
      element("span", { text: Array.isArray(definition.phases) && definition.phases.length ? `phases: ${definition.phases.join(" → ")}` : "phase: adapter-owned" }),
    ]);
    const card = element("article", { className: "job-card" }, [
      element("header", {}, [
        element("div", {}, [element("h3", { text: definition.label || definition.name }), element("code", { text: job.id })]),
        badge,
      ]),
      meta,
    ]);
    card.dataset.jobId = job.id;
    return { card, badge };
  }

  async function waitForJob(jobID, badge) {
    const schedule = [1000, 2000, 5000, 10000];
    let index = 0;
    while (state.activeJobs.has(jobID)) {
      if (document.hidden) {
        await new Promise(resolve => {
          const resume = () => {
            if (!document.hidden) {
              document.removeEventListener("visibilitychange", resume);
              resolve();
            }
          };
          document.addEventListener("visibilitychange", resume);
        });
      }
      await new Promise(resolve => setTimeout(resolve, schedule[Math.min(index, schedule.length - 1)]));
      index += 1;
      let response;
      try {
        response = await api(`/api/v1/jobs/${encodeURIComponent(jobID)}`);
      } catch {
        continue;
      }
      const job = response.data || {};
      badge.textContent = job.status || "unknown";
      badge.className = `badge ${job.status === "success" ? "good" : job.status === "failed" ? "danger" : "caution"}`;
      if (job.status === "success" || job.status === "failed") state.activeJobs.delete(jobID);
    }
  }

  async function renderJobDefinition(root, definition) {
    const form = element("div", { className: "form-grid" });
    for (const parameter of definition.parameters || []) form.append(await parameterInput(parameter));
    const output = element("div", { className: "stack" });
    const run = element("button", { type: "button", className: riskClass(definition.risk), text: definition.label || definition.name });
    run.addEventListener("click", async () => {
      run.disabled = true;
      try {
        const parameters = readParameters(form, definition);
        const response = await api("/api/v1/v04/jobs", { method: "POST", body: JSON.stringify({ name: definition.name, parameters }) });
        const job = response.job || {};
        const rendered = jobStatusCard(job, definition, response.reused === true, response.request_digest);
        output.prepend(rendered.card);
        state.activeJobs.set(job.id, true);
        waitForJob(job.id, rendered.badge);
      } catch (error) {
        output.prepend(element("pre", { className: "job-output", text: error.message }));
      } finally {
        run.disabled = false;
      }
    });
    root.append(element("article", { className: "action-card" }, [
      element("header", {}, [
        element("div", {}, [element("h3", { text: definition.label || definition.name }), element("p", { className: "muted", text: definition.description || "" })]),
        element("span", { className: `badge ${riskClass(definition.risk)}`, text: definition.risk }),
      ]),
      form,
      element("div", { className: "actions-row" }, [run]),
      output,
    ]));
  }

  async function renderInventoryOperations(root, definitions) {
    const byInventory = new Map();
    definitions.forEach(definition => {
      const items = byInventory.get(definition.inventory) || [];
      items.push(definition);
      byInventory.set(definition.inventory, items);
    });
    for (const [inventory, operations] of byInventory.entries()) {
      const section = element("article", { className: "action-card" }, [
        element("header", {}, [element("div", {}, [element("h3", { text: `Inventory · ${inventory}` })])]),
      ]);
      const output = element("div", { className: "stack" });
      section.append(output);
      root.append(section);
      try {
        const response = await api(`/api/v1/inventory?name=${encodeURIComponent(inventory)}`);
        const rows = Array.isArray(response.items) ? response.items : [];
        if (!rows.length) {
          output.append(element("p", { className: "muted", text: "No inventory items." }));
          continue;
        }
        rows.forEach(item => {
          const row = element("div", { className: "action-card" });
          const controls = element("div", { className: "actions-row" });
          for (const operation of operations) {
            const itemID = item[operation.identity_field];
            if (typeof itemID !== "string" || !itemID) continue;
            const button = element("button", { type: "button", text: operation.label || operation.name });
            button.addEventListener("click", async () => {
              button.disabled = true;
              try {
                const result = await api("/api/v1/v04/inventory-operation", { method: "POST", body: JSON.stringify({ name: operation.name, item_id: itemID }) });
                const job = result.job || {};
                const definition = (state.extensions.jobs || []).find(entry => entry.name === operation.job) || { name: operation.job, label: operation.label, risk: "caution" };
                const rendered = jobStatusCard(job, definition, result.reused === true, result.request_digest);
                row.append(rendered.card);
                state.activeJobs.set(job.id, true);
                waitForJob(job.id, rendered.badge);
              } catch (error) {
                row.append(element("pre", { className: "job-output", text: error.message }));
              } finally {
                button.disabled = false;
              }
            });
            controls.append(button);
          }
          const identity = operations.map(operation => item[operation.identity_field]).find(value => typeof value === "string") || "item";
          row.append(element("code", { text: identity }), controls);
          output.append(row);
        });
      } catch (error) {
        output.append(element("pre", { className: "job-output", text: error.message }));
      }
    }
  }

  async function init() {
    let response;
    try {
      response = await api("/api/v1/v04/capabilities");
    } catch (error) {
      if (error.status === 404) return;
      console.error("v0.4 extension", error);
      return;
    }
    state.extensions = response.extensions || {};
    const features = state.extensions.features || {};
    if (!features.typed_jobs && !features.inventory_operations) return;
    const panel = addTab();
    if (!panel) return;
    const jobsRoot = element("div", { className: "stack" });
    panel.append(jobsRoot);
    if (features.typed_jobs) {
      for (const definition of state.extensions.jobs || []) await renderJobDefinition(jobsRoot, definition);
    }
    if (features.inventory_operations) {
      await renderInventoryOperations(jobsRoot, state.extensions.inventory_operations || []);
    }
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", init);
  else init();
})();
