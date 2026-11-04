// Page controller: wires the URL field, player, trimmer, output choice and job flow.
//
// Simple mode never waits on the server: the player loads straight from the
// link and the video info request only adds the title. Advanced mode fetches
// the format list when it is opened.

import { createClip, getJob, getVideoInfo } from "./api.js";
import {
    audioMeta,
    audioTitle,
    estimateBytes,
    formatBytes,
    renderList,
    splitFormats,
    updateSizes,
    videoMeta,
    videoTitle,
    videoCodecName,
    audioCodecName,
} from "./formats.js";
import { Player } from "./player.js";
import { initTheme } from "./theme.js";
import { addRecent, clearRecent, loadRecent } from "./recent.js";
import { formatAgo, formatTime, formatTimeLeft, parseTime, toHHMMSS } from "./time.js";
import { toast } from "./toast.js";
import { Trimmer } from "./trimmer.js";
import { extractVideoId, linkFromLocation, linkStartTime } from "./youtube-url.js";

const DEFAULT_LENGTH = 30;
const POLL_MS = 1500;
const POLL_MAX_BACKOFF_MS = 15000;
const JOB_DEADLINE_MS = 15 * 60 * 1000;
// Poll errors worth retrying: rate limited, or the server is briefly unavailable.
const TRANSIENT_STATUSES = new Set([0, 429, 502, 503, 504]);
// Preview arms only once playback is clearly inside the range.
const PREVIEW_ARM_MARGIN = 0.25;
const TIME_HELP = "Use seconds (34), m:ss (1:28) or h:mm:ss (1:09:24).";
const INVALID_URL = "That doesn't look like a YouTube video link. Paste a watch, youtu.be, Shorts or live URL.";

const $ = (id) => document.getElementById(id);
const el = {
    form: $("clipForm"),
    url: $("url"),
    urlError: $("urlError"),
    editor: $("editor"),
    playerFrame: document.querySelector(".player-frame"),
    player: $("player"),
    playerStatus: $("playerStatus"),
    playerSpinner: $("playerSpinner"),
    videoMeta: $("videoMeta"),
    videoTitle: $("videoTitle"),
    videoUploader: $("videoUploader"),
    from: $("from"),
    to: $("to"),
    setStart: $("setStart"),
    setEnd: $("setEnd"),
    timeError: $("timeError"),
    clipLength: $("clipLength"),
    previewButton: $("previewButton"),
    previewLabel: $("previewLabel"),
    outputSection: $("outputSection"),
    presetOverride: $("presetOverride"),
    presetGroup: $("presetGroup"),
    preciseRow: $("preciseRow"),
    preciseCut: $("preciseCut"),
    advanced: $("advanced"),
    advancedLoading: $("advancedLoading"),
    advancedError: $("advancedError"),
    advancedErrorText: $("advancedErrorText"),
    advancedRetry: $("advancedRetry"),
    advancedContent: $("advancedContent"),
    streamGroup: $("streamGroup"),
    streamsAV: $("streamsAV"),
    streamsA: $("streamsA"),
    videoGroup: $("videoGroup"),
    audioGroup: $("audioGroup"),
    videoFormats: $("videoFormats"),
    audioFormats: $("audioFormats"),
    audioNote: $("audioNote"),
    formatSummary: $("formatSummary"),
    formatSummaryText: $("formatSummaryText"),
    clipButton: $("clipButton"),
    clipButtonLabel: $("clipButtonLabel"),
    job: $("job"),
    jobTitle: $("jobTitle"),
    jobDetail: $("jobDetail"),
    jobProgress: $("jobProgress"),
    jobAnnounce: $("jobAnnounce"),
    jobKeepWaiting: $("jobKeepWaiting"),
    recent: $("recent"),
    recentCount: $("recentCount"),
    recentList: $("recentList"),
    recentClear: $("recentClear"),
    downloadButton: $("downloadButton"),
    trimmerHint: $("trimmerHint"),
};

const state = {
    videoId: null,
    duration: 0,
    durationSource: null, // "player" | "info"
    start: 0,
    end: DEFAULT_LENGTH,
    rangeTouched: false,
    linkStart: 0, // t= of the opened link, in seconds (0: none)
    info: null,
    infoCache: new Map(), // videoId → Promise<info>
    formats: { video: [], audio: [] },
    advancedFor: null, // videoId the advanced lists were built for
    previewing: false,
    previewArmed: false, // set once the player reports playing inside the range
    previewTimer: 0,
    infoError: null,
    playerUnavailable: false,
    playerState: "unstarted",
    jobToken: 0,
    stalledPoll: null,
    busy: false,
};

