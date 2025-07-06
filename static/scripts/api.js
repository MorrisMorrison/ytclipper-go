import { hideProgressBar, enableClipButton, showDownloadLink } from './ui.js';

// Global state for user consent
let allowYouTubeCookies = false;

// Check if user has made a consent decision
function hasUserConsentDecision() {
    return localStorage.getItem('ytclipper-cookie-consent') !== null;
}

// Get user consent preference
function getUserConsentPreference() {
    return localStorage.getItem('ytclipper-cookie-consent') === 'true';
}

// Set user consent preference
function setUserConsentPreference(consent) {
    localStorage.setItem('ytclipper-cookie-consent', consent ? 'true' : 'false');
    allowYouTubeCookies = consent;
}

// Function to get YouTube cookies for the current domain
function getYouTubeCookies() {
    if (!allowYouTubeCookies) {
        return '';
    }
    
    try {
        const cookies = document.cookie
            .split(';')
            .filter(cookie => {
                const name = cookie.trim().split('=')[0];
                // Include important YouTube cookies
                return ['VISITOR_INFO1_LIVE', 'YSC', 'PREF', '__Secure-3PAPISID', '__Secure-3PSID', 'LOGIN_INFO'].includes(name);
            })
            .join('; ');
        return cookies;
    } catch (e) {
        console.log('Could not access YouTube cookies:', e);
        return '';
    }
}

// Show cookie consent banner if needed
function showCookieConsentBanner() {
    if (hasUserConsentDecision()) {
        allowYouTubeCookies = getUserConsentPreference();
        return;
    }
    
    const banner = document.getElementById('cookieConsentBanner');
    if (banner) {
        banner.classList.remove('hidden');
        
        // Setup event listeners
        document.getElementById('acceptCookies').addEventListener('click', () => {
            setUserConsentPreference(true);
            banner.classList.add('hidden');
            toastr.success('YouTube session sharing enabled for better success rates', 'Enhanced Mode Enabled');
        });
        
        document.getElementById('declineCookies').addEventListener('click', () => {
            setUserConsentPreference(false);
            banner.classList.add('hidden');
            toastr.info('Using standard approach without cookie sharing', 'Standard Mode');
        });
    }
}

// Initialize consent on page load
document.addEventListener('DOMContentLoaded', showCookieConsentBanner);

// Function to create request options with YouTube cookies
function createRequestOptions(options = {}) {
    const ytCookies = getYouTubeCookies();
    const headers = {
        ...options.headers,
    };
    
    // Add YouTube cookies if available
    if (ytCookies) {
        headers['X-YouTube-Cookies'] = ytCookies;
    }
    
    return {
        ...options,
        headers
    };
}

export async function fetchAndPopulateFormats(url, dropdown) {
    dropdown.disabled = true;
    dropdown.innerHTML = '<option value="">Loading formats...</option>';

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
