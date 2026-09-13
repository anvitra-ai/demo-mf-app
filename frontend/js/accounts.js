// Accounts tab: create investment account, list, and helper to populate
// account <select> elements elsewhere (mandates/orders tabs) filtered by
// the currently chosen investor.
const Accounts = (() => {
  let cache = [];

  function renderTable() {
    const tbody = document.querySelector("#accounts-table tbody");
    tbody.innerHTML = "";
    if (cache.length === 0) {
      tbody.appendChild(el(`<tr class="empty-row"><td colspan="5">No accounts yet.</td></tr>`));
      return;
    }
    cache.forEach((acc) => {
      tbody.appendChild(el(`
        <tr>
          <td>${acc.id}</td>
          <td>${acc.investor_id}</td>
          <td>${acc.code}</td>
          <td>${acc.name}</td>
          <td>${statusBadge(acc.status)}</td>
        </tr>
      `));
    });
  }

  async function load() {
    try {
      cache = await Api.get("/api/accounts");
      renderTable();
    } catch (err) {
      showBanner("Failed to load accounts: " + apiErrorMessage(err));
    }
  }

  // Populates a <select> with accounts belonging to investorId. Used by the
  // mandates and orders tabs whenever their investor picker changes.
  async function populateAccountSelect(selectEl, investorId) {
    selectEl.innerHTML = "";
    if (!investorId) {
      selectEl.appendChild(el(`<option value="">-- pick an investor first --</option>`));
      return;
    }
    try {
      const accounts = await Api.get(`/api/accounts?investor_id=${encodeURIComponent(investorId)}`);
      if (accounts.length === 0) {
        selectEl.appendChild(el(`<option value="">-- no accounts for this investor --</option>`));
        return;
      }
      accounts.forEach((acc) => {
        selectEl.appendChild(el(`<option value="${acc.id}">${acc.id} (${acc.name})</option>`));
      });
    } catch (err) {
      showBanner("Failed to load accounts for investor: " + apiErrorMessage(err));
    }
  }

  function initForm() {
    const form = document.getElementById("account-form");
    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      const fd = new FormData(form);
      const body = {
        investor_id: fd.get("investor_id"),
        code: fd.get("code").toUpperCase(),
        name: fd.get("name"),
      };
      if (!body.investor_id) {
        showBanner("Pick an investor first.");
        return;
      }
      try {
        const acc = await Api.post("/api/accounts", body);
        showBanner(`Account ${acc.id} created.`, "info");
        form.reset();
        await load();
      } catch (err) {
        showBanner("Failed to create account: " + apiErrorMessage(err));
      }
    });

    document.getElementById("btn-refresh-accounts").onclick = load;
  }

  function init() {
    initForm();
    load();
  }

  return { init, load, populateAccountSelect, getCache: () => cache };
})();