const canonicalUrl = (id) => `https://www.youtube.com/watch?v=${id}`;
const withHours = () => state.duration >= 3600;

// ===== Player =====
const player = new Player(el.player, {
    onReady() {
        el.playerFrame.dataset.player = "ready";
        setPlayerControls(true);
    },
    onDuration(seconds) {
        setDuration(seconds, "player");
    },
    onStateChange(name) {
        state.playerState = name;
        if (!state.previewing) return;
        if (!state.previewArmed) {
            // Ignore the paused/ended echoes of the seek before playback starts.
            if (name === "playing") armPreview();
            return;
        }
        if (name === "paused" || name === "ended") stopPreview(false);
    },
    onUnavailable(message) {
        state.playerUnavailable = true;
        el.playerFrame.dataset.player = "unavailable";
        el.playerSpinner.hidden = true;
        el.playerStatus.textContent = message;
        setPlayerControls(false);
        stopPreview(false);
        trimmer.setPlayhead(null);
        updateTrimmerHint();
    },
});

function setPlayerControls(enabled) {
    el.setStart.disabled = !enabled;
    el.setEnd.disabled = !enabled;
    el.previewButton.disabled = !enabled;
}

// Playhead and preview end detection.
setInterval(() => {
    if (!player.ready || el.editor.hidden) return;
    const t = player.currentTime();
    trimmer.setPlayhead(t);
    if (!state.previewing) return;
    if (!state.previewArmed) {
        if (state.playerState === "playing") armPreview();
        return;
    }
    const left = state.end - t;
    if (left <= 0.05) {
        finishPreview();
    } else if (left < 0.3 && !state.previewTimer) {
        // Stop close to the end instead of up to one tick late.
        state.previewTimer = setTimeout(finishPreview, left * 1000);
    }
}, 100);

let lastSeek = 0;
function seekPlayer(seconds, final) {
    if (!player.ready) return;
    const now = performance.now();
    if (!final && now - lastSeek < 150) return;
    lastSeek = now;
    // While previewing, a plain seek; otherwise just show the frame there.
    if (state.previewing) player.seek(seconds, final);
    else player.scrub(seconds, final);
}

// ===== Trimmer =====
const trimmer = new Trimmer(
    {
        root: $("trimmer"),
        track: $("trimTrack"),
        selection: $("trimSelection"),
        playhead: $("trimPlayhead"),
        startHandle: $("handleStart"),
        endHandle: $("handleEnd"),
        durationLabel: $("trimDuration"),
    },
    {
        onChange(start, end) {
            stopPreview(true);
            state.start = start;
            state.end = end;
            state.rangeTouched = true;
            clearTimeError();
            writeFields();
            rangeChanged();
        },
        onSeek: seekPlayer,
    },
);

// ===== Range and time fields =====
function writeFields() {
    el.from.value = formatTime(state.start, withHours());
    el.to.value = formatTime(state.end, withHours());
    el.from.removeAttribute("aria-invalid");
    el.to.removeAttribute("aria-invalid");
}

function rangeChanged() {
    const length = state.end - state.start;
    el.clipLength.textContent = length > 0 ? formatTime(length) : "–";
    refreshFormatSizes();
}

// readFields validates the typed times; the fields are what gets clipped.
function readFields() {
    const start = parseTime(el.from.value);
    const end = parseTime(el.to.value);
    if (start === null) return { error: `The start time isn't valid. ${TIME_HELP}`, field: el.from };
    if (end === null) return { error: `The end time isn't valid. ${TIME_HELP}`, field: el.to };
    if (end <= start) return { error: "The end must be after the start.", field: el.to };
    if (state.duration && end > state.duration) {
        return { error: `The end is past the end of the video (${formatTime(state.duration)}).`, field: el.to };
    }
    return { start, end };
}

function showTimeError(message, field) {
    el.timeError.textContent = message;
    el.timeError.hidden = false;
    field?.setAttribute("aria-invalid", "true");
}

function clearTimeError() {
    el.timeError.hidden = true;
    el.timeError.textContent = "";
    el.from.removeAttribute("aria-invalid");
    el.to.removeAttribute("aria-invalid");
}

