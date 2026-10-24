// Thin wrappers around the JSON API. Every failure becomes an ApiError with a
// message that can be shown to the user as is.

export class ApiError extends Error {
    constructor(message, status = 0) {
        super(message);
        this.status = status;
    }
}

async function send(url, options) {
    try {
        return await fetch(url, options);
    } catch (err) {
        if (err?.name === "AbortError") throw err;
        throw new ApiError("Can't reach the server. Check your connection and try again.");
    }
}

async function errorFrom(response) {
    if (response.status === 429) {
        return new ApiError("Too many requests. Wait a moment and try again.", 429);
    }
    let message = "";
    try {
        const body = await response.json();
        message = body?.error || body?.message || "";
    } catch {
        // not JSON
    }
    return new ApiError(message || `Request failed (HTTP ${response.status}).`, response.status);
}

// getVideoInfo → { id, title, uploader, duration, thumbnail, formats: [...] }
export async function getVideoInfo(youtubeUrl, signal) {
    const response = await send(`/api/v1/video/info?youtubeUrl=${encodeURIComponent(youtubeUrl)}`, { signal });
    if (!response.ok) throw await errorFrom(response);
    return response.json();
}

// createClip posts { url, from, to, preset | format } and returns the job id.
export async function createClip(payload) {
    const response = await send("/api/v1/clip", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
    });
    if (!response.ok) throw await errorFrom(response);
    const jobId = (await response.text()).trim();
    if (!jobId) throw new ApiError("The server did not return a job.");
    return jobId;
}

// getJob → { id, status, error, fileName, downloadUrl }
export async function getJob(jobId) {
    const response = await send(`/api/v1/jobs/${encodeURIComponent(jobId)}`, { cache: "no-store" });
    if (!response.ok) throw await errorFrom(response);
    return response.json();
}
