const params = new URLSearchParams(location.search);
let truckId = params.get("truckId") || "";
let trucksCache = [];
const cards = new Map();
let selectedCamera = null;
let mainPlayer = null;
let refreshTimer = null;

let apiToken = localStorage.getItem("surveillanceApiToken") || "";

async function apiFetch(url, options = {}) {
  const headers = new Headers(options.headers || {});
  if (apiToken) headers.set("Authorization", `Bearer ${apiToken}`);
  const response = await fetch(url, { ...options, headers });
  if (response.status === 401) {
    apiToken = "";
    localStorage.removeItem("surveillanceApiToken");
    showLogin("登入已過期，請重新登入。");
  }
  return response;
}

const passwordPattern = /^(?=.*[a-z])(?=.*[A-Z])(?=.*\d).{8,72}$/;

function showLogin(message = "") {
  document.getElementById("loginView").hidden = false;
  document.querySelector(".topbar").hidden = true;
  document.querySelector("main").hidden = true;
  document.getElementById("loginPassword").value = "";
  const notice = document.getElementById("loginNotice");
  notice.textContent = message;
  notice.hidden = !message;
}

function showDashboard() {
  document.getElementById("loginView").hidden = true;
  document.querySelector(".topbar").hidden = false;
  document.querySelector("main").hidden = false;
}

async function login(event) {
  event.preventDefault();
  const email = document.getElementById("loginEmail").value.trim();
  const password = document.getElementById("loginPassword").value;
  if (!passwordPattern.test(password)) return showLogin("密碼至少 8 碼，且需包含英文大寫、小寫及數字。");
  const button = document.getElementById("loginButton");
  button.disabled = true;
  try {
    const response = await fetch("/api/auth/login", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ email, password }) });
    if (!response.ok) throw new Error(response.status === 401 ? "帳號或密碼錯誤。" : "登入失敗，請稍後再試。");
    const session = await response.json();
    apiToken = session.token;
    localStorage.setItem("surveillanceApiToken", apiToken);
    showDashboard();
    await showTruckSelection(false);
  } catch (error) { showLogin(error.message); }
  finally { button.disabled = false; }
}

async function logout() {
  if (apiToken) await apiFetch("/api/auth/logout", { method: "POST" }).catch(() => {});
  apiToken = "";
  localStorage.removeItem("surveillanceApiToken");
  await closeViewerIfOpen();
  showLogin();
  document.getElementById("loginPassword").focus();
}

class WHEPPlayer {
  constructor(video, onState) {
    this.video = video;
    this.onState = onState;
    this.pc = null;
    this.sessionUrl = null;
    this.token = null;
  }

  async start(url, token) {
    await this.close();
    url = publicStreamUrl(url);
    this.token = token;
    this.pc = new RTCPeerConnection();
    this.pc.addTransceiver("video", { direction: "recvonly" });
    this.pc.ontrack = (event) => {
      this.video.srcObject = event.streams[0];
      this.video.play().catch(() => {});
      this.onState?.("playing");
    };
    this.pc.onconnectionstatechange = () => {
      if (["failed", "disconnected", "closed"].includes(this.pc?.connectionState)) {
        this.onState?.("error");
      }
    };

    const offer = await this.pc.createOffer();
    await this.pc.setLocalDescription(offer);
    await this.waitForIceGathering();
    const response = await fetch(url, {
      method: "POST",
      headers: { "Authorization": `Bearer ${token}`, "Content-Type": "application/sdp" },
      body: this.pc.localDescription.sdp,
    });
    if (!response.ok) throw new Error(`WHEP ${response.status}`);
    const locationHeader = response.headers.get("Location");
    this.sessionUrl = locationHeader ? new URL(locationHeader, url).href : null;
    await this.pc.setRemoteDescription({ type: "answer", sdp: await response.text() });
  }