// commitFields applies valid typed times to the slider. Returns false and
// shows the problem when they are not valid.
function commitFields({ seek = true } = {}) {
    const result = readFields();
    clearTimeError();
    if (result.error) {
        showTimeError(result.error, result.field);
        return false;
    }
    const startMoved = result.start !== state.start;
    const endMoved = result.end !== state.end;
    state.start = result.start;
    state.end = result.end;
    if (startMoved || endMoved) state.rangeTouched = true;
    trimmer.setRange(state.start, state.end);
    writeFields();
    rangeChanged();
    if (seek && !state.previewing && (startMoved || endMoved)) seekPlayer(startMoved ? state.start : state.end, true);
    return true;
}

// applyRange sets a range from code (buttons), clamped to the video.
function applyRange(start, end) {
    const limit = state.duration || Infinity;
    end = Math.min(Math.max(Math.round(end), 1), limit);
    start = Math.min(Math.max(Math.round(start), 0), end - 1);
    state.start = start;
    state.end = end;
    state.rangeTouched = true;
    clearTimeError();
    trimmer.setRange(start, end);
    writeFields();
    rangeChanged();
}

function setDuration(seconds, source) {
    const duration = Math.floor(seconds);
    if (!(duration > 0)) return;
    // The player knows best; the info duration is only a fallback.
    if (source === "info" && state.durationSource === "player") return;
    if (duration === state.duration) return;
    state.duration = duration;
    state.durationSource = source;

    // Keep valid typed times, otherwise fit the range into the video.
    const typed = readFields();
    if (!typed.error) {
        state.start = typed.start;
        state.end = typed.end;
    }
    let { start, end } = state;
    if (!state.rangeTouched) {
        // The default range: from the link's t= (if it lies inside the
        // video), DEFAULT_LENGTH long.
        start = state.linkStart < duration ? state.linkStart : 0;
        end = Math.min(start + DEFAULT_LENGTH, duration);
    }
    const adjusted = end > duration;
    end = Math.min(end, duration);
    start = Math.min(start, end - 1);
    state.start = Math.max(0, start);
    state.end = end;

    trimmer.setDuration(duration);
    trimmer.setRange(state.start, state.end);
    rangeChanged();
    if (adjusted) toast(`The range was trimmed to the video length (${formatTime(duration)}).`);
    // Don't rewrite a time the user is typing right now.
    if (document.activeElement === el.from || document.activeElement === el.to) return;
    clearTimeError();
    writeFields();
}

el.from.addEventListener("change", () => commitFields());
el.to.addEventListener("change", () => commitFields());

el.setStart.addEventListener("click", () => {
    const t = Math.round(player.currentTime());
    const length = state.end - state.start;
    const end = state.end > t ? state.end : t + length;
    applyRange(t, end);
});

el.setEnd.addEventListener("click", () => {
    const t = Math.round(player.currentTime());
    const length = state.end - state.start;
    const start = state.start < t ? state.start : t - length;
    applyRange(start, t);
});

// ===== Preview =====
function updatePreviewButton() {
    el.previewLabel.textContent = state.previewing ? "Stop preview" : "Preview selection";
    el.previewButton.setAttribute("aria-pressed", String(state.previewing));
}

function clearPreview() {
    clearTimeout(state.previewTimer);
    state.previewTimer = 0;
    state.previewing = false;
    state.previewArmed = false;
    updatePreviewButton();
}

function armPreview() {
    const t = player.currentTime();
    if (t >= state.start - 0.5 && t < state.end - PREVIEW_ARM_MARGIN) state.previewArmed = true;
}

function finishPreview() {
    if (!state.previewing) return;
    clearPreview();
    player.pause();
    player.seek(state.start, true);
}

function stopPreview(pause) {
    if (!state.previewing) return;
    clearPreview();
    if (pause) player.pause();
}

el.previewButton.addEventListener("click", () => {
    if (!player.ready) return;
    if (state.previewing) {
        stopPreview(true);
        return;
    }
    if (!commitFields({ seek: false })) return;
    state.previewing = true;
    state.previewArmed = false;
    player.seek(state.start, true);
    player.play();
    updatePreviewButton();
});

// ===== Video info =====
function infoFor(videoId) {
    let promise = state.infoCache.get(videoId);
    if (!promise) {
        promise = getVideoInfo(canonicalUrl(videoId));
        state.infoCache.set(videoId, promise);
        promise.catch(() => state.infoCache.delete(videoId));
    }
    return promise;
}

