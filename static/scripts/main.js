import { debounce, isYoutubeUrlValid, isTimeInputValid, normalizeTimeToHHMMSS } from './utils.js';
import { fetchAndPopulateFormats, getVideoDuration, getJobStatus } from './api.js';
import { disableDropdown, showProgressBar, hideProgressBar, enableClipButton, disableClipButton, isVideoPlayerVisible, showVideoPlayer, hideVideoPlayer } from './ui.js';

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

    toastr.success("Clip processing started.");
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
        const response = await fetch("/api/v1/clip", { method: "POST", body: JSON.stringify(payload), headers: { "Content-Type": "application/json" } });
        
         switch (response.status) {
      case 201:
        toastr.success(
          "The download will pop up automatically. This may take a few seconds.",
          "Download Started"
        );
        showProgressBar();
        const jobId = await response.text();
        getJobStatus(jobId);
        break;
      case 500:
        toastr.error("Timestamps are not within video length.");
        break;
      default:
        toastr.error("An unexpected error occurred.");
        break;
    }
        
    } catch (err) {
        toastr.error("Failed to create clip: " + err.message);
    }
};

document.getElementById("clipButton").addEventListener("click", onClipButtonClick);

const onPreviewButtonClick = () => {
  const url = document.getElementById("url").value;
  if (!isYoutubeUrlValid(url)) {
    toastr.error("Please provide a valid YouTube URL.", "Invalid Url");
    return;
  }
  if (isVideoPlayerVisible()) {
    hideVideoPlayer();
  } else {
    showVideoPlayer();
  }
};


document.getElementById("previewButton").addEventListener("click", onPreviewButtonClick);