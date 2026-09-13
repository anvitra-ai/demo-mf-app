// Tab navigation + app bootstrap + light polling so status changes driven
// by webhooks/reconciliation on the backend show up without a manual click.
document.addEventListener("DOMContentLoaded", () => {
  const tabButtons = document.querySelectorAll(".tab-btn");
  const panels = document.querySelectorAll(".tab-panel");

  tabButtons.forEach((btn) => {
    btn.addEventListener("click", () => {
      tabButtons.forEach((b) => b.classList.remove("active"));
      panels.forEach((p) => p.classList.remove("active"));
      btn.classList.add("active");
      document.getElementById(`tab-${btn.dataset.tab}`).classList.add("active");
    });
  });

  Investors.init();
  Accounts.init();
  Mandates.init();
  Schemes.init();
  Orders.init();

  // Async statuses (investor/mandate/order/payment) change server-side via
  // webhook + reconciliation poller; refresh the lists periodically so the
  // UI reflects that without requiring a manual click.
  setInterval(() => {
    Investors.load();
    Accounts.load();
    Mandates.load();
    Orders.load();
  }, 8000);
});
