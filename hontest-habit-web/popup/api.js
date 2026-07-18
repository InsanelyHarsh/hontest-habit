// Shared fetch helpers against the hontest-habit-backend API. Loaded as a
// classic script by both popup.html (<script src="api.js">) and
// background.js (importScripts("popup/api.js")), so it must not use ES
// module syntax — plain function declarations attach to the shared global
// (window in the popup, self in the service worker) either way.

const API_BASE = "http://localhost:8080";

async function apiRequest(path, options) {
  const res = await fetch(API_BASE + path, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options && options.headers),
    },
  });

  if (res.status === 204) {
    return null;
  }

  const body = await res.json().catch(() => null);
  if (!res.ok) {
    const message = (body && body.error) || `request failed (${res.status})`;
    throw new Error(message);
  }
  return body;
}

async function signup(email, password) {
  const resp = await apiRequest("/auth/signup", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  await setAuth(resp);
  return resp;
}

async function login(email, password) {
  const resp = await apiRequest("/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
  await setAuth(resp);
  return resp;
}

async function getEntries(token) {
  return apiRequest("/blocklist/entries", {
    method: "GET",
    headers: { Authorization: `Bearer ${token}` },
  });
}

async function createEntry(token, entry) {
  return apiRequest("/blocklist/entries", {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: JSON.stringify(entry),
  });
}

async function deleteEntry(token, id) {
  return apiRequest(`/blocklist/entries/${id}`, {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
  });
}

function setAuth(auth) {
  return chrome.storage.local.set({ auth });
}

async function getAuth() {
  const { auth } = await chrome.storage.local.get("auth");
  return auth || null;
}

function clearAuth() {
  return chrome.storage.local.remove("auth");
}