  waitForIceGathering() {
    if (this.pc.iceGatheringState === "complete") return Promise.resolve();
    return new Promise((resolve) => {
      const timeout = setTimeout(resolve, 2500);
      this.pc.addEventListener("icegatheringstatechange", () => {
        if (this.pc?.iceGatheringState === "complete") {
          clearTimeout(timeout);
          resolve();
        }
      });
    });
  }

  async close() {
    if (this.sessionUrl && this.token) {
      fetch(this.sessionUrl, { method: "DELETE", headers: { "Authorization": `Bearer ${this.token}` } }).catch(() => {});
    }
    this.pc?.close();
    this.pc = null;
    this.sessionUrl = null;
    if (this.video) this.video.srcObject = null;
  }
}

function publicStreamUrl(url) {
  const parsed = new URL(url, location.href);
  if (location.protocol === "https:") {
    parsed.protocol = "https:";
    parsed.hostname = location.hostname;
    parsed.port = "";
    return parsed.href;
  }
  if (parsed.hostname === "localhost" || parsed.hostname === "127.0.0.1") {
    parsed.hostname = location.hostname;
  }
  return parsed.href;
}

const truckView = document.getElementById("truckView");
const cameraView = document.getElementById("cameraView");
const truckList = document.getElementById("truckList");
const truckTemplate = document.getElementById("truckCardTemplate");
const grid = document.getElementById("cameraGrid");
const cameraTemplate = document.getElementById("cameraCardTemplate");
const notice = document.getElementById("notice");
const truckNotice = document.getElementById("truckNotice");
const summary = document.querySelector(".summary");
const summaryText = document.getElementById("summaryText");
const truckLabel = document.getElementById("truckLabel");
const truckSelect = document.getElementById("truckSelect");
const dialog = document.getElementById("viewerDialog");
const mainVideo = document.getElementById("mainVideo");
const viewerState = document.getElementById("viewerState");
const recordingList = document.getElementById("recordingList");

document.getElementById("reloadTrucksButton").addEventListener("click", loadTruckSelection);
document.getElementById("backToTrucksButton").addEventListener("click", showTruckSelection);
document.getElementById("refreshButton").addEventListener("click", () => refreshCameras(true));
document.getElementById("closeViewer").addEventListener("click", closeViewer);
document.getElementById("liveButton").addEventListener("click", startMainStream);
document.getElementById("historyButton").addEventListener("click", loadRecordings);
document.getElementById("loginForm").addEventListener("submit", login);
document.getElementById("logoutButton").addEventListener("click", logout);
truckSelect.addEventListener("change", () => enterCameraGrid(truckSelect.value, true));
dialog.addEventListener("cancel", (event) => { event.preventDefault(); closeViewer(); });
window.addEventListener("popstate", () => {
  const nextTruckId = new URLSearchParams(location.search).get("truckId") || "";
  if (nextTruckId) enterCameraGrid(nextTruckId, false);
  else showTruckSelection(false);
});

function setSummary(text, connected = false) {
  summaryText.textContent = text;
  summary.classList.toggle("connected", connected);
}

function showNotice(element, message) {
  element.textContent = message;
  element.hidden = false;
}

function hideNotice(element) {
  element.hidden = true;
}

function statusLabel(status) {
  return status === "online" ? "在線" : "離線";
}

function stopRefreshTimer() {
  if (refreshTimer) clearInterval(refreshTimer);
  refreshTimer = null;
}

function startRefreshTimer() {
  stopRefreshTimer();
  refreshTimer = setInterval(refreshCameras, 5000);
}

async function loadTrucks() {
  const response = await apiFetch("/api/trucks", { cache: "no-store" });
  if (!response.ok) throw new Error(`API ${response.status}`);
  trucksCache = await response.json();
  return trucksCache;
}

function renderTruckSelect(trucks) {
  truckSelect.replaceChildren();
  for (const truck of trucks) {
    const option = document.createElement("option");
    option.value = truck.truckId;
    option.textContent = `${truck.truckId.toUpperCase()} · ${truck.plateNo} · ${statusLabel(truck.status)}`;
    truckSelect.appendChild(option);
  }
  truckSelect.value = truckId;
}

