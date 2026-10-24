// Advanced mode: turns /api/v1/video/info formats into readable radio lists.

const VIDEO_CODECS = [
    [/^(avc|h264)/, "H.264"],
    [/^(hev|hvc|h265)/, "H.265"],
    [/^(vp0?9)/, "VP9"],
    [/^vp0?8/, "VP8"],
    [/^av01/, "AV1"],
];

const AUDIO_CODECS = [
    [/^mp4a/, "AAC"],
    [/^opus/, "Opus"],
    [/^vorbis/, "Vorbis"],
    [/^mp3/, "MP3"],
    [/^ac-3/, "AC-3"],
    [/^ec-3/, "E-AC-3"],
    [/^flac/, "FLAC"],
];

function family(codec, table) {
    const value = String(codec || "").toLowerCase();
    if (!value || value === "none") return "";
    for (const [pattern, name] of table) {
        if (pattern.test(value)) return name;
    }
    return value.split(".")[0].toUpperCase();
}

export const videoCodecName = (codec) => family(codec, VIDEO_CODECS);
export const audioCodecName = (codec) => family(codec, AUDIO_CODECS);

// splitFormats separates the lists the UI shows. "both" formats (video with
// audio inside) appear in the video list.
export function splitFormats(formats = []) {
    return {
        video: formats.filter((f) => f.kind === "video" || f.kind === "both"),
        audio: formats.filter((f) => f.kind === "audio"),
    };
}

// estimateBytes scales the full-video size to the clip length; falls back to
// the bitrate when YouTube gives no size.
export function estimateBytes(format, clipSeconds, videoDuration) {
    if (!(clipSeconds > 0)) return 0;
    if (format.filesize > 0 && videoDuration > 0) {
        return format.filesize * Math.min(1, clipSeconds / videoDuration);
    }
    const kbps = format.tbr || format.abr || 0;
    return kbps > 0 ? (kbps * 1000 * clipSeconds) / 8 : 0;
}

export function formatBytes(bytes) {
    if (!(bytes > 0)) return "size n/a";
    const mb = bytes / (1024 * 1024);
    if (mb >= 1024) return `~${(mb / 1024).toFixed(1)} GB`;
    if (mb >= 10) return `~${Math.round(mb)} MB`;
    if (mb >= 0.1) return `~${mb.toFixed(1)} MB`;
    return `~${Math.max(1, Math.round(bytes / 1024))} KB`;
}

export function videoTitle(f) {
    const res = f.height ? `${f.height}p` : f.note || `${f.width || "?"}×${f.height || "?"}`;
    return f.fps > 30 ? `${res}${Math.round(f.fps)}` : res;
}

export function videoMeta(f) {
    return [videoCodecName(f.vcodec), f.fps ? `${Math.round(f.fps)} fps` : "", String(f.ext || "").toUpperCase()]
        .filter(Boolean)
        .join(" · ");
}

export function audioTitle(f) {
    const kbps = f.abr || f.tbr;
    return kbps ? `${Math.round(kbps)} kbps` : f.note || `Format ${f.id}`;
}

export function audioMeta(f) {
    return [audioCodecName(f.acodec), String(f.ext || "").toUpperCase()].filter(Boolean).join(" · ");
}

// renderList fills a radiogroup with one radio card per format.
export function renderList(container, formats, name, describe) {
    container.replaceChildren(
        ...formats.map((f) => {
            const label = document.createElement("label");
            label.className = "fmt";

            const input = document.createElement("input");
            input.type = "radio";
            input.name = name;
            input.value = f.id;

            const body = document.createElement("span");
            body.className = "fmt-body";

            const title = document.createElement("span");
            title.className = "fmt-title";
            title.textContent = describe.title(f);
            if (f.kind === "both") {
                const badge = document.createElement("span");
                badge.className = "fmt-badge";
                badge.textContent = "with audio";
                title.append(badge);
            }

            const meta = document.createElement("span");
            meta.className = "fmt-meta";
            meta.textContent = describe.meta(f);

            const size = document.createElement("span");
            size.className = "fmt-size";
            size.dataset.formatId = f.id;

            body.append(title, size, meta);
            label.append(input, body);
            return label;
        }),
    );
}

// updateSizes refreshes the size estimates in place (keeps focus and selection).
export function updateSizes(container, formats, clipSeconds, videoDuration) {
    const byId = new Map(formats.map((f) => [f.id, f]));
    for (const el of container.querySelectorAll(".fmt-size")) {
        const f = byId.get(el.dataset.formatId);
        if (f) el.textContent = formatBytes(estimateBytes(f, clipSeconds, videoDuration));
    }
}
