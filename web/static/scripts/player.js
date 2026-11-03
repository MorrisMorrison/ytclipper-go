// Wrapper around the YouTube IFrame Player API. The player is optional: if the
// API cannot load (blocked, offline, CI) or the video refuses embedding, the
// page keeps working with typed times, and `onUnavailable` says why.

const API_SRC = "https://www.youtube.com/iframe_api";
const API_TIMEOUT_MS = 12000;
const DURATION_TIMEOUT_MS = 15000;

let apiPromise = null;

function loadApi() {
    if (window.YT?.Player) return Promise.resolve(window.YT);
    if (apiPromise) return apiPromise;

    apiPromise = new Promise((resolve, reject) => {
        const previous = window.onYouTubeIframeAPIReady;
        window.onYouTubeIframeAPIReady = () => {
            if (typeof previous === "function") previous();
            resolve(window.YT);
        };
        const script = document.createElement("script");
        script.src = API_SRC;
        script.async = true;
        script.onerror = () => reject(new Error("blocked"));
        document.head.append(script);
        setTimeout(() => reject(new Error("timeout")), API_TIMEOUT_MS);
    });
    // Let a later video try again after a failure.
    apiPromise.catch(() => {
        apiPromise = null;
    });
    return apiPromise;
}

const ERROR_MESSAGES = {
    2: "The player didn't accept this video ID.",
    5: "The player can't play this video.",
    100: "This video isn't available (removed or private).",
    101: "The owner doesn't allow playing this video on other sites.",
    150: "The owner doesn't allow playing this video on other sites.",
};

// YT.PlayerState values → names.
const STATES = { "-1": "unstarted", 0: "ended", 1: "playing", 2: "paused", 3: "buffering", 5: "cued" };
const PLAYING = 1;

export class Player {
    // container: element the player iframe goes into (its children are replaced).
    // callbacks: onReady(), onDuration(seconds), onStateChange(name), onUnavailable(message)
    constructor(container, callbacks) {
        this.container = container;
        this.callbacks = callbacks;
        this.yt = null;
        this.ready = false;
        this.videoId = null;
        this.timers = [];
        // A video that has never played only has its thumbnail: seeking it
        // shows a black frame. hasPlayed flips on the first "playing";
        // priming is set while scrub() plays it muted to get there.
        this.hasPlayed = false;
        this.priming = null; // { wasMuted }
    }

    clearTimers() {
        this.timers.forEach((id) => clearTimeout(id)); // clears intervals too
        this.timers = [];
    }

    // destroy stops playback and removes the player (link cleared).
    destroy() {
        this.videoId = null;
        this.ready = false;
        this.hasPlayed = false;
        this.priming = null;
        this.clearTimers();
        if (this.yt) {
            try {
                this.yt.destroy();
            } catch {
                // already gone
            }
            this.yt = null;
        }
        this.container.replaceChildren();
    }

    // load shows videoId, cued at start seconds (without playing it).
    async load(videoId, start = 0) {
        // A fresh player per video: cueing into an old one briefly reports the
        // previous video's duration.
        this.destroy();
        this.videoId = videoId;
        const mount = document.createElement("div");
        this.container.replaceChildren(mount);

        let YT;
        try {
            YT = await loadApi();
        } catch {
            if (this.videoId === videoId) {
                this.callbacks.onUnavailable("The YouTube player couldn't load here. You can still type the start and end times.");
            }
            return;
        }
        if (this.videoId !== videoId) return;

        this.timers.push(setTimeout(() => {
            if (!this.ready && this.videoId === videoId) {
                this.callbacks.onUnavailable("The YouTube player is taking too long. You can still type the start and end times.");
            }
        }, API_TIMEOUT_MS));

        const yt = new YT.Player(mount, {
            host: "https://www.youtube-nocookie.com",
            videoId,
            playerVars: { playsinline: 1, rel: 0, modestbranding: 1, ...(start > 0 ? { start: Math.floor(start) } : {}) },
            events: {
                onReady: () => {
                    if (this.yt !== yt) return;
                    this.ready = true;
                    this.callbacks.onReady();
                    this.pollDuration(videoId);
                },
                onStateChange: (event) => {
                    if (this.yt !== yt) return;
                    const name = STATES[event.data] ?? "unknown";
                    if (name === "playing") {
                        this.hasPlayed = true;
                        // Primed by scrub(): the frame is there now, stop again.
                        if (this.priming) {
                            yt.pauseVideo();
                            this.endPriming();
                        }
                    }
                    this.callbacks.onStateChange(name);
                },
                onError: (event) => {
                    if (this.yt !== yt) return;
                    this.ready = false;
                    this.callbacks.onUnavailable(`${ERROR_MESSAGES[event.data] ?? "The player hit an error."} You can still type the start and end times.`);
                },
            },
        });
        this.yt = yt;
    }

    // The duration is 0 until the video's metadata is in; poll briefly.
    pollDuration(videoId) {
        const started = Date.now();
        const timer = setInterval(() => {
            if (this.videoId !== videoId || Date.now() - started > DURATION_TIMEOUT_MS) {
                clearInterval(timer);
                return;
            }
            const duration = this.yt?.getDuration?.() ?? 0;
            if (duration > 0) {
                clearInterval(timer);
                this.callbacks.onDuration(Math.floor(duration));
            }
        }, 250);
        this.timers.push(timer);
    }

    currentTime() {
        return this.ready ? this.yt.getCurrentTime?.() ?? 0 : 0;
    }

    seek(seconds, final = true) {
        if (this.ready) this.yt.seekTo(seconds, final);
    }

    // scrub shows the frame at seconds without leaving the video playing
    // (dragging a trim handle). Playback that's already running continues.
    // A video that hasn't played yet stays black after a seek, so the first
    // scrub plays it muted and pauses again on "playing".
    scrub(seconds, final = true) {
        if (!this.ready) return;
        this.yt.seekTo(seconds, final);
        if (!this.hasPlayed) {
            if (!this.priming) {
                this.priming = { wasMuted: this.yt.isMuted() };
                this.yt.mute();
                this.yt.playVideo();
            }
            return;
        }
        if (this.yt.getPlayerState() !== PLAYING) this.yt.pauseVideo();
    }

    endPriming() {
        if (!this.priming) return;
        if (!this.priming.wasMuted) this.yt.unMute();
        this.priming = null;
    }

    play() {
        if (!this.ready) return;
        this.endPriming();
        this.yt.playVideo();
    }

    pause() {
        if (!this.ready) return;
        this.endPriming();
        this.yt.pauseVideo();
    }
}