function renderTruckCards(trucks) {
  truckList.replaceChildren();
  for (const truck of trucks) {
    const element = truckTemplate.content.firstElementChild.cloneNode(true);
    element.querySelector(".truck-id").textContent = truck.truckId.toUpperCase();
    element.querySelector(".truck-plate").textContent = truck.plateNo;
    const badge = element.querySelector(".status-badge");
    badge.textContent = statusLabel(truck.status);
    badge.className = `status-badge ${truck.status}`;
    element.querySelector(".truck-open").addEventListener("click", () => enterCameraGrid(truck.truckId, true));
    truckList.appendChild(element);
  }
}

async function loadTruckSelection() {
  try {
    const trucks = await loadTrucks();
    hideNotice(truckNotice);
    renderTruckCards(trucks);
    setSummary(`${trucks.length} 台車機可選擇`, true);
  } catch (error) {
    showNotice(truckNotice, `讀取車機失敗：${error.message}`);
    setSummary("連線異常，請稍後再試");
  }
}

async function showTruckSelection(updateHistory = true) {
  truckId = "";
  stopRefreshTimer();
  await closeViewerIfOpen();
  await clearCameraGrid();
  cameraView.hidden = true;
  truckView.hidden = false;
  if (updateHistory) history.pushState(null, "", location.pathname);
  await loadTruckSelection();
}

async function enterCameraGrid(nextTruckId, updateHistory = true) {
  if (!nextTruckId) return showTruckSelection(updateHistory);
  const changedTruck = nextTruckId !== truckId;
  truckId = nextTruckId;
  if (updateHistory) history.pushState(null, "", `?truckId=${encodeURIComponent(truckId)}`);

  truckView.hidden = true;
  cameraView.hidden = false;
  updateTruckLabel();

  try {
    const trucks = trucksCache.length ? trucksCache : await loadTrucks();
    renderTruckSelect(trucks);
  } catch {
    truckSelect.replaceChildren();
  }

  if (changedTruck) {
    await closeViewerIfOpen();
    await clearCameraGrid();
  }

  await refreshCameras(true);
  startRefreshTimer();
}

function createCard(camera) {
  const element = cameraTemplate.content.firstElementChild.cloneNode(true);
  const video = element.querySelector("video");
  const frame = element.querySelector(".video-frame");
  element.querySelector(".camera-id").textContent = camera.cameraId.toUpperCase();
  element.querySelector(".camera-name").textContent = camera.name;
  element.querySelector(".camera-open").addEventListener("click", () => openViewer(camera.cameraId));
  grid.appendChild(element);
  const card = { element, video, frame, player: new WHEPPlayer(video, state => frame.classList.toggle("playing", state === "playing")), camera };
  cards.set(camera.cameraId, card);
  return card;
}

function updateTruckLabel() {
  truckLabel.textContent = truckId.toUpperCase();
}

async function clearCameraGrid() {
  for (const card of cards.values()) await card.player.close();
  cards.clear();
  grid.replaceChildren();
}

async function refreshCameras(forceRestart = false) {
  if (!truckId) return;
  try {
    const response = await apiFetch(`/api/trucks/${encodeURIComponent(truckId)}/cameras`, { cache: "no-store" });
    if (!response.ok) throw new Error(`API ${response.status}`);
    const cameras = await response.json();
    hideNotice(notice);
    const onlineCount = cameras.filter(camera => camera.status === "online").length;
    setSummary(`${onlineCount} / ${cameras.length} 支攝影機在線`, true);

    const seen = new Set();
    for (const camera of cameras) {
      seen.add(camera.cameraId);
      const card = cards.get(camera.cameraId) || createCard(camera);
      card.camera = camera;
      const badge = card.element.querySelector(".status-badge");
      badge.textContent = statusLabel(camera.status);
      badge.className = `status-badge ${camera.status}`;
      if (!dialog.open && camera.status === "online" && (forceRestart || !card.frame.classList.contains("playing"))) {
        card.player.start(camera.subUrl, camera.subToken).catch(() => card.frame.classList.remove("playing"));
      }
      if (camera.status !== "online") {
        card.player.close();
        card.frame.classList.remove("playing");
      }
    }

    for (const [cameraId, card] of cards) {
      if (!seen.has(cameraId)) {
        await card.player.close();
        card.element.remove();
        cards.delete(cameraId);
      }
    }
  } catch (error) {
    showNotice(notice, `讀取攝影機失敗：${error.message}`);
    setSummary("連線異常，請稍後再試");
  }
}

