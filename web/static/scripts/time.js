// Time parsing and formatting. All times are whole seconds.

const TIME_PATTERN = /^\d+(:\d{1,2}){0,2}$/;

// parseTime accepts "34", "1:28" and "1:09:24" (the leading part may exceed 59,
// so "90" is 1:30). Returns seconds, or null when the text is not a time.
export function parseTime(text) {
    const value = String(text ?? "").trim();
    if (!TIME_PATTERN.test(value)) return null;
    const parts = value.split(":").map(Number);
    for (let i = 1; i < parts.length; i++) {
        if (parts[i] > 59) return null;
    }
    return parts.reduce((total, part) => total * 60 + part, 0);
}

// formatTime renders seconds as m:ss, or h:mm:ss from one hour on (or when
// withHours is set, so values line up for long videos).
export function formatTime(seconds, withHours = false) {
    const total = Math.max(0, Math.floor(seconds));
    const h = Math.floor(total / 3600);
    const m = Math.floor((total % 3600) / 60);
    const s = total % 60;
    const ss = String(s).padStart(2, "0");
    if (h > 0 || withHours) return `${h}:${String(m).padStart(2, "0")}:${ss}`;
    return `${m}:${ss}`;
}

// toHHMMSS is the wire format the API expects.
export function toHHMMSS(seconds) {
    const total = Math.max(0, Math.floor(seconds));
    const pad = (n) => String(n).padStart(2, "0");
    return `${pad(Math.floor(total / 3600))}:${pad(Math.floor((total % 3600) / 60))}:${pad(total % 60)}`;
}
