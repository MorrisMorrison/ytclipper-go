import { debounce, isYoutubeUrlValid, isTimeInputValid, normalizeTimeToHHMMSS } from './utils.js';
import { fetchAndPopulateFormats, getVideoDuration } from './api.js';
import { disableDropdown, enableDropdown, showProgressBar, hideProgressBar, enableClipButton, disableClipButton } from './ui.js';

const onUrlInputChange = debounce(async (event) => {
    const url = event.target.value;
    const dropdown = document.getElementById("formatSelect");

    if (!isYoutubeUrlValid(url)) {
        toastr.error("Please enter a valid YouTube URL");
        disableDropdown(dropdown);
        return;
    }

    try {
        await fetchAndPopulateFormats(url, dropdown);
    } catch (err) {
        toastr.error("Failed to fetch formats: " + err.message);
    }
}, 500);

document.getElementById("url").addEventListener("input", onUrlInputChange);

const onClipButtonClick = async () => {
    disableClipButton();
    hideProgressBar();

    const url = document.getElementById("url").value;
    const from = document.getElementById("from").value;
    const to = document.getElementById("to").value;
    const format = document.getElementById("formatSelect").value;

    if (!isYoutubeUrlValid(url) || !isTimeInputValid(from) || !isTimeInputValid(to) || !format) {
        toastr.error("Invalid input. Check the URL, timestamps, and format.");
        enableClipButton();
        return;
    }

    try {
        const videoDuration = await getVideoDuration(url);
        if (normalizeTimeToHHMMSS(from) > videoDuration || normalizeTimeToHHMMSS(to) > videoDuration) {
            toastr.error("Timestamps exceed video duration.");
            enableClipButton();
            return;
        }

        showProgressBar();
        const payload = { url, from: normalizeTimeToHHMMSS(from), to: normalizeTimeToHHMMSS(to), format };
        await fetch("/api/v1/clip", { method: "POST", body: JSON.stringify(payload), headers: { "Content-Type": "application/json" } });
        toastr.success("Clip processing started.");
    } catch (err) {
        toastr.error("Failed to create clip: " + err.message);
    } finally {
        hideProgressBar();
        enableClipButton();
    }
};

document.getElementById("clipButton").addEventListener("click", onClipButtonClick);