function useInfo(info) {
    state.info = info;
    state.infoError = null;
    if (info.thumbnail) el.playerFrame.style.backgroundImage = `url("${encodeURI(info.thumbnail)}")`;
    if (info.title) {
        el.videoTitle.textContent = info.title;
        el.videoUploader.textContent = info.uploader || "";
        el.videoMeta.hidden = false;
    }
    if (info.duration) setDuration(info.duration, "info");
}

// ===== URL =====
function showUrlError(message) {
    el.urlError.textContent = message || "";
    el.urlError.hidden = !message;
    if (message) el.url.setAttribute("aria-invalid", "true");
    else el.url.removeAttribute("aria-invalid");
}

function closeVideo() {
    stopPreview(false);
    state.videoId = null;
    player.destroy(); // stops audio that would keep playing in the hidden editor
    el.editor.hidden = true;
}

// updateTrimmerHint explains why the slider isn't ready yet.
function updateTrimmerHint() {
    if (!state.duration && state.infoError && !player.ready) {
        el.trimmerHint.textContent = `Couldn't get the video length (${state.infoError}). You can still type the start and end times.`;
    } else {
        el.trimmerHint.textContent = "Waiting for the video length… You can type the times below.";
    }
}

// openVideo shows the editor for videoId. linkStart (seconds, from the link's
// t=) is where the default range begins.
function openVideo(videoId, linkStart = 0) {
    stopPreview(true);
    state.videoId = videoId;
    state.duration = 0;
    state.durationSource = null;
    state.linkStart = linkStart;
    state.start = linkStart;
    state.end = linkStart + DEFAULT_LENGTH;
    state.rangeTouched = false;
    state.info = null;
    state.infoError = null;
    state.playerUnavailable = false;
    state.advancedFor = null;

    el.editor.hidden = false;
    updateTrimmerHint();
    el.playerFrame.dataset.player = "loading";
    el.playerFrame.style.backgroundImage = `url("https://i.ytimg.com/vi/${videoId}/hqdefault.jpg")`;
    el.playerSpinner.hidden = false;
    el.playerStatus.textContent = "Loading player…";
    el.videoMeta.hidden = true;
    setPlayerControls(false);
    if (!state.busy) el.job.hidden = true;

    trimmer.reset();
    trimmer.setRange(state.start, state.end);
    clearTimeError();
    writeFields();
    rangeChanged();

    player.load(videoId, linkStart);
    infoFor(videoId)
        .then((info) => {
            if (state.videoId === videoId) useInfo(info);
        })
        .catch((err) => {
            // Simple mode works without it; advanced mode shows its own error.
            if (state.videoId !== videoId) return;
            state.infoError = err.message;
            updateTrimmerHint();
        });
    if (el.advanced.open) loadAdvanced();
    else updateOutputMode();
}

function applyUrl() {
    const raw = el.url.value.trim();
    if (!raw) {
        showUrlError(null);
        closeVideo();
        return;
    }
    const videoId = extractVideoId(raw);
    if (!videoId) {
        showUrlError(INVALID_URL);
        closeVideo();
        return;
    }
    showUrlError(null);
    if (videoId !== state.videoId) openVideo(videoId, linkStartTime(raw) ?? 0);
}

let urlTimer = 0;
function scheduleUrl(delay) {
    clearTimeout(urlTimer);
    urlTimer = setTimeout(applyUrl, delay);
}
el.url.addEventListener("input", () => {
    // Don't flag a half-typed link; check once typing pauses.
    if (!el.urlError.hidden) showUrlError(null);
    scheduleUrl(400);
});
el.url.addEventListener("paste", () => scheduleUrl(0));
el.url.addEventListener("change", () => scheduleUrl(0));

// ===== Advanced =====
function selected(container) {
    return container.querySelector("input:checked")?.value ?? null;
}

function findFormat(list, id) {
    return list.find((f) => f.id === id) ?? null;
}

function chosenFormats() {
    const audioOnly = el.streamsA.checked;
    const video = audioOnly ? null : findFormat(state.formats.video, selected(el.videoFormats));
    const audioIncluded = video?.kind === "both";
    const audio = audioIncluded ? null : findFormat(state.formats.audio, selected(el.audioFormats));
    return { audioOnly, video, audio, audioIncluded };
}

