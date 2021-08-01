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

export const KEY_RESOLUTION = 'videoResolution';

export const KEY_VIDEO_PRESENTS = 'videoPresents';
export const KEY_AUDIO_PRESENTS = 'audioPresents';

export const getStoredVideoPresents = () => {
    let v = JSON.parse(localStorage.getItem(KEY_VIDEO_PRESENTS));
    if (v === null) {
        setStoredVideoPresents(true);
        v = JSON.parse(localStorage.getItem(KEY_VIDEO_PRESENTS));
    }
    return v;
}

export const setStoredVideoPresents = (v) => {
    localStorage.setItem(KEY_VIDEO_PRESENTS, JSON.stringify(v));
}

export const getStoredAudioPresents = () => {
    let v = JSON.parse(localStorage.getItem(KEY_AUDIO_PRESENTS));
    if (v === null) {
        setStoredAudioPresents(true);
        v = JSON.parse(localStorage.getItem(KEY_AUDIO_PRESENTS));
    }
    return v;
}

export const setStoredAudioPresents = (v) => {
    localStorage.setItem(KEY_AUDIO_PRESENTS, JSON.stringify(v));
}