export const getHeight = (elementId, modifier, defaultValue) => {
    const maybeSendButton = document.getElementById(elementId);
    if (maybeSendButton) {
        const styles = window.getComputedStyle(maybeSendButton);
        const margin = parseFloat(styles['marginTop']) + parseFloat(styles['marginBottom']);
        return modifier(Math.ceil(maybeSendButton.offsetHeight + margin));
    }
    return defaultValue;
}

export const getWebsocketUrlPrefix = () => {
    return ((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host
}

const defaultResolution = 'vga';

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

const defaultCodec = 'h264';

export const KEY_CODEC = 'codec';

export const getCodec = () => {
    let got = localStorage.getItem(KEY_CODEC);
    if (!got) {
        localStorage.setItem(KEY_CODEC, defaultCodec);
        got = localStorage.getItem(KEY_CODEC);
    }
    return got;
}

export const setCodec = (newCodec) => {
    localStorage.setItem(KEY_CODEC, newCodec);
}

export const KEY_VIDEO_PRESENTS = 'videoPresents';
export const KEY_AUDIO_PRESENTS = 'audioPresents';

export const getStoredVideoDevicePresents = () => {
    let v = JSON.parse(localStorage.getItem(KEY_VIDEO_PRESENTS));
    if (v === null) {
        console.log("Resetting video presents to default");
        setStoredVideoPresents(true);
        v = JSON.parse(localStorage.getItem(KEY_VIDEO_PRESENTS));
    }
    return v;
}

export const setStoredVideoPresents = (v) => {
    localStorage.setItem(KEY_VIDEO_PRESENTS, JSON.stringify(v));
}

export const getStoredAudioDevicePresents = () => {
    let v = JSON.parse(localStorage.getItem(KEY_AUDIO_PRESENTS));
    if (v === null) {
        console.log("Resetting audio presents to default");
        setStoredAudioPresents(true);
        v = JSON.parse(localStorage.getItem(KEY_AUDIO_PRESENTS));
    }
    return v;
}

export const setStoredAudioPresents = (v) => {
    localStorage.setItem(KEY_AUDIO_PRESENTS, JSON.stringify(v));
}

export const KEY_LANGUAGE= 'language';

export const getStoredLanguage = () => {
    let v = JSON.parse(localStorage.getItem(KEY_LANGUAGE));
    if (v === null) {
        console.log("Resetting language to default");
        setStoredLanguage('en');
        v = JSON.parse(localStorage.getItem(KEY_LANGUAGE));
    }
    return v;
}

export const setStoredLanguage = (v) => {
    localStorage.setItem(KEY_LANGUAGE, JSON.stringify(v));
}

export const findIndex = (array, element) => {
    return array.findIndex(value => value.id === element.id);
};

export const replaceInArray = (array, element) => {
    const foundIndex = findIndex(array, element);
    if (foundIndex === -1) {
        return false;
    } else {
        array[foundIndex] = element;
        return true;
    }
};

export const replaceOrAppend = (array, newArray) => {
    newArray.forEach((element, index) => {
        const replaced = replaceInArray(array, element);
        if (!replaced) {
            array.push(element);
        }
    });
};

export const moveToFirstPosition = (array, element) => {
    const idx = findIndex(array, element);
    if (idx > 0) {
        array.splice(idx, 1);
        array.unshift(element);
    }
}

export const hasLength = (str) => {
    if (!str) {
        return false
    } else {
        return str.length
    }
}

export const setIcon = (newMessages) => {
    var link = document.querySelector("link[rel~='icon']");
    if (!link) {
        link = document.createElement('link');
        link.rel = 'icon';
        document.getElementsByTagName('head')[0].appendChild(link);
    }
    if (newMessages) {
        link.href = '/favicon_new.svg';
    } else {
        link.href = '/favicon.svg';
    }
}
