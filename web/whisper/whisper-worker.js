// Web Worker for whisper.wasm transcription
// This runs in a separate thread to avoid blocking the UI

let whisperModule = null;
let modelLoaded = false;

// Handle messages from the main thread
self.onmessage = async function (e) {
  const { type, data } = e.data;

  switch (type) {
    case "init":
      await initWhisper(data.modelUrl);
      break;
    case "transcribe":
      await transcribe(data.audio);
      break;
  }
};

async function initWhisper(modelUrl) {
  try {
    self.postMessage({ type: "status", text: "Whisperモジュールを読み込み中..." });

    // Load the whisper.js module
    importScripts("whisper.js");

    // Initialize the module
    whisperModule = await Module();

    self.postMessage({ type: "status", text: "モデルをダウンロード中..." });

    // Fetch the model
    const response = await fetch(modelUrl);
    if (!response.ok) {
      throw new Error(`Model download failed: ${response.status}`);
    }

    const modelData = await response.arrayBuffer();
    const modelBytes = new Uint8Array(modelData);

    // Write model to virtual filesystem
    const modelPath = "/model.bin";
    whisperModule.FS.writeFile(modelPath, modelBytes);

    // Initialize whisper with the model
    const ret = whisperModule.init(modelPath);
    if (ret === false) {
      throw new Error("Whisper initialization failed");
    }

    modelLoaded = true;
    self.postMessage({ type: "ready" });
    self.postMessage({ type: "status", text: "" });
  } catch (err) {
    self.postMessage({ type: "error", text: `初期化エラー: ${err.message}` });
  }
}

async function transcribe(audioFloat32) {
  if (!modelLoaded) {
    self.postMessage({ type: "error", text: "モデルが読み込まれていません" });
    return;
  }

  try {
    self.postMessage({ type: "status", text: "文字起こし中..." });

    // Feed audio to whisper and get transcription
    const result = whisperModule.full_default(audioFloat32, "ja", false);

    self.postMessage({ type: "status", text: "" });

    if (result) {
      self.postMessage({ type: "transcription", text: result.trim() });
    } else {
      self.postMessage({ type: "transcription", text: "" });
    }
  } catch (err) {
    self.postMessage({ type: "error", text: `文字起こしエラー: ${err.message}` });
  }
}
