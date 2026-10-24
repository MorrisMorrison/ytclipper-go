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
    } catch {
        throw new ApiError("Can't reach the server. Check your connection and try again.");
    }
}

async function errorFrom(response) {
    let body = null;
    try {
        body = await response.json();
    } catch {
        // not JSON
    }
    // Handlers send {"error"}; Echo's rate limiter sends {"message": "rate limit exceeded"}.
    let message = body?.error || "";
    if (!message && response.status === 429) message = "Too many requests. Wait a moment and try again.";
    if (!message) message = body?.message || "";
    return new ApiError(message || `Request failed (HTTP ${response.status}).`, response.status);
}

// getVideoInfo → { id, title, uploader, duration, thumbnail, formats: [...] }
export async function getVideoInfo(youtubeUrl) {
    const response = await send(`/api/v1/video/info?youtubeUrl=${encodeURIComponent(youtubeUrl)}`);
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