// advancedFormat is the yt-dlp format the advanced panel resolves to, or null
// when simple mode (preset) applies.
function advancedFormat() {
    if (!el.advanced.open || state.advancedFor !== state.videoId) return null;
    const { audioOnly, video, audio } = chosenFormats();
    if (audioOnly) return audio ? audio.id : null;
    if (!video) return null;
    return audio ? `${video.id}+${audio.id}` : video.id;
}

function updateOutputMode() {
    const active = Boolean(advancedFormat());
    el.outputSection.dataset.advanced = active ? "on" : "off";
    el.presetOverride.hidden = !active;
    updatePreciseCut();
}

// cutsVideo tells whether the clip has a video stream, the only case where a
// precise cut changes anything (MP3 clips are re-encoded anyway).
function cutsVideo() {
    if (advancedFormat()) return !el.streamsA.checked;
    return (el.form.elements.preset.value || "video") === "video";
}

function updatePreciseCut() {
    el.preciseRow.hidden = !cutsVideo();
}

// The precise-cut switch is remembered in this browser.
const PRECISE_CUT_KEY = "ytclipper.preciseCut";
try {
    el.preciseCut.checked = localStorage.getItem(PRECISE_CUT_KEY) === "1";
} catch {
    // storage unavailable: start with the switch off
}
el.preciseCut.addEventListener("change", () => {
    try {
        localStorage.setItem(PRECISE_CUT_KEY, el.preciseCut.checked ? "1" : "0");
    } catch {
        // not remembered, that's fine
    }
});
el.presetGroup.addEventListener("change", updatePreciseCut);

function updateAdvancedState() {
    const { audioOnly, video, audioIncluded } = chosenFormats();
    el.videoGroup.hidden = audioOnly;
    el.audioNote.hidden = !audioIncluded || audioOnly;
    for (const input of el.audioFormats.querySelectorAll("input")) input.disabled = audioIncluded && !audioOnly;
    if (!audioOnly && !video && state.formats.video.length) {
        // keep a video selected in video+audio mode
        el.videoFormats.querySelector("input")?.click();
        return;
    }
    refreshFormatSizes();
    updateOutputMode();
}

function refreshFormatSizes() {
    if (el.advancedContent.hidden || !state.info) return;
    const length = Math.max(0, state.end - state.start);
    const duration = state.info.duration || state.duration;
    updateSizes(el.videoFormats, state.formats.video, length, duration);
    updateSizes(el.audioFormats, state.formats.audio, length, duration);

    const format = advancedFormat();
    el.formatSummary.textContent = format ?? "–";
    const { audioOnly, video, audio } = chosenFormats();
    const parts = [];
    if (!audioOnly && video) parts.push(`${videoTitle(video)} ${videoCodecName(video.vcodec)}`.trim());
    if (audio) parts.push(`${audioTitle(audio)} ${audioCodecName(audio.acodec)}`.trim());
    const bytes = [!audioOnly ? video : null, audio].reduce((sum, f) => sum + (f ? estimateBytes(f, length, duration) : 0), 0);
    el.formatSummaryText.textContent = parts.length ? `${parts.join(" + ")} · ${formatBytes(bytes)}` : "";
}

function preselect(container, id) {
    const input = id && container.querySelector(`input[value="${CSS.escape(id)}"]`);
    if (input) input.checked = true;
}

function renderAdvanced(info) {
    const formats = splitFormats(info.formats);
    state.formats = formats;
    if (!formats.video.length && !formats.audio.length) {
        showAdvancedError("YouTube didn't offer any downloadable formats for this video.");
        return;
    }
    renderList(el.videoFormats, formats.video, "videoFormat", { title: videoTitle, meta: videoMeta });
    renderList(el.audioFormats, formats.audio, "audioFormat", { title: audioTitle, meta: audioMeta });

    // Defaults: like the video preset (best H.264 up to 1080p), and the best
    // audio in a matching container.
    const video =
        formats.video.find((f) => videoCodecName(f.vcodec) === "H.264" && f.height <= 1080) ?? formats.video[0];
    const wantExt = video?.ext === "webm" ? "webm" : "m4a";
    const audio = formats.audio.find((f) => f.ext === wantExt) ?? formats.audio[0];
    preselect(el.videoFormats, video?.id);
    preselect(el.audioFormats, audio?.id);

    el.streamsAV.disabled = !formats.video.length;
    el.streamsA.disabled = !formats.audio.length;
    if (!formats.video.length) el.streamsA.checked = true;
    if (!formats.audio.length) el.streamsAV.checked = true;
    el.audioGroup.hidden = !formats.audio.length;

    state.advancedFor = state.videoId;
    el.advancedContent.hidden = false;
    updateAdvancedState();
}

