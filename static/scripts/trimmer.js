// Two-handle range slider over the video, video-editor style.
//
// - Drag a handle to move one end; drag the highlighted selection to move both.
// - Pressing on the bare track jumps the nearest handle there and keeps dragging.
// - Handles are ARIA sliders: arrows move 1 s, Shift+arrows and Page keys 10 s,
//   Home/End jump to the limits.
// All values are whole seconds; the selection is at least MIN_LENGTH long.

import { formatTime } from "./time.js";

const MIN_LENGTH = 1;

export class Trimmer {
    // els: { root, track, selection, playhead, startHandle, endHandle, durationLabel }
    // onChange(start, end, { final }) fires on every move; onSeek(seconds, final) asks the player to seek.
    constructor(els, { onChange, onSeek }) {
        this.els = els;
        this.onChange = onChange;
        this.onSeek = onSeek;
        this.duration = 0;
        this.start = 0;
        this.end = 0;
        this.drag = null;

        this.bindHandle(els.startHandle, "start");
        this.bindHandle(els.endHandle, "end");
        els.selection.addEventListener("pointerdown", (event) => this.beginDrag(event, "range"));
        els.track.addEventListener("pointerdown", (event) => this.onTrackDown(event));
        els.track.addEventListener("pointermove", (event) => this.onMove(event));
        els.track.addEventListener("pointerup", (event) => this.endDrag(event));
        els.track.addEventListener("pointercancel", (event) => this.endDrag(event));
        els.track.addEventListener("lostpointercapture", (event) => this.endDrag(event));
    }

    get ready() {
        return this.duration > 0;
    }

    setDuration(duration) {
        this.duration = Math.max(0, Math.floor(duration));
        const ready = this.duration > MIN_LENGTH;
        this.els.root.dataset.ready = String(ready);
        for (const handle of [this.els.startHandle, this.els.endHandle]) {
            handle.tabIndex = ready ? 0 : -1;
            handle.setAttribute("aria-disabled", String(!ready));
        }
        this.els.durationLabel.textContent = ready ? formatTime(this.duration) : "";
        this.render();
    }

    reset() {
        this.drag = null;
        this.setDuration(0);
        this.setPlayhead(null);
    }

    // setRange is for outside changes (typed times); it never fires onChange.
    setRange(start, end) {
        this.start = start;
        this.end = end;
        this.render();
    }

    setPlayhead(seconds) {
        const head = this.els.playhead;
        if (seconds == null || !this.ready) {
            head.hidden = true;
            return;
        }
        head.hidden = false;
        head.style.left = `${this.percent(seconds)}%`;
    }

    percent(seconds) {
        return this.duration ? Math.min(100, Math.max(0, (seconds / this.duration) * 100)) : 0;
    }

    render() {
        const { selection, startHandle, endHandle } = this.els;
        const left = this.percent(this.start);
        const right = this.percent(this.end);
        selection.style.left = `${left}%`;
        selection.style.width = `${Math.max(0, right - left)}%`;
        startHandle.style.left = `${left}%`;
        endHandle.style.left = `${right}%`;

        const withHours = this.duration >= 3600;
        this.setAria(startHandle, this.start, 0, Math.max(0, this.end - MIN_LENGTH), withHours);
        this.setAria(endHandle, this.end, Math.min(this.duration, this.start + MIN_LENGTH), this.duration, withHours);
    }

    setAria(handle, value, min, max, withHours) {
        handle.setAttribute("aria-valuemin", String(min));
        handle.setAttribute("aria-valuemax", String(max));
        handle.setAttribute("aria-valuenow", String(value));
        handle.setAttribute("aria-valuetext", formatTime(value, withHours));
    }

    // move sets one end (or shifts both for "range"), clamped, and reports it.
    move(which, seconds, final) {
        const d = this.duration;
        const t = Math.round(seconds);
        if (which === "start") {
            this.start = clamp(t, 0, this.end - MIN_LENGTH);
        } else if (which === "end") {
            this.end = clamp(t, this.start + MIN_LENGTH, d);
        } else {
            const length = this.end - this.start;
            this.start = clamp(t, 0, d - length);
            this.end = this.start + length;
        }
        this.render();
        this.onChange(this.start, this.end, { final });
        this.onSeek(which === "end" ? this.end : this.start, final);
    }

    timeAt(clientX) {
        const rect = this.els.track.getBoundingClientRect();
        const ratio = rect.width ? (clientX - rect.left) / rect.width : 0;
        return clamp(ratio, 0, 1) * this.duration;
    }

    bindHandle(handle, which) {
        handle.addEventListener("pointerdown", (event) => this.beginDrag(event, which));
        handle.addEventListener("keydown", (event) => this.onKey(event, which));
    }

    beginDrag(event, which) {
        if (!this.ready || event.button > 0) return;
        event.preventDefault();
        event.stopPropagation();
        const grabbedAt = this.timeAt(event.clientX);
        this.drag = { which, pointerId: event.pointerId, offset: which === "range" ? grabbedAt - this.start : 0 };
        this.els.track.setPointerCapture(event.pointerId);
        const handle = which === "end" ? this.els.endHandle : this.els.startHandle;
        if (which !== "range") {
            handle.dataset.dragging = "true";
            handle.focus({ preventScroll: true });
        }
    }

    onTrackDown(event) {
        if (!this.ready || event.button > 0 || this.drag) return;
        const t = this.timeAt(event.clientX);
        const which = Math.abs(t - this.start) <= Math.abs(t - this.end) ? "start" : "end";
        this.beginDrag(event, which);
        this.move(which, t, false);
    }

    onMove(event) {
        if (!this.drag || event.pointerId !== this.drag.pointerId) return;
        this.move(this.drag.which, this.timeAt(event.clientX) - this.drag.offset, false);
    }

    endDrag(event) {
        if (!this.drag || event.pointerId !== this.drag.pointerId) return;
        const { which } = this.drag;
        this.drag = null;
        delete this.els.startHandle.dataset.dragging;
        delete this.els.endHandle.dataset.dragging;
        // Report the final position once so the player does an exact seek.
        this.onChange(this.start, this.end, { final: true });
        this.onSeek(which === "end" ? this.end : this.start, true);
    }

    onKey(event, which) {
        if (!this.ready) return;
        const big = 10;
        const step = event.shiftKey ? big : 1;
        const value = which === "start" ? this.start : this.end;
        let next;
        switch (event.key) {
            case "ArrowLeft":
            case "ArrowDown":
                next = value - step;
                break;
            case "ArrowRight":
            case "ArrowUp":
                next = value + step;
                break;
            case "PageDown":
                next = value - big;
                break;
            case "PageUp":
                next = value + big;
                break;
            case "Home":
                next = 0;
                break;
            case "End":
                next = this.duration;
                break;
            default:
                return;
        }
        event.preventDefault();
        this.move(which, next, true);
    }
}

function clamp(value, min, max) {
    return Math.min(max, Math.max(min, value));
}
