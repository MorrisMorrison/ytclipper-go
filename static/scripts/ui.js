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

export function showVideoPlayer (){
  const player = videojs("videoPlayer", {
    techOrder: ["youtube"],
    sources: [
      {
        type: "video/youtube",
        src: document.getElementById("url").value,
      },
    ],
  });
  document.getElementById("videoPlayerWrapper").classList.remove("hidden");
};
export function hideVideoPlayer (){
  document.getElementById("videoPlayerWrapper").classList.add("hidden")}
export function  isVideoPlayerVisible() {
  !document.getElementById("videoPlayerWrapper").classList.contains("hidden")}

export function showDownloadLink(downloadUrl){
  const downloadLinkUrlWrapper = document.getElementById("downloadLinkWrapper");
  const downloadLink = document.getElementById("downloadLink");
  downloadLink.setAttribute("href", downloadUrl);
  downloadLinkUrlWrapper.classList.remove("hidden");
};

export function hideDownloadLink(){
  document.getElementById("downloadLinkWrapper").classList.add("hidden")}
export function handleDarkMode(){
  const body = document.body;
  if (
    localStorage.theme === "dark" ||
    (!("theme" in localStorage) &&
      window.matchMedia("(prefers-color-scheme: dark)").matches)
  ) {
    body.classList.add("dark");
  } else {
    body.classList.remove("dark");
  }
};
export function setTheme(){
  localStorage.theme = localStorage.theme === "dark" ? "light" : "dark";
  handleDarkMode();
};
document.getElementById("themeSlider").addEventListener("click", setTheme);


window.onload = () => {
  if (!localStorage.theme) {
    localStorage.theme = "dark";
  }
  handleDarkMode();
};