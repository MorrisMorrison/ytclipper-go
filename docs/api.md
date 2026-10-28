# API
The page uses a small JSON API, which you can also call directly. The [`http/`](../http/) folder has ready-made requests for REST clients. Errors are JSON: `{"error": "..."}`.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/video/info?youtubeUrl=…` | Title, uploader, duration (s), thumbnail and the format list |
| `POST` | `/api/v1/clip` | Start a clip job; returns the job id as plain text (201) |
| `GET` | `/api/v1/jobs/{id}` | Job state as JSON; `fileName` and `downloadUrl` once completed |
| `GET` | `/api/v1/clip?jobId=…` | Download the finished clip as `<title> [h-mm-ss - h-mm-ss].<ext>` |
| `GET` | `/api/v1/jobs/status?jobId=…` | Legacy: job state as status code (201 running, 200 done, 500 failed) |
| `GET` | `/health` | Health check (never behind auth): `Server is running (v0.0.1)`, with the running version (`dev-<sha>` for main builds) |

## Page routes
Besides `/`, the page is served for YouTube paths, so swapping the host of a YouTube link opens the editor with that video. The page reads the video from its own address and then resets the address bar to `/`.

| Path | Same as |
|---|---|
| `/watch?v=ID` | `https://www.youtube.com/watch?v=ID` (other parameters are kept; `t=` sets the clip start) |
| `/shorts/ID`, `/live/ID` | `https://www.youtube.com/shorts/ID`, `…/live/ID` |
| `/ID` | `https://youtu.be/ID` |
| `/?url=<YouTube link>` | that link |

Only valid 11-character video IDs match; other paths return `404`. The API, `/static/…` and `/health` routes take precedence.

Video info comes from one `yt-dlp -J` call and is cached in memory per video for 10 minutes, so the formats, duration and clip endpoints don't ask YouTube again. Each format has `id`, `ext`, `kind` (`video` = video only, `audio` = audio only, `both`), `width`, `height`, `fps`, `vcodec`, `acodec`, `tbr`, `abr` (kbit/s), `filesize` (bytes for the whole video, approximate if that's all YouTube reports, `0` if unknown) and `note`.

A clip request names either a `preset` or a `format`:

| Field | Value | Result |
|---|---|---|
| `preset` | `video` | Best H.264/AAC up to 1080p, merged to MP4 |
| `preset` | `audio` | MP3 at ~192 kbit/s |
| `format` | `137` or `137+140` | That yt-dlp format; a video-only format gets the best audio merged in (M4A for MP4 video) |

`from` and `to` are `H:MM:SS` (1 to 3 hour digits, e.g. `0:01:30` or `00:01:30`).

| Response | When |
|---|---|
| `201` + job id | Job queued |
| `400` `{"error": "..."}` | Invalid input, e.g. `Invalid time format. Use HH:MM:SS.`, `End time must be after start time.`, `Clip is too long (max 30 minutes).` |
| `429` `{"error": "The server is busy, try again in a minute."}` | The job queue is full (`YTCLIPPER_MAX_QUEUED_JOBS`) |
| `429` `{"message": "rate limit exceeded"}` | The per-IP rate limit was hit (any endpoint) |

A job can fail later (`status: "error"`) with, among yt-dlp's own errors: `The end time is after the end of the video (3:32).`, `The clip is larger than the 300 MB limit. Choose a shorter range or a smaller format.`, `Timed out while downloading the clip` or `The server was too busy to start the clip. Try again later.`

```bash
curl -X POST localhost:8080/api/v1/clip   -H 'Content-Type: application/json'   -d '{"url":"https://www.youtube.com/watch?v=dQw4w9WgXcQ","from":"00:00:10","to":"00:00:20","preset":"video"}'
# -> 3f1c...  (job id)
curl localhost:8080/api/v1/jobs/3f1c...
# -> {"id":"3f1c...","status":"completed","error":"","fileName":"Video title [0-00-10 - 0-00-20].mp4","downloadUrl":"/api/v1/clip?jobId=3f1c..."}
```

---
[Back to the README](../README.md)
