// Client-side mirror of the server's URL check (api/validations.go). The
// server validates again; this only decides whether to load the player.

const VIDEO_ID = /^[A-Za-z0-9_-]{11}$/;
const HOSTS = new Set(["youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com"]);

// extractVideoId returns the 11-character video ID of a watch, youtu.be,
// Shorts or live link, or null.
export function extractVideoId(raw) {
    let url;
    try {
        url = new URL(String(raw ?? "").trim());
    } catch {
        return null;
    }
    if (url.protocol !== "http:" && url.protocol !== "https:") return null;

    const host = url.hostname.toLowerCase();
    const parts = url.pathname.replace(/^\/+|\/+$/g, "").split("/");
    let id = "";
    if (host === "youtu.be") {
        id = parts.length === 1 ? parts[0] : "";
    } else if (HOSTS.has(host)) {
        if (parts.length === 1 && parts[0] === "watch") {
            id = url.searchParams.get("v") ?? "";
        } else if (parts.length === 2 && (parts[0] === "shorts" || parts[0] === "live")) {
            id = parts[1];
        }
    }
    return VIDEO_ID.test(id) ? id : null;
}
