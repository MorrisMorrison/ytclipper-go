import { hideProgressBar, enableClipButton, showDownloadLink } from './ui.js';

// Function to create request options
function createRequestOptions(options = {}) {
    return {
        ...options,
        headers: {
            ...options.headers,
        }
    };
}

export async function fetchAndPopulateFormats(url, dropdown) {
    dropdown.disabled = true;
    dropdown.innerHTML = '<option value="">Loading formats...</option>';
            toastr.success(
          "This may take a few seconds.",
          "Fetching available formats."
        )
    try {
        const requestOptions = createRequestOptions();
        const response = await fetch(`/api/v1/video/formats?youtubeUrl=${encodeURIComponent(url)}`, requestOptions);
        if (!response.ok) throw new Error(await response.text());

        const formats = await response.json();
        populateDropdown(formats, dropdown);
    } catch (err) {
        dropdown.innerHTML = '<option value="">Error loading formats</option>';
        throw err;
    }
}

export async function getVideoDuration(youtubeUrl) {
    const url = `/api/v1/video/duration?youtubeUrl=${encodeURIComponent(youtubeUrl)}`;
    const requestOptions = createRequestOptions();
    const response = await fetch(url, requestOptions);
    if (response.ok) return await response.text();
    throw new Error('Failed to fetch video duration');
}

function populateDropdown(formats, dropdown) {
    dropdown.innerHTML = "";
    const groups = {
        "Audio Only": formats.filter(f => f.formatType === "audio only"),
        "Video Only": formats.filter(f => f.formatType === "video only"),
        "Audio and Video": formats.filter(f => f.formatType === "audio and video"),
    };

    for (const [label, items] of Object.entries(groups)) {
        if (items.length === 0) continue;
        const group = document.createElement("optgroup");
        group.label = label;
        items.forEach(format => {
            const option = document.createElement("option");
            option.value = format.id;
            option.textContent = `${format.label} (${format.extension}, ${format.codec || 'N/A'}, ${format.bitrate || 'N/A'})`;
            group.appendChild(option);
        });
        dropdown.appendChild(group);
    }
    dropdown.disabled = false;
}

export async function getJobStatus(jobId){
  const url = window.location.href + "api/v1/jobs/status?jobId=" + jobId;
  try {
    const res = await fetch(url, { method: "GET" });
    switch (res.status) {
      case 200:
        const result = await res.text();
        hideProgressBar();
        const downloadUrl = "/api/v1/clip?jobId=" + jobId;
        showDownloadLink(downloadUrl);
        window.open(downloadUrl);
        enableClipButton();
        break;
      case 201:
        setTimeout(() => getJobStatus(jobId), 2000);
        break;
      case 408:
        toastr.error(
          "The download timed out. Please try again in a few minutes or use the contact form.",
          "Download Timeout"
        );
        enableClipButton();
        break;
      case 500:
        toastr.error(
          "An error occurred when downloading the clip. Please try again in a few minutes or use the contact form.",
          "Download Error"
        );
        enableClipButton();
        break;
      default:
        toastr.error(
          "An error occurred when retrieving the job status. Please try again in a few minutes or use the contact form.",
          "Unknown Error"
        );
        enableClipButton();
        break;
    }
  } catch (error) {
    console.error("CLIENT - GETJOBSTATUS - An error occurred:", error);
    enableClipButton();
  }
};