async function openViewer(cameraId) {
  selectedCamera = cards.get(cameraId)?.camera;
  if (!selectedCamera) return;
  for (const card of cards.values()) await card.player.close();
  document.getElementById("viewerCameraId").textContent = selectedCamera.cameraId.toUpperCase();
  document.getElementById("viewerTitle").textContent = selectedCamera.name;
  recordingList.hidden = true;
  recordingList.replaceChildren();
  dialog.showModal();
  await startMainStream();
}

async function startMainStream() {
  if (!selectedCamera) return;
  mainVideo.pause();
  mainVideo.removeAttribute("src");
  mainVideo.load();
  viewerState.textContent = "準備載入即時影像";
  viewerState.classList.remove("ready");
  recordingList.hidden = true;
  try {
    const response = await apiFetch(`/api/trucks/${encodeURIComponent(truckId)}/cameras/${encodeURIComponent(selectedCamera.cameraId)}/play`, { method: "POST" });
    if (!response.ok) throw new Error(`API ${response.status}`);
    const stream = await response.json();
    mainPlayer ||= new WHEPPlayer(mainVideo, state => viewerState.classList.toggle("ready", state === "playing"));
    await mainPlayer.start(stream.url, stream.accessToken);
  } catch (error) {
    viewerState.textContent = `即時影像載入失敗：${error.message}`;
  }
}

async function loadRecordings() {
  if (!selectedCamera) return;
  await mainPlayer?.close();
  viewerState.textContent = "讀取歷史錄影";
  viewerState.classList.remove("ready");
  try {
    const response = await apiFetch(`/api/trucks/${encodeURIComponent(truckId)}/recordings?cameraId=${encodeURIComponent(selectedCamera.cameraId)}`);
    if (!response.ok) throw new Error(`API ${response.status}`);
    const recordings = await response.json();
    recordingList.replaceChildren();
    for (const recording of recordings) {
      const button = document.createElement("button");
      button.className = "recording-item";
      button.innerHTML = `${new Date(recording.start).toLocaleString()}<small>${Math.round(recording.durationSeconds)} 秒</small>`;
      button.addEventListener("click", () => playRecording(recording.url));
      recordingList.appendChild(button);
    }
    if (!recordings.length) recordingList.textContent = "尚無歷史錄影";
    recordingList.hidden = false;
    viewerState.textContent = "選擇一段歷史錄影";
  } catch (error) {
    viewerState.textContent = `讀取歷史錄影失敗：${error.message}`;
  }
}

function playRecording(url) {
  mainVideo.srcObject = null;
  mainVideo.src = url;
  mainVideo.muted = false;
  mainVideo.play().catch(() => {});
  viewerState.classList.add("ready");
}

async function closeViewerIfOpen() {
  if (dialog.open) await closeViewer(false);
}

async function closeViewer(restartGrid = true) {
  await mainPlayer?.close();
  mainVideo.pause();
  mainVideo.removeAttribute("src");
  mainVideo.load();
  selectedCamera = null;
  if (dialog.open) dialog.close();
  if (restartGrid && truckId) refreshCameras(true);
}

async function init() {
  if (!apiToken) {
    showLogin();
    return;
  }
  showDashboard();
  if (truckId) await enterCameraGrid(truckId, false);
  else await showTruckSelection(false);
}

init();
window.addEventListener("beforeunload", () => {
  stopRefreshTimer();
  mainPlayer?.close();
  for (const card of cards.values()) card.player.close();
});
