import { normalizeTimeToHHMMSS } from './utils.js';

export async function fetchAndPopulateFormats(url, dropdown) {
    dropdown.disabled = true;
    dropdown.innerHTML = '<option value="">Loading formats...</option>';

    try {
        const response = await fetch(`/api/v1/video/formats?youtubeUrl=${encodeURIComponent(url)}`);
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
    const response = await fetch(url);
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