function showAdvancedError(message) {
    el.advancedErrorText.textContent = message;
    el.advancedError.hidden = false;
    el.advancedContent.hidden = true;
}

async function loadAdvanced() {
    const videoId = state.videoId;
    el.advancedError.hidden = true;
    if (!videoId) {
        el.advancedContent.hidden = true;
        el.advancedLoading.hidden = true;
        showAdvancedError("Paste a YouTube link first.");
        updateOutputMode();
        return;
    }
    if (state.advancedFor === videoId) {
        el.advancedContent.hidden = false;
        updateOutputMode();
        return;
    }
    el.advancedContent.hidden = true;
    el.advancedLoading.hidden = false;
    updateOutputMode();
    try {
        const info = await infoFor(videoId);
        if (state.videoId !== videoId) return;
        if (!state.info) useInfo(info);
        renderAdvanced(info);
    } catch (err) {
        if (state.videoId !== videoId) return;
        showAdvancedError(`Couldn't load the formats: ${err.message}`);
    } finally {
        if (state.videoId === videoId) el.advancedLoading.hidden = true;
        updateOutputMode();
    }
}

el.advanced.addEventListener("toggle", () => {
    if (el.advanced.open) loadAdvanced();
    else updateOutputMode();
});
el.advancedRetry.addEventListener("click", () => loadAdvanced());
el.streamGroup.addEventListener("change", updateAdvancedState);
el.videoFormats.addEventListener("change", updateAdvancedState);
el.audioFormats.addEventListener("change", updateAdvancedState);

// ===== Job flow =====
function setBusy(busy) {
    state.busy = busy;
    el.clipButton.disabled = busy;
    el.clipButtonLabel.textContent = busy ? "Working…" : "Create clip";
}

// showJob updates the status panel. Only the title (and an error's message)
// goes to the live region, so screen readers hear state changes, not every
// elapsed-time or progress update. progress (0-100) shows the progress bar,
// null hides it and undefined leaves it as it is.
function showJob(status, title, detail = "", progress = null) {
    el.job.hidden = false;
    el.job.dataset.state = status;
    el.jobDetail.textContent = detail;
    if (progress !== undefined) setProgress(progress);
    el.downloadButton.hidden = status !== "completed";
    el.jobKeepWaiting.hidden = status !== "stalled";
    if (el.jobTitle.textContent !== title || status === "error") {
        el.jobTitle.textContent = title;
        el.jobAnnounce.textContent = status === "error" ? `${title}. ${detail}` : title;
    }
    if (status === "completed" || status === "error" || status === "stalled") {
        el.job.scrollIntoView({ block: "nearest", behavior: "smooth" });
    }
}

function setProgress(progress) {
    el.jobProgress.hidden = progress === null;
    if (progress === null) return;
    const value = Math.max(0, Math.min(100, Math.round(progress)));
    el.jobProgress.setAttribute("aria-valuenow", String(value));
    el.jobProgress.style.setProperty("--progress", `${value}%`);
}

// processingView is what the panel shows for a processing job: its phase as
// the title, with progress and time left once the download reports them.
function processingView(job, seconds) {
    if (job.phase === "finishing") {
        return ["Finishing up…", "Saving the file. Almost done.", 100];
    }
    if (typeof job.progress === "number") {
        const left = typeof job.etaSeconds === "number" ? ` · ${formatTimeLeft(job.etaSeconds)}` : "";
        return ["Downloading your clip…", `${job.progress} %${left}`, job.progress];
    }
    return ["Creating your clip…", `${seconds} s so far. This usually takes a few seconds to a minute.`, null];
}

function failJob(message) {
    // The panel announces the error; no toast, or it would be read twice.
    showJob("error", "The clip couldn't be created", message);
    setBusy(false);
}

function startDownload(url, fileName) {
    const link = document.createElement("a");
    link.href = url;
    link.download = fileName || "";
    link.hidden = true;
    document.body.append(link);
    link.click();
    link.remove();
}

function completeJob(job, clip) {
    const url = job.downloadUrl || `/api/v1/clip?jobId=${encodeURIComponent(job.id)}`;
    rememberClip(job, clip);
    el.downloadButton.href = url;
    el.downloadButton.download = job.fileName || "";
    showJob("completed", "Your clip is ready", job.fileName || "");
    setBusy(false);
    startDownload(url, job.fileName);
}

