const releaseBase = "https://github.com/Falcn8/acctpass/releases/download/app-v0.1.0";

const platforms = {
  mac: {
    title: "macOS downloads",
    note: "For M1 and newer Macs, choose Apple Silicon.",
    links: [
      { label: "Apple Silicon DMG", file: "acctpass_0.1.0_aarch64.dmg", primary: true },
      { label: "Intel Mac DMG", file: "acctpass_0.1.0_x64.dmg", primary: false },
    ],
  },
  windows: {
    title: "Windows downloads",
    note: "The EXE is the simplest installer. MSI is available for managed systems.",
    links: [
      { label: "Windows setup EXE", file: "acctpass_0.1.0_x64-setup.exe", primary: true },
      { label: "Windows MSI", file: "acctpass_0.1.0_x64_en-US.msi", primary: false },
    ],
  },
  linux: {
    title: "Linux downloads",
    note: "Choose the package format used by your x64 Linux distribution.",
    links: [
      { label: "Linux AppImage", file: "acctpass_0.1.0_amd64.AppImage", primary: true },
      { label: "Debian package", file: "acctpass_0.1.0_amd64.deb", primary: false },
      { label: "RPM package", file: "acctpass-0.1.0-1.x86_64.rpm", primary: false },
    ],
  },
};

const tabs = Array.from(document.querySelectorAll("[data-platform]"));
const panel = document.querySelector("#download-panel");
const title = document.querySelector("[data-platform-title]");
const note = document.querySelector("[data-platform-note]");
const links = document.querySelector("[data-download-links]");

function createIcon(name, className = "icon") {
  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  const use = document.createElementNS("http://www.w3.org/2000/svg", "use");
  svg.classList.add(...className.split(" "));
  svg.setAttribute("aria-hidden", "true");
  use.setAttribute("href", `assets/icons.svg#icon-${name}`);
  svg.append(use);
  return svg;
}

function createDownloadLink(download) {
  const link = document.createElement("a");
  link.className = `download-button${download.primary ? " primary" : ""}`;
  link.href = `${releaseBase}/${download.file}`;
  link.setAttribute("download", "");
  link.append(document.createTextNode(download.label), createIcon("download"));
  return link;
}

function renderPlatform(key, { focus = false } = {}) {
  const platform = platforms[key];
  const selectedTab = tabs.find((tab) => tab.dataset.platform === key);
  if (!platform || !selectedTab || !panel || !title || !note || !links) return;

  tabs.forEach((tab) => {
    const selected = tab === selectedTab;
    tab.setAttribute("aria-selected", String(selected));
    tab.tabIndex = selected ? 0 : -1;
  });

  panel.setAttribute("aria-labelledby", selectedTab.id);
  title.textContent = platform.title;
  note.textContent = platform.note;
  links.replaceChildren(...platform.links.map(createDownloadLink));

  if (focus) selectedTab.focus({ preventScroll: true });
}

tabs.forEach((tab, index) => {
  tab.id = `platform-${tab.dataset.platform}`;

  tab.addEventListener("click", () => renderPlatform(tab.dataset.platform));
  tab.addEventListener("keydown", (event) => {
    if (!["ArrowLeft", "ArrowRight", "Home", "End"].includes(event.key)) return;
    event.preventDefault();

    let nextIndex = index;
    if (event.key === "ArrowLeft") nextIndex = (index - 1 + tabs.length) % tabs.length;
    if (event.key === "ArrowRight") nextIndex = (index + 1) % tabs.length;
    if (event.key === "Home") nextIndex = 0;
    if (event.key === "End") nextIndex = tabs.length - 1;
    renderPlatform(tabs[nextIndex].dataset.platform, { focus: true });
  });
});

const detectedPlatform = (() => {
  const value = `${navigator.userAgentData?.platform || ""} ${navigator.platform || ""}`.toLowerCase();
  if (value.includes("win")) return "windows";
  if (value.includes("linux")) return "linux";
  return "mac";
})();

renderPlatform(detectedPlatform);

const nav = document.querySelector("[data-nav]");
let navFrame = 0;

function updateNav() {
  nav?.classList.toggle("is-floating", window.scrollY > 32);
  navFrame = 0;
}

window.addEventListener(
  "scroll",
  () => {
    if (!navFrame) navFrame = window.requestAnimationFrame(updateNav);
  },
  { passive: true },
);

updateNav();
