const timeObjectToSeconds = (time) =>
  time.hours * 60 * 60 + time.minutes * 60 + time.seconds;
const isTimeInputValid = (time) => {
  return /^(\d+(:[0-5]?\d){0,2})$/.test(time);
};
const isYoutubeUrlValid = (url) =>
  /http(?:s?):\/\/(?:www\.)?youtu(?:be\.com\/watch\?v=|\.be\/)([\w\-\_]*)(&(amp;)?‌​[\w\?‌​=]*)?/.test(
    url
  );
const isTimestampWithinDuration = (timestamp, duration) =>
  timestamp <= duration;
const getTimeAsObject = (time) => {
  const parts = time.split(":").map(Number).reverse();
  const seconds = parts[0] || 0; 
  const minutes = parts[1] || 0;
  const hours = parts[2] || 0;
  return { hours, minutes, seconds };
};
const normalizeTimeToHHMMSS = (time) => {
  const { hours, minutes, seconds } = getTimeAsObject(time);
  const pad = (num) => String(num).padStart(2, "0"); 
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`;
};

const convertToSeconds = (timeString) =>
  timeObjectToSeconds(getTimeAsObject(timeString));
