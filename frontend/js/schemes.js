// Schemes tab: search master data, used both for browsing and for
// client-side validation of the order form (min_purchase/purchase_multiple).
const Schemes = (() => {
  function renderTable(schemes) {
    const tbody = document.querySelector("#schemes-table tbody");
    tbody.innerHTML = "";
    if (schemes.length === 0) {
      tbody.appendChild(el(`<tr class="empty-row"><td colspan="7">No schemes found.</td></tr>`));
      return;
    }
    schemes.forEach((s) => {
      tbody.appendChild(el(`
        <tr>
          <td>${s.scheme_code}</td>
          <td>${s.name}</td>
          <td>${s.amc_name || s.amc_code || ""}</td>
          <td>${s.nav ?? ""}</td>
          <td>${s.min_purchase ?? ""}</td>
          <td>${s.purchase_multiple ?? ""}</td>
          <td>${s.purchase_allowed ? "Yes" : "No"}</td>
        </tr>
      `));
    });
  }

  async function search(query) {
    try {
      const schemes = await Api.get(`/api/schemes?search=${encodeURIComponent(query || "")}`);
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
    document.getElementById("btn-scheme-search").onclick = () => {
      search(document.getElementById("scheme-search").value);
    };
    document.getElementById("scheme-search").addEventListener("keydown", (e) => {
      if (e.key === "Enter") {
        e.preventDefault();
        search(document.getElementById("scheme-search").value);
      }
    });
    search("");
  }

  return { init, search, get };
})();