// pollJob polls until the job finishes. Rate limiting and brief server or
// network trouble back off and retry; after JOB_DEADLINE_MS it asks the user
// whether to keep waiting.
async function pollJob(poll) {
    if (poll.token !== state.jobToken) return;
    if (Date.now() > poll.deadline) {
        showJob("stalled", "This is taking unusually long", "The server hasn't finished the clip yet. Keep waiting, or create it again later.");
        setBusy(false);
        state.stalledPoll = poll;
        return;
    }

    let job;
    try {
        job = await getJob(poll.jobId);
    } catch (err) {
        if (poll.token !== state.jobToken) return;
        if (err.status === 404) {
            failJob("The job wasn't found. It may have expired; please try again.");
        } else if (TRANSIENT_STATUSES.has(err.status)) {
            poll.failures++;
            const delay = Math.min(POLL_MAX_BACKOFF_MS, POLL_MS * 2 ** poll.failures);
            showJob(el.job.dataset.state === "processing" ? "processing" : "queued", el.jobTitle.textContent || "Waiting for a free worker…", `${err.message} Retrying…`, undefined);
            setTimeout(() => pollJob(poll), delay);
        } else {
            failJob(err.message);
        }
        return;
    }
    if (poll.token !== state.jobToken) return;
    poll.failures = 0;

    const seconds = Math.round((Date.now() - poll.startedAt) / 1000);
    switch (job.status) {
        case "completed":
            completeJob(job, poll.clip);
            return;
        case "error":
            failJob(job.error || "Something went wrong while creating the clip.");
            return;
        case "processing":
            showJob("processing", ...processingView(job, seconds));
            break;
        default:
            showJob("queued", "Waiting for a free worker…", "Your clip starts as soon as one is free.");
    }
    setTimeout(() => pollJob(poll), POLL_MS);
}

el.jobKeepWaiting.addEventListener("click", () => {
    const poll = state.stalledPoll;
    if (!poll || poll.token !== state.jobToken) return;
    state.stalledPoll = null;
    poll.deadline = Date.now() + JOB_DEADLINE_MS;
    setBusy(true);
    showJob("processing", "Creating your clip…", "Still waiting for the server…", undefined);
    pollJob(poll);
});

async function startJob(payload, clip) {
    const token = ++state.jobToken;
    state.stalledPoll = null;
    setBusy(true);
    showJob("queued", "Starting…");
    let jobId;
    try {
        jobId = await createClip(payload);
    } catch (err) {
        if (token === state.jobToken) failJob(err.message);
        return;
    }
    showJob("queued", "Waiting for a free worker…", "Your clip starts as soon as one is free.");
    const now = Date.now();
    pollJob({ jobId, token, clip, startedAt: now, deadline: now + JOB_DEADLINE_MS, failures: 0 });
}

el.form.addEventListener("submit", (event) => {
    event.preventDefault();
    if (state.busy) return;

    const raw = el.url.value.trim();
    const videoId = extractVideoId(raw);
    if (!videoId) {
        showUrlError(raw ? INVALID_URL : "Paste a YouTube link first.");
        el.url.focus();
        return;
    }
    if (videoId !== state.videoId) {
        // Enter in the URL field: show the editor first.
        clearTimeout(urlTimer);
        showUrlError(null);
        openVideo(videoId, linkStartTime(raw) ?? 0);
        return;
    }
    if (!commitFields({ seek: false })) {
        el.timeError.scrollIntoView({ block: "center", behavior: "smooth" });
        return;
    }
    stopPreview(true);

    const payload = { url: canonicalUrl(videoId), from: toHHMMSS(state.start), to: toHHMMSS(state.end) };
    const format = advancedFormat();
    if (format) payload.format = format;
    else payload.preset = el.form.elements.preset.value || "video";
    if (el.preciseCut.checked && cutsVideo()) payload.preciseCut = true;
    startJob(payload, { videoId, from: state.start, to: state.end });
});

// ===== Recent clips =====
// Downloadable state per job id, checked once per page load when the list is
// opened: "available", "expired" or undefined (not checked yet).
const recentState = new Map();

