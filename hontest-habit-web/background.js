// Classic (non-module) service worker, so importScripts works; brings in
// login/signup/getEntries/getAuth/etc. from the shared API helper.
importScripts("popup/api.js");

const REFRESH_ALARM = "refresh-blocklist";
const VISIT_RETENTION_MS = 40 * 24 * 60 * 60 * 1000; // 40 days, covers monthly limits

async function refreshCache() {
  const auth = await getAuth();
  if (!auth) return;
  try {
    const entries = (await getEntries(auth.token)) || [];
    await chrome.storage.local.set({ blocklistCache: entries });
  } catch (err) {
    console.error("background: blocklist cache refresh failed", err);
  }
}

chrome.runtime.onInstalled.addListener(() => {
  chrome.alarms.create(REFRESH_ALARM, { periodInMinutes: 5 });
  refreshCache();
});

chrome.runtime.onStartup.addListener(() => {
  chrome.alarms.create(REFRESH_ALARM, { periodInMinutes: 5 });
  refreshCache();
});

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === REFRESH_ALARM) refreshCache();
});

// Popup pings this after login/add/remove so a change is enforced right
// away instead of waiting for the next alarm tick.
chrome.runtime.onMessage.addListener((message) => {
  if (message && message.type === "refresh-blocklist-cache") {
    refreshCache();
  }
});

chrome.storage.onChanged.addListener((changes, area) => {
  if (area === "local" && changes.auth) {
    refreshCache();
  }
});

function entryHost(entryUrl) {
  try {
    return new URL(entryUrl).host.toLowerCase();
  } catch {
    return null;
  }
}

// Daily windows only carry a time-of-day; the stored date is irrelevant, so
// this compares minutes-since-midnight and handles a window that wraps past
// midnight (e.g. 22:00-06:00).
function isWithinTimeWindow(now, startIso, endIso) {
  if (!startIso || !endIso) return false;

  const start = new Date(startIso);
  const end = new Date(endIso);
  const nowMinutes = now.getHours() * 60 + now.getMinutes();
  const startMinutes = start.getHours() * 60 + start.getMinutes();
  const endMinutes = end.getHours() * 60 + end.getMinutes();

  if (startMinutes <= endMinutes) {
    return nowMinutes >= startMinutes && nowMinutes <= endMinutes;
  }
  return nowMinutes >= startMinutes || nowMinutes <= endMinutes;
}

// Buckets a date into the period it belongs to, so visits can be counted
// per current day/week/month without needing exact rolling windows.
function periodKey(date, frequency) {
  if (frequency === "MT") {
    return `${date.getFullYear()}-${date.getMonth()}`;
  }
  if (frequency === "WL") {
    const epochDay = Math.floor(date.getTime() / 86400000);
    return `W${Math.floor((epochDay + 4) / 7)}`;
  }
  return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
}

async function isOverFrequencyLimit(now, entry) {
  const { visitLog = {} } = await chrome.storage.local.get("visitLog");
  const timestamps = visitLog[entry.id] || [];
  const key = periodKey(now, entry.limit.frequency);
  const countInPeriod = timestamps.filter(
    (t) => periodKey(new Date(t), entry.limit.frequency) === key
  ).length;
  return countInPeriod >= entry.limit.limit;
}

async function recordVisit(entryId, now) {
  const { visitLog = {} } = await chrome.storage.local.get("visitLog");
  const timestamps = visitLog[entryId] || [];
  const cutoff = now.getTime() - VISIT_RETENTION_MS;
  const pruned = timestamps.filter((t) => new Date(t).getTime() >= cutoff);
  pruned.push(now.toISOString());
  visitLog[entryId] = pruned;
  await chrome.storage.local.set({ visitLog });
}

chrome.webNavigation.onBeforeNavigate.addListener(async (details) => {
  if (details.frameId !== 0) return;

  let url;
  try {
    url = new URL(details.url);
  } catch {
    return;
  }
  if (url.protocol !== "http:" && url.protocol !== "https:") return;

  const { blocklistCache = [] } = await chrome.storage.local.get("blocklistCache");
  const entry = blocklistCache.find((e) => entryHost(e.url) === url.host.toLowerCase());
  if (!entry) return;

  const now = new Date();
  const blockedByTime = isWithinTimeWindow(now, entry.daily_start_time, entry.daily_end_time);
  const blockedByFrequency = !blockedByTime && (await isOverFrequencyLimit(now, entry));

  if (blockedByTime || blockedByFrequency) {
    const reason = blockedByTime ? "time-window" : "frequency-limit";
    const blockedUrl =
      chrome.runtime.getURL("blocked/blocked.html") +
      `?site=${encodeURIComponent(url.host)}&reason=${reason}`;
    chrome.tabs.update(details.tabId, { url: blockedUrl });
    return;
  }

  await recordVisit(entry.id, now);
});
