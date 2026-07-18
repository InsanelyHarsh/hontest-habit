const FREQUENCY_LABELS = { DL: "daily", WL: "weekly", MT: "monthly" };

const authView = document.getElementById("auth-view");
const blocklistView = document.getElementById("blocklist-view");
const logoutBtn = document.getElementById("logout-btn");

const authForm = document.getElementById("auth-form");
const authEmail = document.getElementById("auth-email");
const authPassword = document.getElementById("auth-password");
const authError = document.getElementById("auth-error");
const authSubmit = document.getElementById("auth-submit");
const authSwitchText = document.getElementById("auth-switch-text");
const authSwitchBtn = document.getElementById("auth-switch-btn");

const addEntryForm = document.getElementById("add-entry-form");
const urlInput = document.getElementById("url-input");
const startTimeInput = document.getElementById("start-time");
const endTimeInput = document.getElementById("end-time");
const resetTimeBtn = document.getElementById("reset-time-btn");
const limitCountInput = document.getElementById("limit-count");
const frequencySelect = document.getElementById("frequency-select");
const entryError = document.getElementById("entry-error");
const blockedList = document.getElementById("blocked-list");

let authMode = "login";

function showAuthView() {
  authView.classList.remove("hidden");
  blocklistView.classList.add("hidden");
  logoutBtn.classList.add("hidden");
}

async function showBlocklistView() {
  authView.classList.add("hidden");
  blocklistView.classList.remove("hidden");
  logoutBtn.classList.remove("hidden");
  await renderEntries();
}

function setAuthMode(mode) {
  authMode = mode;
  authError.classList.add("hidden");
  if (mode === "login") {
    authSubmit.textContent = "Log in";
    authSwitchText.textContent = "Need an account?";
    authSwitchBtn.textContent = "Sign up";
  } else {
    authSubmit.textContent = "Sign up";
    authSwitchText.textContent = "Already have an account?";
    authSwitchBtn.textContent = "Log in";
  }
}

authSwitchBtn.addEventListener("click", () => {
  setAuthMode(authMode === "login" ? "signup" : "login");
});

authForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  authError.classList.add("hidden");

  const email = authEmail.value.trim();
  const password = authPassword.value;

  try {
    if (authMode === "login") {
      await login(email, password);
    } else {
      await signup(email, password);
    }
    authForm.reset();
    chrome.runtime.sendMessage({ type: "refresh-blocklist-cache" }).catch(() => {});
    await showBlocklistView();
  } catch (err) {
    authError.textContent = err.message;
    authError.classList.remove("hidden");
  }
});

logoutBtn.addEventListener("click", async () => {
  await clearAuth();
  showAuthView();
});

function timeToTimestamp(timeStr) {
  if (!timeStr) return undefined;
  const [hours, minutes] = timeStr.split(":").map(Number);
  const date = new Date();
  date.setHours(hours, minutes, 0, 0);
  return date.toISOString();
}

function timestampToTime(isoStr) {
  const date = new Date(isoStr);
  return date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

resetTimeBtn.addEventListener("click", () => {
  startTimeInput.value = "00:00";
  endTimeInput.value = "23:59";
});

addEntryForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  entryError.classList.add("hidden");

  const auth = await getAuth();
  if (!auth) {
    showAuthView();
    return;
  }

  const limitCount = Number(limitCountInput.value);
  if (!Number.isFinite(limitCount) || limitCount < 1) {
    entryError.textContent = "Limit must be at least 1.";
    entryError.classList.remove("hidden");
    return;
  }

  const entry = {
    url: urlInput.value.trim(),
    daily_start_time: timeToTimestamp(startTimeInput.value),
    daily_end_time: timeToTimestamp(endTimeInput.value),
    limit: {
      frequency: frequencySelect.value,
      limit: limitCount,
    },
  };

  try {
    await createEntry(auth.token, entry);
    addEntryForm.reset();
    frequencySelect.value = "DL";
    chrome.runtime.sendMessage({ type: "refresh-blocklist-cache" }).catch(() => {});
    await renderEntries();
  } catch (err) {
    entryError.textContent = err.message;
    entryError.classList.remove("hidden");
  }
});

function entrySummary(entry) {
  const parts = [];
  if (entry.daily_start_time && entry.daily_end_time) {
    parts.push(`${timestampToTime(entry.daily_start_time)}-${timestampToTime(entry.daily_end_time)}`);
  }
  parts.push(`(${entry.limit.limit}x ${FREQUENCY_LABELS[entry.limit.frequency] || entry.limit.frequency})`);
  return parts.join(" ");
}

async function renderEntries() {
  const auth = await getAuth();
  if (!auth) {
    showAuthView();
    return;
  }

  let entries;
  try {
    entries = (await getEntries(auth.token)) || [];
  } catch (err) {
    entryError.textContent = err.message;
    entryError.classList.remove("hidden");
    return;
  }

  blockedList.innerHTML = "";
  for (const entry of entries) {
    const li = document.createElement("li");

    const info = document.createElement("div");
    info.className = "entry-info";
    const urlSpan = document.createElement("span");
    urlSpan.className = "entry-url";
    urlSpan.textContent = entry.url;
    const summarySpan = document.createElement("span");
    summarySpan.className = "entry-summary";
    summarySpan.textContent = entrySummary(entry);
    info.appendChild(urlSpan);
    info.appendChild(summarySpan);

    const removeBtn = document.createElement("button");
    removeBtn.className = "remove-btn";
    removeBtn.setAttribute("aria-label", `Remove ${entry.url}`);
    removeBtn.addEventListener("click", async () => {
      try {
        await deleteEntry(auth.token, entry.id);
        chrome.runtime.sendMessage({ type: "refresh-blocklist-cache" }).catch(() => {});
        await renderEntries();
      } catch (err) {
        entryError.textContent = err.message;
        entryError.classList.remove("hidden");
      }
    });

    li.appendChild(info);
    li.appendChild(removeBtn);
    blockedList.appendChild(li);
  }
}

(async function init() {
  setAuthMode("login");
  const auth = await getAuth();
  if (auth) {
    await showBlocklistView();
  } else {
    showAuthView();
  }
})();
