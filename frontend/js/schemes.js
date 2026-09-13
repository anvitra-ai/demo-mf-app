// Schemes tab: search master data, used both for browsing and for
// client-side validation of the order form (min_purchase/purchase_multiple).
const Schemes = (() => {
  function renderTable(schemes) {
    const tbody = document.querySelector("#schemes-table tbody");
    tbody.innerHTML = "";
    if (schemes.length === 0) {
      tbody.appendChild(el(`<tr class="empty-row"><td colspan="9">No schemes found.</td></tr>`));
      return;
    }
    schemes.forEach((s) => {
      const tr = el(`
        <tr>
          <td>${s.scheme_code}</td>
          <td>${s.name}</td>
          <td>${s.amc_name || s.amc_code || ""}</td>
          <td>${s.category || ""}</td>
          <td>${s.nav ?? ""}</td>
          <td>${s.min_purchase ?? ""}</td>
          <td>${s.purchase_multiple ?? ""}</td>
          <td>${s.purchase_allowed ? "Yes" : "No"}</td>
          <td></td>
        </tr>
      `);
      const useBtn = el(`<button class="link-btn">Use</button>`);
      useBtn.onclick = () => useInOrder(s.scheme_code);
      tr.querySelector("td:last-child").appendChild(useBtn);
      tbody.appendChild(tr);
    });
  }

  function useInOrder(code) {
    gotoTab("orders");
    const input = document.getElementById("order-scheme-code");
    input.value = code;
    input.dispatchEvent(new Event("blur"));
    input.scrollIntoView({ behavior: "smooth", block: "center" });
  }

  function currentFilters() {
    const q = {
      search: document.getElementById("scheme-search").value,
      amc: document.getElementById("scheme-filter-amc").value,
      category: document.getElementById("scheme-filter-category").value,
      plan: document.getElementById("scheme-filter-plan").value,
      option: document.getElementById("scheme-filter-option").value,
    };
    if (document.getElementById("scheme-filter-purchase-allowed").checked) {
      q.purchase_allowed = "true";
    }
    return q;
  }

  async function search(filters) {
    try {
      const params = new URLSearchParams(
        Object.fromEntries(Object.entries(filters).filter(([, v]) => v))
      );
      const schemes = await Api.get(`/api/schemes?${params.toString()}`);
      renderTable(schemes);
    } catch (err) {
      showBanner("Scheme search failed: " + apiErrorMessage(err));
    }
  }

  // Fetches a single scheme by code, used by the order form to validate the
  // amount before submitting (guide §4.0).
  async function get(code) {
    return Api.get(`/api/schemes/${encodeURIComponent(code)}`);
  }

  function init() {
    document.getElementById("btn-scheme-search").onclick = () => search(currentFilters());
    document.getElementById("btn-scheme-filter").onclick = () => search(currentFilters());
    document.getElementById("btn-scheme-filter-clear").onclick = () => {
      document.getElementById("scheme-filter-amc").value = "";
      document.getElementById("scheme-filter-category").value = "";
      document.getElementById("scheme-filter-plan").value = "";
      document.getElementById("scheme-filter-option").value = "";
      document.getElementById("scheme-filter-purchase-allowed").checked = false;
      search(currentFilters());
    };
    document.getElementById("scheme-search").addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        search(currentFilters());
      }
    });
    search(currentFilters());
  }

  return { init, search, get };
})();
