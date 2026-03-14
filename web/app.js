"use strict";

const $ = (sel) => document.querySelector(sel);

// --- State ---
let ws = null;
let audioQueue = [];
let isPlaying = false;
let isRecording = false;
let whisperReady = false;
let whisperWorker = null;
let audioContext = null;
let mediaStream = null;
let audioChunks = [];
let scriptProcessor = null;

// --- Elements ---
const setupScreen = $("#setup-screen");
const sessionScreen = $("#session-screen");
const ruleSelect = $("#rule-select");
const scenarioSelect = $("#scenario-select");
const startBtn = $("#start-btn");
const chatLog = $("#chat-log");
const textInput = $("#text-input");
const sendBtn = $("#send-btn");
const recordBtn = $("#record-btn");
const connectionStatus = $("#connection-status");
const recordingStatus = $("#recording-status");

// --- Whisper.wasm ---
function initWhisper() {
  whisperWorker = new Worker("whisper/whisper-worker.js");

  whisperWorker.onmessage = (e) => {
    const { type, text } = e.data;
    switch (type) {
      case "ready":
        whisperReady = true;
        recordBtn.title = "押して話す";
        recordingStatus.textContent = "";
        break;
      case "status":
        recordingStatus.textContent = text;
        break;
      case "transcription":
        if (text) {
          sendMessage(text);
        }
        setInputEnabled(true);
        break;
      case "error":
        console.error("whisper error:", text);
        recordingStatus.textContent = text;
        setInputEnabled(true);
        break;
    }
  };

  recordBtn.title = "モデル読み込み中...";
  recordingStatus.textContent = "Whisperモデル読み込み中...";

  whisperWorker.postMessage({
    type: "init",
    data: { modelUrl: "whisper/ggml-base.bin" },
  });
}

// --- Init ---
async function init() {
  const [rules, scenarios] = await Promise.all([
    fetch("/api/rules").then((r) => r.json()),
    fetch("/api/scenarios").then((r) => r.json()),
  ]);

  rules.forEach((r) => {
    const opt = document.createElement("option");
    opt.value = r.name;
    opt.textContent = `${r.name} - ${r.description}`;
    ruleSelect.appendChild(opt);
  });

  scenarios.forEach((s) => {
    const opt = document.createElement("option");
    opt.value = s.name;
    opt.textContent = `${s.name} - ${s.description}`;
    scenarioSelect.appendChild(opt);
  });

  // Start loading whisper in background
  initWhisper();
}

// --- WebSocket ---
function connectWebSocket() {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  ws = new WebSocket(`${protocol}//${location.host}/ws`);

  ws.binaryType = "arraybuffer";

  ws.onclose = () => {
    connectionStatus.textContent = "切断";
    connectionStatus.classList.remove("connected");
  };

  ws.onmessage = (event) => {
    if (event.data instanceof ArrayBuffer) {
      queueAudio(event.data);
    } else {
      const msg = JSON.parse(event.data);
      handleServerMessage(msg);
    }
  };
}

function handleServerMessage(msg) {
  switch (msg.type) {
    case "response":
      addMessage("gm", "GM", msg.text);
      setInputEnabled(true);
      break;
    case "error":
      addMessage("error", "エラー", msg.text);
      setInputEnabled(true);
      break;
  }
}

function sendMessage(text) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return;
  if (!text.trim()) return;

  addMessage("player", "あなた", text);
  ws.send(JSON.stringify({ type: "message", text }));
  textInput.value = "";
  setInputEnabled(false);
}

// --- Audio Playback ---
function queueAudio(arrayBuffer) {
  audioQueue.push(arrayBuffer);
  if (!isPlaying) playNext();
}

function playNext() {
  if (audioQueue.length === 0) {
    isPlaying = false;
    return;
  }
  isPlaying = true;
  const buffer = audioQueue.shift();
  const blob = new Blob([buffer], { type: "audio/wav" });
  const url = URL.createObjectURL(blob);
  const audio = new Audio(url);
  audio.onended = () => {
    URL.revokeObjectURL(url);
    playNext();
  };
  audio.play().catch((err) => {
    console.error("audio play error:", err);
    playNext();
  });
}

