// Mandates tab: register an ENACH mandate, list, and live-refresh status
// (mf-atlas refreshes mandate status live on every GET -- guide §3.2).
const Mandates = (() => {
  let cache = [];

  function renderTable() {
    const tbody = document.querySelector("#mandates-table tbody");
    tbody.innerHTML = "";
    if (cache.length === 0) {
      tbody.appendChild(el(`<tr class="empty-row"><td colspan="6">No mandates yet.</td></tr>`));
      return;
    }
    cache.forEach((m) => {
      const tr = el(`
        <tr>
          <td>${m.id}</td>
          <td>${m.investor_id}</td>
          <td>${m.type}</td>
          <td>${m.amount}</td>
          <td>${statusBadge(m.status)}</td>
          <td></td>
        </tr>
      `);
      const actionsTd = tr.querySelector("td:last-child");
      const refreshBtn = el(`<button class="link-btn">Refresh</button>`);
      refreshBtn.onclick = () => refresh(m.id);
      actionsTd.appendChild(refreshBtn);
      if (m.auth_link) {
        actionsTd.appendChild(document.createTextNode(" · "));
        actionsTd.appendChild(el(`<a href="${m.auth_link}" target="_blank" rel="noopener">Bank approval link</a>`));
      }
      tbody.appendChild(tr);
    });
  }

  async function load() {
    try {
      cache = await Api.get("/api/mandates");
      renderTable();
    } catch (err) {
      showBanner("Failed to load mandates: " + apiErrorMessage(err));
    }
  }

  async function refresh(id) {
    try {
      const updated = await Api.post(`/api/mandates/${id}/refresh`);
      showBanner(`Mandate ${id} status: ${updated.status}.`, "info");
      await load();
    } catch (err) {
      showBanner("Refresh failed: " + apiErrorMessage(err));
    }
  }

  // Populates a mandate <select> (used on the orders/payments tabs) with
  // mandates belonging to investorId.
  async function populateMandateSelect(selectEl, investorId, includeNoneOption) {
    const noneOption = includeNoneOption ? `<option value="">-- none --</option>` : "";
    selectEl.innerHTML = "";
    if (!investorId) {
      selectEl.appendChild(el(`${noneOption || '<option value="">-- pick an investor first --</option>'}`));
      return;
    }
    try {
      const mandates = await Api.get(`/api/mandates?investor_id=${encodeURIComponent(investorId)}`);
      if (noneOption) selectEl.appendChild(el(noneOption));
      if (mandates.length === 0 && !noneOption) {
        selectEl.appendChild(el(`<option value="">-- no mandates for this investor --</option>`));
        return;
      }
      mandates.forEach((m) => {
        selectEl.appendChild(el(`<option value="${m.id}">${m.id} (${m.status}, up to ${m.amount})</option>`));
      });
    } catch (err) {
      showBanner("Failed to load mandates for investor: " + apiErrorMessage(err));
    }
  }

  function initForm() {
    const form = document.getElementById("mandate-form");
    const investorSelect = document.getElementById("mandate-investor-select");
    const accountSelect = document.getElementById("mandate-account-select");

    investorSelect.addEventListener("change", () => {
      Accounts.populateAccountSelect(accountSelect, investorSelect.value);
    });

    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      const fd = new FormData(form);
      const body = {
        investor_id: fd.get("investor_id"),
        investment_account_id: fd.get("investment_account_id"),
        account_no: fd.get("account_no"),
        ifsc: fd.get("ifsc").toUpperCase(),
        account_type: fd.get("account_type"),
        amount: Number(fd.get("amount")),
        start_date: fd.get("start_date"),
        end_date: fd.get("end_date"),
      };
      if (!body.investor_id || !body.investment_account_id) {
        showBanner("Pick an investor and investment account first.");
        return;
      }
      try {
        const m = await Api.post("/api/mandates", body);
        showBanner(`Mandate ${m.id} created (status: ${m.status}). The investor's bank must still approve the eNACH registration.`, "info");
        form.reset();
        await load();
      } catch (err) {
        showBanner("Failed to create mandate: " + apiErrorMessage(err));
      }
    });

    document.getElementById("btn-refresh-mandates").onclick = load;
  }

  function init() {
    initForm();
    load();
  }

  return { init, load, populateMandateSelect, getCache: () => cache };
})();
