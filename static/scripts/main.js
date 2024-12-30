function debounce(func, delay) {
    let timeout;
    return function (...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), delay);
    };
}
function disableDropdown() {
    const dropdown = document.getElementById("formatSelect");
    dropdown.disabled = true;
    dropdown.innerHTML = '<option value="">Loading formats...</option>';
}

function enableDropdown() {
  const dropdown = document.getElementById("formatSelect");
  dropdown.disabled = true;
  dropdown.innerHTML = '<option value="">Loading formats...</option>';
}

const onUrlInputChange = debounce(async (event) => {
    const url = event.target.value;

    if (!isYoutubeUrlValid(url)) {
        toastr.error("Please enter a valid YouTube URL");
        disableDropdown();
        return;
    }

    try {
        await fetchAndPopulateFormats(url);
    } catch (err) {
        toastr.error("Failed to fetch formats: " + err.message);
    }
}, 500);

async function fetchAndPopulateFormats(url) {
  const dropdown = document.getElementById("formatSelect");
  disableDropdown();

  try {
      const response = await fetch(`/api/v1/video/formats?youtubeUrl=${encodeURIComponent(url)}`);
      if (!response.ok) {
          throw new Error(await response.text());
      }

      const formats = await response.json();
      dropdown.innerHTML = ""; // Clear existing options

      // Separate formats into groups
      const audioFormats = formats.filter(format => format.formatType === "audio only");
      const videoFormats = formats.filter(format => format.formatType === "video only");
      const audioVideoFormats = formats.filter(format => format.formatType === "audio and video");

      // Function to create an optgroup with options
      function createOptGroup(label, formats) {
          const group = document.createElement("optgroup");
          group.label = label;

          formats.forEach(format => {
              const option = document.createElement("option");
              option.value = format.id;
              const typeInfo = ` (${format.formatType})`;
              option.textContent = `${format.label} (${format.extension}, ${format.codec}, ${format.bitrate || 'N/A'})${typeInfo}`;
              group.appendChild(option);
          });

          return group;
      }

      // Append groups only if they have items
      if (audioFormats.length > 0) {
          dropdown.appendChild(createOptGroup("Audio Only", audioFormats));
      }
      if (videoFormats.length > 0) {
          dropdown.appendChild(createOptGroup("Video Only", videoFormats));
      }
      if (audioVideoFormats.length > 0) {
          dropdown.appendChild(createOptGroup("Audio and Video", audioVideoFormats));
      }

      dropdown.disabled = false;
  } catch (err) {
      dropdown.innerHTML = '<option value="">Error loading formats</option>';
      dropdown.disabled = true;
      throw err;
  }
}



const getVideoDuration = async (youtubeUrl) => {
  const url =
    window.location.href + "api/v1/video/duration?youtubeUrl=" + youtubeUrl;
  try {
    const res = await fetch(url, { method: "GET" });
    if (res.status === 200) {
      const durationInSeconds = await res.text();
      return durationInSeconds;
    }
  } catch (error) {
    console.error(
      "CLIENT - GETJOBSTATUS - Error fetching video duration:",
      error
    );
    enableClipButton();
  }
};

const onClipButtonClick = async () => {
  disableClipButton();
  hideDownloadLink();

  const url = window.location.href + "api/v1/clip";
  const youtubeUrl = getUrlInput();
  const from = document.getElementById("from").value;
  const to = document.getElementById("to").value;
  const format = document.getElementById("formatSelect").value; // Get selected format

  if (url === "" || from === "" || to === "" || format === "") {
    toastr.error("Please provide a URL, both timestamps, and a format.", "Invalid Input");
    enableClipButton();
    return;
  }

  if (!isTimeInputValid(from) || !isTimeInputValid(to)) {
    toastr.error("Please provide timestamps as HH:MM:SS.", "Invalid Format");
    enableClipButton();
    return;
  }

  if (!isYoutubeUrlValid(youtubeUrl)) {
    toastr.error("Please provide a valid YouTube URL.", "Invalid Url");
    enableClipButton();
    return;
  }

  try {
    const videoDuration = await getVideoDuration(youtubeUrl);
    const fromInSeconds = convertToSeconds(from);
    const toInSeconds = convertToSeconds(to);
    const durationInSeconds = convertToSeconds(videoDuration);

    if (
      !isTimestampWithinDuration(fromInSeconds, durationInSeconds) ||
      !isTimestampWithinDuration(toInSeconds, durationInSeconds)
    ) {
      toastr.error(
        "Please use timestamps that are within the video's duration.",
        "Invalid Timestamps"
      );

      enableClipButton();
      return;
    }

    const payload = JSON.stringify({
      url: youtubeUrl,
      from: from,
      to: to,
      format: format, // Include selected format in the payload
    });

    const headers = {
      "content-type": "application/json",
    };

    const response = await fetch(url, {
      method: "POST",
      headers: headers,
      body: payload,
    });

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
    }
  } catch (error) {
    console.error("CLIENT - GETJOBSTATUS - An error occurred:", error);
    enableClipButton();
  }
};


const getJobStatus = async (jobId) => {
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

const onPreviewButtonClick = () => {
  const url = getUrlInput();
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

const getUrlInput = () => document.getElementById("url").value;

const showProgressBar = () =>
  document.getElementById("progressBarWrapper").classList.remove("hidden");
const hideProgressBar = () =>
  document.getElementById("progressBarWrapper").classList.add("hidden");

const enableClipButton = () =>
  (document.getElementById("clipButton").disabled = false);
const disableClipButton = () =>
  (document.getElementById("clipButton").disabled = true);

const showVideoPlayer = () => {
  const player = videojs("video-player", {
    techOrder: ["youtube"],
    sources: [
      {
        type: "video/youtube",
        src: getUrlInput(),
      },
    ],
  });

  document.getElementById("video-player-wrapper").classList.remove("hidden");
};
const hideVideoPlayer = () =>
  document.getElementById("video-player").classList.add("hidden");
const isVideoPlayerVisible = () =>
  !document.getElementById("video-player").classList.contains("hidden");

const showDownloadLink = (downloadUrl) => {
  const downloadLinkUrlWrapper = document.getElementById("downloadLinkWrapper");
  const downloadLink = document.getElementById("downloadLink");
  downloadLink.setAttribute("href", downloadUrl);
  downloadLinkUrlWrapper.classList.remove("hidden");
};
const hideDownloadLink = () =>
  document.getElementById("downloadLinkWrapper").classList.add("hidden");

const handleDarkMode = () => {
  if (
    localStorage.theme === "dark" ||
    (!("theme" in localStorage) &&
      window.matchMedia("(prefers-color-scheme: dark)").matches)
  ) {
    document.documentElement.classList.add("dark");
  } else {
    document.documentElement.classList.remove("dark");
  }
};

const setTheme = () => {
  localStorage.theme = localStorage.theme === "dark" ? "light" : "dark";
  handleDarkMode();
};

window.onload = () => {
  localStorage.theme = "dark";
  handleDarkMode();
};