// --- Recording with 16kHz PCM conversion ---
async function startRecording() {
  if (!whisperReady) {
    recordingStatus.textContent = "Whisperがまだ準備中です";
    return;
  }

  try {
    mediaStream = await navigator.mediaDevices.getUserMedia({
      audio: {
        channelCount: 1,
        sampleRate: 16000,
      },
    });

    // Create AudioContext at 16kHz for whisper
    audioContext = new AudioContext({ sampleRate: 16000 });
    const source = audioContext.createMediaStreamSource(mediaStream);

    // Use ScriptProcessorNode to capture raw PCM
    // (AudioWorklet would be better but more complex to set up)
    scriptProcessor = audioContext.createScriptProcessor(4096, 1, 1);
    audioChunks = [];

    scriptProcessor.onaudioprocess = (e) => {
      if (isRecording) {
        const data = e.inputBuffer.getChannelData(0);
        audioChunks.push(new Float32Array(data));
      }
    };

    source.connect(scriptProcessor);
    scriptProcessor.connect(audioContext.destination);

    isRecording = true;
    recordBtn.classList.add("recording");
    recordingStatus.textContent = "録音中...";
  } catch (err) {
    console.error("recording error:", err);
    recordingStatus.textContent = "マイクへのアクセスに失敗しました";
  }
}

function stopRecording() {
  if (!isRecording) return;

  isRecording = false;
  recordBtn.classList.remove("recording");
  recordingStatus.textContent = "文字起こし中...";

  // Stop media stream
  if (mediaStream) {
    mediaStream.getTracks().forEach((t) => t.stop());
    mediaStream = null;
  }

  // Disconnect audio nodes
  if (scriptProcessor) {
    scriptProcessor.disconnect();
    scriptProcessor = null;
  }

  if (audioContext) {
    audioContext.close();
    audioContext = null;
  }

  // Merge chunks into single Float32Array
  if (audioChunks.length === 0) {
    recordingStatus.textContent = "";
    return;
  }

  const totalLength = audioChunks.reduce((sum, c) => sum + c.length, 0);
  const merged = new Float32Array(totalLength);
  let offset = 0;
  for (const chunk of audioChunks) {
    merged.set(chunk, offset);
    offset += chunk.length;
  }
  audioChunks = [];

  // Send to whisper worker for transcription
  setInputEnabled(false);
  whisperWorker.postMessage({
    type: "transcribe",
    data: { audio: merged },
  });
}

// --- UI Helpers ---
function addMessage(type, label, text) {
  const div = document.createElement("div");
  div.className = `message ${type}`;

  const labelEl = document.createElement("div");
  labelEl.className = "label";
  labelEl.textContent = label;
  div.appendChild(labelEl);

  const content = document.createElement("div");
  content.textContent = text;
  div.appendChild(content);

  chatLog.appendChild(div);
  chatLog.scrollTop = chatLog.scrollHeight;
}

function setInputEnabled(enabled) {
  textInput.disabled = !enabled;
  sendBtn.disabled = !enabled;
  recordBtn.disabled = !enabled;
}

// --- Event Listeners ---
startBtn.addEventListener("click", () => {
  connectWebSocket();

  ws.onopen = () => {
    connectionStatus.textContent = "接続中";
    connectionStatus.classList.add("connected");

    ws.send(
      JSON.stringify({
        type: "start",
        rule: ruleSelect.value,
        scenario: scenarioSelect.value,
      })
    );

    setupScreen.hidden = true;
    sessionScreen.hidden = false;
    setInputEnabled(false);
  };
});

sendBtn.addEventListener("click", () => {
  sendMessage(textInput.value);
});

textInput.addEventListener("keydown", (e) => {
  if (e.key === "Enter" && !e.isComposing) {
    sendMessage(textInput.value);
  }
});

recordBtn.addEventListener("mousedown", () => {
  if (!isRecording) startRecording();
});

recordBtn.addEventListener("mouseup", () => {
  if (isRecording) stopRecording();
});

recordBtn.addEventListener("mouseleave", () => {
  if (isRecording) stopRecording();
});

// --- Start ---
init();
