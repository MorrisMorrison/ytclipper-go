export function debounce(func, delay) {
    let timeout;
    return function (...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), delay);
    };
}

export function isYoutubeUrlValid(url) {
    const regex = /http(?:s?):\/\/(?:www\.)?youtu(?:be\.com\/watch\?v=|\.be\/)([\w\-\_]*)(&(amp;)?‌​[\w\?‌​=]*)?/;
    return regex.test(url);
}

export function isTimeInputValid(time) {
    return /^([0-1]?\d|2[0-3]):[0-5]?\d:[0-5]?\d$/.test(time);
}

export function normalizeTimeToHHMMSS(time) {
    const parts = time.split(":").map((v) => v.padStart(2, "0"));
    while (parts.length < 3) parts.unshift("00");
    return parts.join(":");
}
