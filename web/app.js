const truckId = new URLSearchParams(location.search).get("truckId") || "truck001";
const cards = new Map();
let selectedCamera = null;
let mainPlayer = null;
let refreshTimer = null;

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

const grid = document.getElementById("cameraGrid");
const template = document.getElementById("cameraCardTemplate");
const notice = document.getElementById("notice");
const summary = document.querySelector(".summary");
const summaryText = document.getElementById("summaryText");
const dialog = document.getElementById("viewerDialog");
const mainVideo = document.getElementById("mainVideo");
const viewerState = document.getElementById("viewerState");
const recordingList = document.getElementById("recordingList");

document.getElementById("truckLabel").textContent = truckId.toUpperCase();
document.getElementById("refreshButton").addEventListener("click", () => refreshCameras(true));
document.getElementById("closeViewer").addEventListener("click", closeViewer);
document.getElementById("liveButton").addEventListener("click", startMainStream);
document.getElementById("historyButton").addEventListener("click", loadRecordings);
dialog.addEventListener("cancel", (event) => { event.preventDefault(); closeViewer(); });

function createCard(camera) {
  const element = template.content.firstElementChild.cloneNode(true);
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

async function refreshCameras(forceRestart = false) {
  try {
    const response = await fetch(`/api/trucks/${encodeURIComponent(truckId)}/cameras`, { cache: "no-store" });
    if (!response.ok) throw new Error(`API ${response.status}`);
    const cameras = await response.json();
    notice.hidden = true;
    summary.classList.add("connected");
    const onlineCount = cameras.filter(camera => camera.status === "online").length;
    summaryText.textContent = `${onlineCount} / ${cameras.length} 路在線`;

    for (const camera of cameras) {
      const card = cards.get(camera.cameraId) || createCard(camera);
      card.camera = camera;
      const badge = card.element.querySelector(".status-badge");
      badge.textContent = camera.status === "online" ? "在線" : "離線";
      badge.className = `status-badge ${camera.status}`;
      if (!dialog.open && camera.status === "online" && (forceRestart || !card.frame.classList.contains("playing"))) {
        card.player.start(camera.subUrl, camera.subToken).catch(() => card.frame.classList.remove("playing"));
      }
      if (camera.status !== "online") {
        card.player.close();
        card.frame.classList.remove("playing");
      }
    }
  } catch (error) {
    notice.textContent = `無法取得攝影機資料：${error.message}`;
    notice.hidden = false;
    summary.classList.remove("connected");
    summaryText.textContent = "伺服器連線失敗";
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
  viewerState.textContent = "正在載入高畫質即時影像…";
  viewerState.classList.remove("ready");
  recordingList.hidden = true;
  try {
    const response = await fetch(`/api/trucks/${encodeURIComponent(truckId)}/cameras/${encodeURIComponent(selectedCamera.cameraId)}/play`, { method: "POST" });
    if (!response.ok) throw new Error(`API ${response.status}`);
    const stream = await response.json();
    mainPlayer ||= new WHEPPlayer(mainVideo, state => viewerState.classList.toggle("ready", state === "playing"));
    await mainPlayer.start(stream.url, stream.accessToken);
  } catch (error) {
    viewerState.textContent = `無法播放即時影像：${error.message}`;
  }
}

async function loadRecordings() {
  if (!selectedCamera) return;
  await mainPlayer?.close();
  viewerState.textContent = "正在讀取歷史錄影…";
  viewerState.classList.remove("ready");
  try {
    const response = await fetch(`/api/trucks/${encodeURIComponent(truckId)}/recordings?cameraId=${encodeURIComponent(selectedCamera.cameraId)}`);
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
    if (!recordings.length) recordingList.textContent = "目前沒有錄影資料";
    recordingList.hidden = false;
    viewerState.textContent = "請選擇一段錄影";
  } catch (error) {
    viewerState.textContent = `無法取得歷史錄影：${error.message}`;
  }
}

function playRecording(url) {
  mainVideo.srcObject = null;
  mainVideo.src = url;
  mainVideo.muted = false;
  mainVideo.play().catch(() => {});
  viewerState.classList.add("ready");
}

async function closeViewer() {
  await mainPlayer?.close();
  mainVideo.pause();
  mainVideo.removeAttribute("src");
  mainVideo.load();
  selectedCamera = null;
  dialog.close();
  refreshCameras(true);
}

refreshCameras();
refreshTimer = setInterval(refreshCameras, 5000);
window.addEventListener("beforeunload", () => {
  clearInterval(refreshTimer);
  mainPlayer?.close();
  for (const card of cards.values()) card.player.close();
});
