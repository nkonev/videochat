export const getHeight = (elementId, modifier, defaultValue) => {
    const maybeSendButton = document.getElementById(elementId);
    if (maybeSendButton) {
        const styles = window.getComputedStyle(maybeSendButton);
        const margin = parseFloat(styles['marginTop']) + parseFloat(styles['marginBottom']);
        return modifier(Math.ceil(maybeSendButton.offsetHeight + margin));
    }
    return defaultValue;
}

export const getCorrectUserAvatar = (stringExistsAvatar) => {
    if (!stringExistsAvatar) {
        return stringExistsAvatar;
    }
    const cacheKey = +new Date();
    return stringExistsAvatar + "?" + cacheKey;
}

export const getWebsocketUrlPrefix = () => {
    return ((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host
}

export const audioMuteDefault = true;

const defaultResolution = 'hd';

export const KEY_RESOLUTION = 'videoResolution';

export const getVideoResolution = () => {
    let got = localStorage.getItem(KEY_RESOLUTION);
    if (!got) {
        localStorage.setItem(KEY_RESOLUTION, defaultResolution);
        got = localStorage.getItem(KEY_RESOLUTION);
    }
    return got;
}

export const setVideoResolution = (newVideoResolution) => {
    localStorage.setItem(KEY_RESOLUTION, newVideoResolution);
}
