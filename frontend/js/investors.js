// Investors tab: registration form, list, detail view, retry/sync actions.
const Investors = (() => {
  let cache = [];

  function populateSelects() {
    document.querySelectorAll("select.investor-select").forEach((sel) => {
      const onlyRegistered = sel.classList.contains("order-investor-select");
      const prev = sel.value;
      sel.innerHTML = "";
      const list = onlyRegistered ? cache.filter((i) => i.status === "REGISTERED") : cache;
      if (list.length === 0) {
        sel.appendChild(el(`<option value="">-- ${onlyRegistered ? "no REGISTERED investors yet" : "no investors yet"} --</option>`));
      }
      list.forEach((inv) => {
        sel.appendChild(el(`<option value="${inv.id}">${inv.name || inv.id} (${inv.status})</option>`));
      });
      if (list.some((i) => i.id === prev)) sel.value = prev;
      sel.dispatchEvent(new Event("change"));
    });
  }

  function renderTable() {
    const tbody = document.querySelector("#investors-table tbody");
    tbody.innerHTML = "";
    if (cache.length === 0) {
      tbody.appendChild(el(`<tr class="empty-row"><td colspan="5">No investors yet. Register one on the left.</td></tr>`));
      return;
    }
    cache.forEach((inv) => {
      const tr = el(`
        <tr>
          <td>${inv.id}</td>
          <td>${inv.name || ""}</td>
          <td>${statusBadge(inv.status)}</td>
          <td>${fmtDate(inv.updated_at)}</td>
          <td></td>
        </tr>
      `);
      const actionsTd = tr.querySelector("td:last-child");

      const viewBtn = el(`<button class="link-btn">View</button>`);
      viewBtn.onclick = () => showDetail(inv.id);
      actionsTd.appendChild(viewBtn);

      if (inv.status === "REJECTED") {
        actionsTd.appendChild(document.createTextNode(" · "));
        const retryBtn = el(`<button class="link-btn">Retry</button>`);
        retryBtn.onclick = () => retry(inv.id);
        actionsTd.appendChild(retryBtn);
      }

      tbody.appendChild(tr);
    });
  }

  async function load() {
    try {
      cache = await Api.get("/api/investors");
      renderTable();
      populateSelects();
    } catch (err) {
      showBanner("Failed to load investors: " + apiErrorMessage(err));
    }
  }

  async function showDetail(id) {
    const panel = document.getElementById("investor-detail");
    const body = document.getElementById("investor-detail-body");
    panel.classList.remove("hidden");
    body.innerHTML = "Loading...";
    try {
      const inv = await Api.get(`/api/investors/${id}`);
      renderDetail(inv);
    } catch (err) {
      body.innerHTML = "";
      showBanner("Failed to load investor: " + apiErrorMessage(err));
    }
  }

  function renderDetail(inv) {
    const body = document.getElementById("investor-detail-body");
    body.innerHTML = "";
    body.appendChild(el(`
      <div class="detail-grid">
        <div><span class="k">ID</span><span class="v">${inv.id}</span></div>
        <div><span class="k">Status</span><span class="v">${statusBadge(inv.status)}</span></div>
        <div><span class="k">Name</span><span class="v">${inv.name || ""}</span></div>
        <div><span class="k">PAN</span><span class="v">${inv.pan || ""}</span></div>
        <div><span class="k">Updated</span><span class="v">${fmtDate(inv.updated_at)}</span></div>
        ${inv.provider_remark ? `<div><span class="k">Provider remark</span><span class="v">${inv.provider_remark}</span></div>` : ""}
        ${inv.auth_link ? `<div><span class="k">NSE confirmation link</span><span class="v"><a href="${inv.auth_link}" target="_blank" rel="noopener">${inv.auth_link}</a></span></div>` : ""}
      </div>
    `));

    const actions = el(`<div class="actions"></div>`);
    const refreshBtn = el(`<button class="secondary">Refresh status</button>`);
    refreshBtn.onclick = async () => {
      try {
        const updated = await Api.post(`/api/investors/${inv.id}/refresh`);
        renderDetail(updated);
        await load();
      } catch (err) {
        showBanner("Refresh failed: " + apiErrorMessage(err));
      }
    };
    actions.appendChild(refreshBtn);

    if (inv.status === "REGISTERED" && !inv.auth_link) {
      const linkBtn = el(`<button class="secondary">Fetch NSE confirmation link</button>`);
      linkBtn.onclick = refreshBtn.onclick;
      actions.appendChild(linkBtn);
    }

    if (inv.status === "REJECTED") {
      const retryBtn = el(`<button class="secondary">Retry (no data change)</button>`);
      retryBtn.onclick = () => retry(inv.id);
      actions.appendChild(retryBtn);
    }

    body.appendChild(actions);
  }

  async function retry(id) {
    try {
      const updated = await Api.post(`/api/investors/${id}/retry`);
      showBanner(`Investor ${id} re-queued for registration (status: ${updated.status}).`, "info");
      await load();
    } catch (err) {
      showBanner("Retry failed: " + apiErrorMessage(err));
    }
  }

  async function syncNow() {
    try {
      const result = await Api.post("/api/investors/sync");
      showBanner(`Sync complete: checked ${result.checked}, reconciled ${result.reconciled}.`, "info");
      await load();
    } catch (err) {
      showBanner("Sync failed: " + apiErrorMessage(err));
    }
  }

  function initForm() {
    const form = document.getElementById("investor-form");
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      const fd = new FormData(form);
      const body = {
        name: fd.get("name"),
        pan: fd.get("pan").toUpperCase(),
        dob: fd.get("dob"),
        mobile: fd.get("mobile"),
        email: fd.get("email"),
        gender: fd.get("gender"),
        address: {
          line1: fd.get("line1"),
          line2: fd.get("line2") || undefined,
          city: fd.get("city"),
          state: fd.get("state"),
          pincode: fd.get("pincode"),
          country: fd.get("country"),
        },
        bank_accounts: [
          {
            account_no: fd.get("account_no"),
            ifsc: fd.get("ifsc").toUpperCase(),
            account_type: fd.get("account_type"),
            is_default: true,
          },
        ],
        holding_type: fd.get("holding_type"),
        tax_status: fd.get("tax_status"),
      };
      try {
        const inv = await Api.post("/api/investors", body);
        showBanner(`Investor ${inv.id} created (status: ${inv.status}). Registration happens asynchronously -- use Refresh/Sync to check progress.`, "info");
        form.reset();
        await load();
      } catch (err) {
        showBanner("Failed to create investor: " + apiErrorMessage(err));
      }
    });

    document.getElementById("btn-refresh-investors").onclick = load;
    document.getElementById("btn-sync-investors").onclick = syncNow;
  }

  function init() {
    initForm();
    load();
  }

  return { init, load, getCache: () => cache };
})();
