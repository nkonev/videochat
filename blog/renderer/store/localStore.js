const defaultResolution = 'h720';

export const KEY_VIDEO_RESOLUTION = 'videoResolution';
export const KEY_SCREEN_RESOLUTION = 'screenResolution';

export const getVideoResolution = () => {
    let got = localStorage.getItem(KEY_VIDEO_RESOLUTION);
    if (!got) {
        localStorage.setItem(KEY_VIDEO_RESOLUTION, defaultResolution);
        got = localStorage.getItem(KEY_VIDEO_RESOLUTION);
    }
    return got;
}

export const NULL_SCREEN_RESOLUTION = 'null';

export const getScreenResolution = () => {
    let got = JSON.parse(localStorage.getItem(KEY_SCREEN_RESOLUTION));
    if (got === null) {
        setScreenResolution(NULL_SCREEN_RESOLUTION);
        got = JSON.parse(localStorage.getItem(KEY_SCREEN_RESOLUTION));
    }
    return got;
}

export const setVideoResolution = (newVideoResolution) => {
    localStorage.setItem(KEY_VIDEO_RESOLUTION, newVideoResolution);
}

export const setScreenResolution = (newVideoResolution) => {
    localStorage.setItem(KEY_SCREEN_RESOLUTION, JSON.stringify(newVideoResolution));
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

export const KEY_LANGUAGE = 'language';

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

export const KEY_VIDEO_SIMULCAST = 'videoSimulcast';
export const KEY_SCREEN_SIMULCAST = 'screenSimulcast';

export const getStoredVideoSimulcast = () => {
    let v = JSON.parse(localStorage.getItem(KEY_VIDEO_SIMULCAST));
    if (v === null) {
        console.log("Resetting video simulcast to default");
        setStoredVideoSimulcast(true);
        v = JSON.parse(localStorage.getItem(KEY_VIDEO_SIMULCAST));
    }
    return v;
}

export const setStoredVideoSimulcast = (v) => {
    localStorage.setItem(KEY_VIDEO_SIMULCAST, JSON.stringify(v));
}

export const getStoredScreenSimulcast = () => {
    let v = JSON.parse(localStorage.getItem(KEY_SCREEN_SIMULCAST));
    if (v === null) {
        console.log("Resetting screen simulcast presents to default");
        setStoredScreenSimulcast(true);
        v = JSON.parse(localStorage.getItem(KEY_SCREEN_SIMULCAST));
    }
    return v;
}

export const setStoredScreenSimulcast = (v) => {
    localStorage.setItem(KEY_SCREEN_SIMULCAST, JSON.stringify(v));
}

export const KEY_ROOM_DYNACAST = 'roomDynacast';

export const getStoredRoomDynacast = () => {
    let v = JSON.parse(localStorage.getItem(KEY_ROOM_DYNACAST));
    if (v === null) {
        console.log("Resetting video dynacast to default");
        setStoredRoomDynacast(true);
        v = JSON.parse(localStorage.getItem(KEY_ROOM_DYNACAST));
    }
    return v;
}

export const setStoredRoomDynacast = (v) => {
    localStorage.setItem(KEY_ROOM_DYNACAST, JSON.stringify(v));
}

export const KEY_ROOM_ADAPTIVE_STREAM = 'roomAdaptiveStream';

export const getStoredRoomAdaptiveStream = () => {
    let v = JSON.parse(localStorage.getItem(KEY_ROOM_ADAPTIVE_STREAM));
    if (v === null) {
        console.log("Resetting adaptive stream to default");
        setStoredRoomAdaptiveStream(true);
        v = JSON.parse(localStorage.getItem(KEY_ROOM_ADAPTIVE_STREAM));
    }
    return v;
}

export const setStoredRoomAdaptiveStream = (v) => {
    localStorage.setItem(KEY_ROOM_ADAPTIVE_STREAM, JSON.stringify(v));
}


export const VIDEO_POSITION_AUTO = 'auto';
export const VIDEO_POSITION_ON_THE_TOP = 'onTheTop'; // as usual
export const VIDEO_POSITION_SIDE = 'side'; // new

export const KEY_VIDEO_POSITION = 'videoPosition';

export const getStoredVideoPosition = () => {
    let v = JSON.parse(localStorage.getItem(KEY_VIDEO_POSITION));
    if (v === null) {
        console.log("Resetting videoPosition to default");
        setStoredVideoPosition(VIDEO_POSITION_AUTO);
        v = JSON.parse(localStorage.getItem(KEY_VIDEO_POSITION));
    }
    return v;
}

export const setStoredVideoPosition = (v) => {
    localStorage.setItem(KEY_VIDEO_POSITION, JSON.stringify(v));
}

export const NULL_CODEC = 'null';

export const KEY_CODEC = 'codec';
export const getStoredCodec = () => {
    let v = JSON.parse(localStorage.getItem(KEY_CODEC));
    if (v === null) {
        console.log("Resetting codec to default");
        setStoredCodec(NULL_CODEC);
        v = JSON.parse(localStorage.getItem(KEY_CODEC));
    }
    return v;
}

export const setStoredCodec = (v) => {
    localStorage.setItem(KEY_CODEC, JSON.stringify(v));
}

const KEY_TOP_MESSAGE = "topMessage"
export const setTopMessagePosition = (chatId, messageId) => {
  localStorage.setItem(KEY_TOP_MESSAGE + "_" + chatId, JSON.stringify(messageId));
}

export const getTopMessagePosition = (chatId) => {
  return JSON.parse(localStorage.getItem(KEY_TOP_MESSAGE + "_" + chatId));
}

export const removeTopMessagePosition = (chatId) => {
  localStorage.removeItem(KEY_TOP_MESSAGE + "_" + chatId);
}

const KEY_TOP_USER = "topUser"

export const setTopUserPosition = (userId) => {
    localStorage.setItem(KEY_TOP_USER, JSON.stringify(userId));
}

export const getTopUserPosition = () => {
    return JSON.parse(localStorage.getItem(KEY_TOP_USER));
}

export const removeTopUserPosition = () => {
    localStorage.removeItem(KEY_TOP_USER);
}


const KEY_TOP_BLOG = "topBlog"

export const setTopBlogPosition = (userId) => {
    localStorage.setItem(KEY_TOP_BLOG, JSON.stringify(userId));
}

export const getTopBlogPosition = () => {
    return JSON.parse(localStorage.getItem(KEY_TOP_BLOG));
}

export const removeTopBlogPosition = () => {
    localStorage.removeItem(KEY_TOP_BLOG);
}
