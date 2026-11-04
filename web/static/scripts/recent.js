// Recent clips, kept only in this browser (localStorage): enough to download a
// clip again while the server still has it, or to open the same video and
// range again. Nothing about them is stored on the server.

const KEY = "ytclipper.recentClips";
export const MAX_RECENT = 10;

function storage() {
    try {
        return window.localStorage;
    } catch {
        return null;
    }
}

function valid(entry) {
    return (
        entry &&
        typeof entry.jobId === "string" &&
        typeof entry.videoId === "string" &&
        Number.isFinite(entry.from) &&
        Number.isFinite(entry.to) &&
        Number.isFinite(entry.at)
    );
}

// loadRecent returns the stored clips, newest first. Broken or foreign data
// reads as an empty list.
export function loadRecent() {
    try {
        const list = JSON.parse(storage()?.getItem(KEY) || "[]");
        return Array.isArray(list) ? list.filter(valid).slice(0, MAX_RECENT) : [];
    } catch {
        return [];
    }
}

function save(list) {
    try {
        storage()?.setItem(KEY, JSON.stringify(list));
    } catch {
        // full or unavailable: the list just isn't remembered
    }
}

// addRecent stores a finished clip at the top and returns the new list.
// entry: { jobId, videoId, title, from, to (seconds), kind ("MP4"), fileName, at (ms) }
export function addRecent(entry) {
    const list = [entry, ...loadRecent().filter((e) => e.jobId !== entry.jobId)].slice(0, MAX_RECENT);
    save(list);
    return list;
}

export function clearRecent() {
    try {
        storage()?.removeItem(KEY);
    } catch {
        // nothing to clear
    }
}
