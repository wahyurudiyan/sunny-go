const rowsEl = document.getElementById("config-rows");
const emptyEl = document.getElementById("empty");
const globalErrorEl = document.getElementById("global-error");
const rowTpl = document.getElementById("tpl-row");

async function api(path, opts) {
  const res = await fetch(path, opts);
  const body = await res.json().catch(() => null);
  if (!res.ok) {
    throw new Error((body && body.error) || `request failed: ${res.status}`);
  }
  return body;
}

function buildRow(item) {
  const node = rowTpl.content.cloneNode(true);
  const tr = node.querySelector("tr");
  tr.dataset.key = item.key;

  node.querySelector('[data-role="key"]').textContent = item.key;
  node.querySelector('[data-role="source"]').textContent = item.source;

  const valueEl = node.querySelector('[data-role="value"]');
  const revealBtn = node.querySelector('[data-action="reveal"]');
  const editBtn = node.querySelector('[data-action="edit"]');
  const editForm = node.querySelector('[data-role="edit-form"]');
  const editInput = node.querySelector('[data-role="edit-input"]');
  const cancelBtn = node.querySelector('[data-action="cancel-edit"]');
  const rowErrorEl = node.querySelector('[data-role="row-error"]');

  let revealed = false;

  revealBtn.addEventListener("click", async () => {
    rowErrorEl.textContent = "";
    if (revealed) {
      valueEl.textContent = "••••••••";
      valueEl.classList.add("masked");
      revealBtn.textContent = "Reveal";
      revealed = false;
      return;
    }
    try {
      const full = await api(`/api/config/${encodeURIComponent(item.key)}`);
      valueEl.textContent = full.value === "" ? "(empty)" : full.value;
      valueEl.classList.remove("masked");
      revealBtn.textContent = "Hide";
      revealed = true;
    } catch (err) {
      rowErrorEl.textContent = err.message;
    }
  });

  if (!item.writable) {
    editBtn.remove();
  } else {
    editBtn.addEventListener("click", async () => {
      rowErrorEl.textContent = "";
      try {
        const full = await api(`/api/config/${encodeURIComponent(item.key)}`);
        editInput.value = full.value;
      } catch (err) {
        rowErrorEl.textContent = err.message;
        return;
      }
      editBtn.classList.add("hidden");
      editForm.classList.remove("hidden");
      editInput.focus();
    });

    cancelBtn.addEventListener("click", () => {
      editForm.classList.add("hidden");
      editBtn.classList.remove("hidden");
    });

    editForm.addEventListener("submit", async (e) => {
      e.preventDefault();
      rowErrorEl.textContent = "";
      try {
        await api(`/api/config/${encodeURIComponent(item.key)}`, {
          method: "PUT",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ value: editInput.value }),
        });
        editForm.classList.add("hidden");
        editBtn.classList.remove("hidden");
      } catch (err) {
        rowErrorEl.textContent = err.message;
      }
    });
  }

  return node;
}

async function refresh() {
  try {
    const items = await api("/api/config");
    globalErrorEl.textContent = "";
    rowsEl.innerHTML = "";
    emptyEl.classList.toggle("hidden", items.length > 0);
    for (const item of items) {
      rowsEl.appendChild(buildRow(item));
    }
  } catch (err) {
    globalErrorEl.textContent = err.message;
  }
}

refresh();

const events = new EventSource("/api/events");
events.onmessage = () => refresh();
