// Orders & Payments tab. Order creation is step 1 of the two-call purchase
// flow (guide §4); an order with no payment against it never gets units
// allotted, so every order row gets a "Pay" action.
const Orders = (() => {
  let cache = [];
  let paymentOrder = null;

  function renderTable() {
    const tbody = document.querySelector("#orders-table tbody");
    tbody.innerHTML = "";
    if (cache.length === 0) {
      tbody.appendChild(el(`<tr class="empty-row"><td colspan="6">No orders yet.</td></tr>`));
      return;
    }
    cache.forEach((o) => {
      const tr = el(`
        <tr>
          <td>${o.id}</td>
          <td>${o.scheme_code}</td>
          <td>${o.amount}</td>
          <td>${statusBadge(o.status)}</td>
          <td>${statusBadge(o.payment_status)}</td>
          <td></td>
        </tr>
      `);
      const actionsTd = tr.querySelector("td:last-child");

      const refreshBtn = el(`<button class="link-btn">Refresh</button>`);
      refreshBtn.onclick = () => refresh(o.id);
      actionsTd.appendChild(refreshBtn);

      actionsTd.appendChild(document.createTextNode(" · "));
      const payBtn = el(`<button class="link-btn">Pay</button>`);
      payBtn.onclick = () => openPaymentPanel(o);
      actionsTd.appendChild(payBtn);

      tbody.appendChild(tr);
    });
  }

  async function load() {
    try {
      cache = await Api.get("/api/orders");
      renderTable();
    } catch (err) {
      showBanner("Failed to load orders: " + apiErrorMessage(err));
    }
  }

  async function refresh(id) {
    try {
      const updated = await Api.post(`/api/orders/${id}/refresh`);
      showBanner(`Order ${id} status: ${updated.status} (payment: ${updated.payment_status}).`, "info");
      await load();
    } catch (err) {
      showBanner("Refresh failed: " + apiErrorMessage(err));
    }
  }

  function openPaymentPanel(order) {
    paymentOrder = order;
    const panel = document.getElementById("payment-panel");
    panel.classList.remove("hidden");
    document.getElementById("payment-order-id").textContent = order.id;
    document.getElementById("payment-result").innerHTML = "";
    Mandates.populateMandateSelect(document.getElementById("payment-mandate-select"), order.investor_id, false);
    panel.scrollIntoView({ behavior: "smooth" });
  }

  function initOrderForm() {
    const form = document.getElementById("order-form");
    const investorSelect = document.getElementById("order-investor-select");
    const accountSelect = document.getElementById("order-account-select");
    const mandateSelect = document.getElementById("order-mandate-select");
    const schemeCodeInput = document.getElementById("order-scheme-code");
    const schemeInfo = document.getElementById("order-scheme-info");

    investorSelect.addEventListener("change", () => {
      Accounts.populateAccountSelect(accountSelect, investorSelect.value);
      Mandates.populateMandateSelect(mandateSelect, investorSelect.value, true);
    });

    let lastLookup = "";
    schemeCodeInput.addEventListener("blur", async () => {
      const code = schemeCodeInput.value.trim();
      if (!code || code === lastLookup) return;
      lastLookup = code;
      try {
        const scheme = await Schemes.get(code);
        schemeInfo.textContent = `${scheme.name} -- min purchase ${scheme.min_purchase}, multiple ${scheme.purchase_multiple}, purchase allowed: ${scheme.purchase_allowed ? "yes" : "no"}`;
      } catch (err) {
        schemeInfo.textContent = `Could not look up scheme ${code}: ${apiErrorMessage(err)}`;
      }
    });

    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      const fd = new FormData(form);
      const body = {
        investor_id: fd.get("investor_id"),
        investment_account_id: fd.get("investment_account_id"),
        scheme_code: fd.get("scheme_code"),
        amount: Number(fd.get("amount")),
        purchase_type: fd.get("purchase_type"),
        bank_account_id: fd.get("bank_account_id") || undefined,
        mandate_id: fd.get("mandate_id") || undefined,
      };
      if (!body.investor_id || !body.investment_account_id) {
        showBanner("Pick an investor and investment account first.");
        return;
      }
      try {
        const order = await Api.post("/api/orders", body);
        showBanner(`Order ${order.id} created (status: ${order.status}). It still needs a payment before units are allotted.`, "info");
        form.reset();
        schemeInfo.textContent = "";
        await load();
        openPaymentPanel(order);
      } catch (err) {
        showBanner("Failed to create order: " + apiErrorMessage(err));
      }
    });

    document.getElementById("btn-refresh-orders").onclick = load;
  }

  function initPaymentForm() {
    const form = document.getElementById("payment-form");
    const modeSelect = document.getElementById("payment-mode-select");

    modeSelect.addEventListener("change", () => {
      document.querySelectorAll(".mode-fields").forEach((div) => {
        div.classList.toggle("hidden", div.dataset.mode !== modeSelect.value);
      });
    });

    form.addEventListener("submit", async (e) => {
      e.preventDefault();
      if (!paymentOrder) return;
      const fd = new FormData(form);
      const mode = fd.get("mode");
      const body = {
        investor_id: paymentOrder.investor_id,
        investment_account_id: paymentOrder.investment_account_id,
        order_ids: [paymentOrder.id],
        mode,
        mandate_id: fd.get("mandate_id") || undefined,
        vpa: fd.get("vpa") || undefined,
        bank_account_id: fd.get("bank_account_id") || undefined,
        cheque_number: fd.get("cheque_number") || undefined,
        cheque_date: fd.get("cheque_date") || undefined,
      };
      try {
        const payment = await Api.post("/api/payments", body);
        renderPaymentResult(payment);
        await load();
      } catch (err) {
        showBanner("Payment failed: " + apiErrorMessage(err));
      }
    });
  }

  function renderPaymentResult(payment) {
    const div = document.getElementById("payment-result");
    div.innerHTML = "";
    div.appendChild(el(`<p>Payment ${payment.id} initiated: ${statusBadge(payment.status)}</p>`));

    if (payment.mode === "UPI" || payment.mode === "NETBANKING") {
      if (payment.payment_url) {
        div.appendChild(el(`<p>Nothing settles until the investor completes payment here: <a href="${payment.payment_url}" target="_blank" rel="noopener">${payment.payment_url}</a></p>`));
      } else {
        div.appendChild(el(`<p>No payment_url yet -- refresh the payment shortly.</p>`));
      }
    } else if (payment.mode === "NEFT_RTGS") {
      div.appendChild(renderUtrForm(payment.id));
    } else if (payment.mode === "MANDATE") {
      div.appendChild(el(`<p>Auto-debit triggered against the mandate. No further action needed -- wait for settlement.</p>`));
    } else if (payment.mode === "CHEQUE") {
      div.appendChild(el(`<p>Cheque details registered. Funds clear out of band; no further API call needed.</p>`));
    }

    const refreshBtn = el(`<button class="secondary">Refresh payment status</button>`);
    refreshBtn.onclick = async () => {
      try {
        const updated = await Api.post(`/api/payments/${payment.id}/refresh`);
        renderPaymentResult(updated);
        await load();
      } catch (err) {
        showBanner("Refresh failed: " + apiErrorMessage(err));
      }
    };
    div.appendChild(refreshBtn);
  }

  function renderUtrForm(paymentId) {
    const wrap = el(`
      <form class="utr-form">
        <label>UTR <input name="utr" required /></label>
        <label>Transfer date <input name="transfer_date" type="date" /></label>
        <label>Bank name <input name="bank_name" /></label>
        <label>IFSC <input name="ifsc" /></label>
        <label>Account number <input name="account_no" /></label>
        <button type="submit">Submit UTR</button>
      </form>
    `);
    wrap.addEventListener("submit", async (e) => {
      e.preventDefault();
      const fd = new FormData(wrap);
      try {
        await Api.post(`/api/payments/${paymentId}/utr`, {
          utr: fd.get("utr"),
          transfer_date: fd.get("transfer_date"),
          bank_name: fd.get("bank_name"),
          ifsc: fd.get("ifsc"),
          account_no: fd.get("account_no"),
        });
        showBanner("UTR submitted.", "info");
      } catch (err) {
        showBanner("UTR submission failed: " + apiErrorMessage(err));
      }
    });
    return wrap;
  }

  function init() {
    initOrderForm();
    initPaymentForm();
    load();
  }

  return { init, load };
})();
