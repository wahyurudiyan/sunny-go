// sgo ui frontend. Plain fetch calls against the JSON API in
// ../handlers.go — no framework, no build step, matching sgo's own
// "generate a project, don't require Node to build the tool that
// generates it" stance.

const app = document.getElementById("app");
const projectDirLabel = document.getElementById("project-dir");

async function api(method, path, body) {
  const res = await fetch(path, {
    method,
    headers: body ? { "Content-Type": "application/json" } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `${method} ${path} failed (${res.status})`);
  }
  return data;
}

async function loadState() {
  const state = await api("GET", "/api/state");
  if (state.hasProject) {
    projectDirLabel.textContent = state.projectDir;
    renderDashboard(state);
  } else {
    projectDirLabel.textContent = "";
    renderCreateProject();
  }
}

function renderCreateProject() {
  app.replaceChildren(document.getElementById("tpl-create-project").content.cloneNode(true));

  const form = document.getElementById("form-init");
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const errEl = form.querySelector('[data-role="error"]');
    errEl.textContent = "";

    const fd = new FormData(form);
    const body = {
      name: fd.get("name"),
      module: fd.get("module"),
      httpFramework: fd.get("httpFramework"),
      persistenceMode: fd.get("persistenceMode"),
      db: fd.getAll("db"),
      cache: fd.getAll("cache"),
      search: fd.getAll("search"),
    };

    try {
      await api("POST", "/api/init", body);
      await loadState();
    } catch (err) {
      errEl.textContent = err.message;
    }
  });
}

function renderDashboard(state) {
  app.replaceChildren(document.getElementById("tpl-dashboard").content.cloneNode(true));

  renderConfigSummary(state.config);
  wireConfigEditor();
  wireNewServiceForm();
  renderServices(state);
}

function renderConfigSummary(cfg) {
  const dl = app.querySelector('[data-role="config-summary"]');
  const rows = [
    ["module", cfg.module],
    ["httpFramework", cfg.httpFramework],
    ["persistence.mode", cfg.persistence.mode],
    ["persistence.engines", (cfg.persistence.engines || []).join(", ") || "—"],
    ["cache", (cfg.cache || []).join(", ") || "—"],
    ["search", (cfg.search || []).join(", ") || "—"],
  ];
  dl.replaceChildren(
    ...rows.flatMap(([k, v]) => {
      const dt = document.createElement("dt");
      dt.textContent = k;
      const dd = document.createElement("dd");
      dd.textContent = v;
      return [dt, dd];
    })
  );
}

function wireConfigEditor() {
  const form = document.getElementById("form-config");
  const textarea = form.querySelector("textarea");

  api("GET", "/api/config").then((r) => {
    textarea.value = r.yaml;
  });

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const errEl = form.querySelector('[data-role="config-error"]');
    errEl.textContent = "";
    try {
      const r = await api("PUT", "/api/config", { yaml: textarea.value });
      textarea.value = r.yaml;
      await loadState();
    } catch (err) {
      errEl.textContent = err.message;
    }
  });
}

function wireNewServiceForm() {
  const form = document.getElementById("form-new-service");
  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    const errEl = form.querySelector('[data-role="new-service-error"]');
    errEl.textContent = "";
    const name = new FormData(form).get("name");
    try {
      await api("POST", "/api/services", { name });
      form.reset();
      await loadState();
    } catch (err) {
      errEl.textContent = err.message;
    }
  });
}

function renderServices(state) {
  const container = app.querySelector('[data-role="services"]');
  const rows = [];

  for (const svc of state.services || []) {
    rows.push(serviceRow(svc.name, svc));
  }
  for (const name of state.pendingProtos || []) {
    rows.push(serviceRow(name, { name, proto: true, contractGen: false, domainEntity: false, serviceImpl: false }, true));
  }

  if (rows.length === 0) {
    const p = document.createElement("p");
    p.className = "muted";
    p.textContent = "No services yet.";
    container.replaceChildren(p);
    return;
  }

  container.replaceChildren(...rows);
}

function serviceRow(name, status, pending) {
  const node = document.getElementById("tpl-service-row").content.cloneNode(true);
  const article = node.querySelector(".service");

  article.querySelector('[data-role="name"]').textContent = pending ? `${name} (not generated yet)` : name;

  const checks = article.querySelector('[data-role="checks"]');
  const items = [
    ["proto", status.proto],
    ["contract/gen", status.contractGen],
    ["domain", status.domainEntity],
    ["service", status.serviceImpl],
  ];
  checks.replaceChildren(
    ...items.map(([label, ok]) => {
      const span = document.createElement("span");
      span.className = ok ? "check-ok" : "check-missing";
      span.textContent = `${ok ? "✓" : "○"} ${label}`;
      return span;
    })
  );

  const serviceErr = article.querySelector('[data-role="service-error"]');

  article.querySelector('[data-action="generate"]').addEventListener("click", async () => {
    serviceErr.textContent = "";
    try {
      await api("POST", `/api/services/${encodeURIComponent(name)}/generate`);
      await loadState();
    } catch (err) {
      serviceErr.textContent = err.message;
    }
  });

  const editor = article.querySelector('[data-role="proto-editor"]');
  const textarea = editor.querySelector("textarea");
  const protoErr = editor.querySelector('[data-role="proto-error"]');

  article.querySelector('[data-action="toggle-proto"]').addEventListener("click", async () => {
    const willShow = editor.classList.contains("hidden");
    editor.classList.toggle("hidden");
    if (willShow && !textarea.value) {
      try {
        const r = await api("GET", `/api/services/${encodeURIComponent(name)}/proto`);
        textarea.value = r.content;
      } catch (err) {
        protoErr.textContent = err.message;
      }
    }
  });

  editor.querySelector('[data-action="save-proto"]').addEventListener("click", async () => {
    protoErr.textContent = "";
    try {
      await api("PUT", `/api/services/${encodeURIComponent(name)}/proto`, { content: textarea.value });
    } catch (err) {
      protoErr.textContent = err.message;
    }
  });

  return article;
}

loadState().catch((err) => {
  app.replaceChildren(Object.assign(document.createElement("p"), { className: "error", textContent: err.message }));
});