function rememberClip(job, clip) {
    if (!clip) return;
    const title = (state.videoId === clip.videoId && state.info?.title) || "";
    const ext = (job.fileName || "").split(".").pop();
    addRecent({
        jobId: job.id,
        videoId: clip.videoId,
        title,
        from: clip.from,
        to: clip.to,
        kind: ext && ext !== job.fileName ? ext.toUpperCase() : "",
        fileName: job.fileName || "",
        at: Date.now(),
    });
    recentState.set(job.id, "available");
    renderRecent();
}

function recentButton(label, icon) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "button button-quiet button-sm";
    button.innerHTML = icon;
    button.append(label);
    return button;
}

const ICON_DOWNLOAD = '<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3v12M7 10l5 5 5-5M5 21h14"/></svg>';
const ICON_AGAIN = '<svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 12a9 9 0 1 0 3-6.7L3 8"/><path d="M3 3v5h5"/></svg>';

function renderRecent() {
    const list = loadRecent();
    el.recent.hidden = list.length === 0;
    el.recentCount.textContent = String(list.length);
    el.recentList.replaceChildren(
        ...list.map((entry) => {
            const item = document.createElement("li");
            item.className = "recent-item";
            item.dataset.jobId = entry.jobId;

            const text = document.createElement("div");
            text.className = "recent-text";
            const title = document.createElement("p");
            title.className = "recent-title";
            title.textContent = entry.title || `Video ${entry.videoId}`;
            title.title = title.textContent;
            const meta = document.createElement("p");
            meta.className = "recent-meta";
            const withHours = entry.to >= 3600;
            const parts = [`${formatTime(entry.from, withHours)} – ${formatTime(entry.to, withHours)}`];
            if (entry.kind) parts.push(entry.kind);
            parts.push(formatAgo(entry.at));
            meta.textContent = parts.join(" · ");
            if (recentState.get(entry.jobId) === "expired") {
                const gone = document.createElement("span");
                gone.className = "recent-expired";
                gone.textContent = " · deleted from the server";
                meta.append(gone);
            }
            text.append(title, meta);

            const actions = document.createElement("div");
            actions.className = "recent-actions";
            if (recentState.get(entry.jobId) === "available") {
                const download = document.createElement("a");
                download.className = "button button-secondary button-sm recent-download";
                download.href = `/api/v1/clip?jobId=${encodeURIComponent(entry.jobId)}`;
                download.download = entry.fileName || "";
                download.innerHTML = ICON_DOWNLOAD;
                download.append("Download");
                actions.append(download);
            }
            const again = recentButton("Clip again", ICON_AGAIN);
            again.classList.add("recent-again");
            again.title = "Open this video with the same range";
            again.addEventListener("click", () => clipAgain(entry));
            actions.append(again);

            item.append(text, actions);
            return item;
        }),
    );
}

// checkRecent asks the server which clips it still has. It only runs when the
// list is opened, never on the main flow.
async function checkRecent() {
    const unknown = loadRecent().filter((entry) => !recentState.has(entry.jobId));
    if (!unknown.length) return;
    await Promise.all(
        unknown.map(async (entry) => {
            try {
                const job = await getJob(entry.jobId);
                recentState.set(entry.jobId, job.status === "completed" ? "available" : "expired");
            } catch (err) {
                // 404: the job (and its file) is gone. Anything else: try again next time.
                if (err.status === 404) recentState.set(entry.jobId, "expired");
            }
        }),
    );
    renderRecent();
}

function clipAgain(entry) {
    el.url.value = canonicalUrl(entry.videoId);
    showUrlError(null);
    if (entry.videoId !== state.videoId) openVideo(entry.videoId, entry.from);
    applyRange(entry.from, entry.to);
    el.recent.open = false;
    el.form.scrollIntoView({ block: "start", behavior: "smooth" });
}

el.recent.addEventListener("toggle", () => {
    if (el.recent.open) checkRecent();
});
el.recentClear.addEventListener("click", () => {
    clearRecent();
    recentState.clear();
    renderRecent();
});
renderRecent();

// ===== Start =====
initTheme($("themeToggle"));

// Opened through a domain swap (/watch?v=…, /shorts/…, /<id>), /?url=… or a
// link shared to the installed app: load that video, then show the plain
// address so a reload starts fresh.
const linked = linkFromLocation(window.location);
if (linked) {
    el.url.value = linked;
    history.replaceState(null, "", "/");
}
if (el.url.value) applyUrl();

// Installable app with a share target (manifest.webmanifest). Not needed
// for the page itself, so failures are ignored.
if ("serviceWorker" in navigator) {
    navigator.serviceWorker.register("/sw.js").catch(() => {});
}
