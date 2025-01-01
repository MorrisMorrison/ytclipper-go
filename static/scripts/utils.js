export function debounce(func, delay) {
    let timeout;
    return function (...args) {
        clearTimeout(timeout);
        timeout = setTimeout(() => func.apply(this, args), delay);
    };
}

export function timeObjectToSeconds(time){
  time.hours * 60 * 60 + time.minutes * 60 + time.seconds}

export function isTimeInputValid(time){
  return /^(\d+(:[0-5]?\d){0,2})$/.test(time);
};
export function isYoutubeUrlValid(url) {
    const regex = /^(https?:\/\/)?(www\.)?(youtube\.com\/watch\?v=|youtu\.be\/)[a-zA-Z0-9_-]+$/;
    return regex.test(url);
}

export function isTimestampWithinDuration(timestamp, duration){
  timestamp <= duration}

export function getTimeAsObject(time){
  const parts = time.split(":").map(Number).reverse();
  const seconds = parts[0] || 0; 
  const minutes = parts[1] || 0;
  const hours = parts[2] || 0;
  return { hours, minutes, seconds };
}

export function normalizeTimeToHHMMSS(time){
  const { hours, minutes, seconds } = getTimeAsObject(time);
  const pad = (num) => String(num).padStart(2, "0"); 
  return `${pad(hours)}:${pad(minutes)}:${pad(seconds)}`
}

export function convertToSeconds(timeString){
  timeObjectToSeconds(getTimeAsObject(timeString))}
