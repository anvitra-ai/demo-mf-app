// Thin fetch wrapper against our own backend JSON API (not mf-atlas
// directly -- the backend holds the mf-atlas credentials).
const Api = (() => {
  async function request(method, path, body) {
    const opts = { method, headers: {} };
    if (body !== undefined) {
      opts.headers["Content-Type"] = "application/json";
      opts.body = JSON.stringify(body);
    }
    const res = await fetch(path, opts);
    const isJson = (res.headers.get("content-type") || "").includes("application/json");
    const data = isJson ? await res.json() : null;
    if (!res.ok) {
      const err = new Error((data && data.error && data.error.message) || res.statusText);
      err.code = data && data.error && data.error.code;
      err.status = res.status;
      err.details = data && data.error && data.error.details;
      throw err;
    }
    return data;
  }

  return {
    get: (path) => request("GET", path),
    post: (path, body) => request("POST", path, body === undefined ? {} : body),
  };
})();

function showBanner(message, type = "error") {
  const area = document.getElementById("banner-area");
  const div = document.createElement("div");
  div.className = `banner ${type}`;
  div.textContent = message;
  area.appendChild(div);
  setTimeout(() => div.remove(), 8000);
}

function apiErrorMessage(err) {
  if (err.code) return `${err.code}: ${err.message}`;
  return err.message || String(err);
}

function statusBadge(status) {
  if (!status) return "";
  return `<span class="badge status-${status}">${status}</span>`;
}

function fmtDate(d) {
  if (!d) return "";
  const t = new Date(d);
  if (isNaN(t.getTime())) return d;
  return t.toLocaleString();
}

function el(html) {
  const tpl = document.createElement("template");
  tpl.innerHTML = html.trim();
  return tpl.content.firstChild;
}
