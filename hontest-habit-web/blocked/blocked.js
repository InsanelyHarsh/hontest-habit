const REASON_MESSAGES = {
  "time-window": "This site is blocked during your set daily hours.",
  "frequency-limit": "You've reached your visit limit for this site.",
};

const params = new URLSearchParams(window.location.search);
const site = params.get("site");
const reason = params.get("reason");

document.getElementById("blocked-message").textContent =
  REASON_MESSAGES[reason] || "This site is currently blocked.";

if (site) {
  document.getElementById("blocked-site").textContent = site;
}
