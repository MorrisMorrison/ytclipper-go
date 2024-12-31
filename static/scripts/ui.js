export function disableDropdown(dropdown) {
    dropdown.disabled = true;
    dropdown.innerHTML = '<option value="">Loading formats...</option>';
}

export function enableDropdown(dropdown) {
    dropdown.disabled = false;
}

export function showProgressBar() {
    document.getElementById("progressBarWrapper").classList.remove("hidden");
}

export function hideProgressBar() {
    document.getElementById("progressBarWrapper").classList.add("hidden");
}

export function enableClipButton() {
    document.getElementById("clipButton").disabled = false;
}

export function disableClipButton() {
    document.getElementById("clipButton").disabled = true;
}
