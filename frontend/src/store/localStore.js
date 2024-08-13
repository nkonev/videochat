import {hasLength} from "@/utils";

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

export const KEY_CHAT_EDIT_MESSAGE_DTO = 'chatEditMessageDto';

export const getStoredChatEditMessageDto = (chatId, defVal) => {
    let v = JSON.parse(localStorage.getItem(KEY_CHAT_EDIT_MESSAGE_DTO + '_' + chatId));
    if (v === null) {
        return defVal;
    }
    return v;
}

export const getStoredChatEditMessageDtoOrNull = (chatId) => {
  let v = JSON.parse(localStorage.getItem(KEY_CHAT_EDIT_MESSAGE_DTO + '_' + chatId));
  return v;
}

export const setStoredChatEditMessageDto = (v, chatId) => {
    if (hasLength(v.text)) {
        localStorage.setItem(KEY_CHAT_EDIT_MESSAGE_DTO + '_' + chatId, JSON.stringify(v));
    } else {
        removeStoredChatEditMessageDto(chatId)
    }
}

export const removeStoredChatEditMessageDto = (chatId) => {
    localStorage.removeItem(KEY_CHAT_EDIT_MESSAGE_DTO + '_' + chatId);
}

export const VIDEO_POSITION_HORIZONTAL = 'horizontal'; // as usual
export const VIDEO_POSITION_VERTICAL = 'vertical'; // new
export const VIDEO_POSITION_GALLERY = 'gallery';

export const KEY_VIDEO_POSITION = 'videoPosition';

export const getStoredVideoPosition = () => {
    let v = JSON.parse(localStorage.getItem(KEY_VIDEO_POSITION));
    if (v === null) {
        console.log("Resetting videoPosition to default", VIDEO_POSITION_HORIZONTAL);
        setStoredVideoPosition(VIDEO_POSITION_HORIZONTAL);
        v = JSON.parse(localStorage.getItem(KEY_VIDEO_POSITION));
    }
    return v;
}

export const setStoredVideoPosition = (v) => {
    localStorage.setItem(KEY_VIDEO_POSITION, JSON.stringify(v));
}

export const positionItems = () => {
    return [VIDEO_POSITION_HORIZONTAL, VIDEO_POSITION_VERTICAL, VIDEO_POSITION_GALLERY]
}


export const KEY_PRESENTER = 'presenter';

export const getStoredPresenter = () => {
    let v = JSON.parse(localStorage.getItem(KEY_PRESENTER));
    if (v === null) {
        console.log("Resetting presenter to default");
        setStoredPresenter(true);
        v = JSON.parse(localStorage.getItem(KEY_PRESENTER));
    }
    return v;
}

export const setStoredPresenter = (v) => {
    localStorage.setItem(KEY_PRESENTER, JSON.stringify(v));
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


const KEY_TOP_CHAT = "topChat"

export const setTopChatPosition = (chatId) => {
    localStorage.setItem(KEY_TOP_CHAT, JSON.stringify(chatId));
}

export const getTopChatPosition = () => {
    return JSON.parse(localStorage.getItem(KEY_TOP_CHAT));
}

export const removeTopChatPosition = () => {
    localStorage.removeItem(KEY_TOP_CHAT);
}


export const KEY_MESSAGE_EDIT_NORMALIZE_TEXT = 'messageEditNormalizeText';

export const getStoredMessageEditNormalizeText = () => {
    let v = JSON.parse(localStorage.getItem(KEY_MESSAGE_EDIT_NORMALIZE_TEXT));
    if (v === null) {
        console.log("Resetting messageEditNormalizeText to default");
        setStoredMessageEditNormalizeText(true);
        v = JSON.parse(localStorage.getItem(KEY_MESSAGE_EDIT_NORMALIZE_TEXT));
    }
    return v;
}

export const setStoredMessageEditNormalizeText = (v) => {
    localStorage.setItem(KEY_MESSAGE_EDIT_NORMALIZE_TEXT, JSON.stringify(v));
}

export const KEY_TREAT_NEWLINES_AS_IN_HTML = 'treatNewlinesAsInHtml';

export const getTreatNewlinesAsInHtml = () => {
    let v = JSON.parse(localStorage.getItem(KEY_TREAT_NEWLINES_AS_IN_HTML));
    if (v === null) {
        console.log("Resetting treatNewlinesAsInHtml to default");
        setTreatNewlinesAsInHtml(true);
        v = JSON.parse(localStorage.getItem(KEY_TREAT_NEWLINES_AS_IN_HTML));
    }
    return v;
}

export const setTreatNewlinesAsInHtml = (v) => {
    localStorage.setItem(KEY_TREAT_NEWLINES_AS_IN_HTML, JSON.stringify(v));
}

export const KEY_MESSAGE_EDIT_SEND_BUTTONS_TYPE = 'messageEditSendButtonsType';

export const getStoredMessageEditSendButtonsType = (defaultValue) => {
    let v = JSON.parse(localStorage.getItem(KEY_MESSAGE_EDIT_SEND_BUTTONS_TYPE));
    if (v === null) {
        console.log("Resetting messageEditSendButtonsType to default");
        setStoredMessageEditSendButtonsType(defaultValue); // see MessageEditSettingsModalContent
        v = JSON.parse(localStorage.getItem(KEY_MESSAGE_EDIT_SEND_BUTTONS_TYPE));
    }
    return v;
}

export const setStoredMessageEditSendButtonsType = (v) => {
    localStorage.setItem(KEY_MESSAGE_EDIT_SEND_BUTTONS_TYPE, JSON.stringify(v));
}

export const KEY_RECORDING_TAB = 'recordingTab';

export const getStoreRecordingTab = (defaultValue) => {
    let v = JSON.parse(localStorage.getItem(KEY_RECORDING_TAB));
    if (v === null) {
        console.log("Resetting recordingTab to default");
        setStoreRecordingTab(defaultValue);
        v = JSON.parse(localStorage.getItem(KEY_RECORDING_TAB));
    }
    return v;
}

export const setStoreRecordingTab = (v) => {
    localStorage.setItem(KEY_RECORDING_TAB, JSON.stringify(v));
}


const KEY_NOTIFICATION_PREFIX = 'notification';

export const getBrowserNotification = (chatId, defaultValue, notificationType) => {
    let v = JSON.parse(localStorage.getItem(KEY_NOTIFICATION_PREFIX + "_" + chatId + "_" + notificationType));
    if (v === null && defaultValue !== null) {
        setBrowserNotification(chatId, notificationType, defaultValue);
        v = JSON.parse(localStorage.getItem(KEY_NOTIFICATION_PREFIX + "_" + chatId + "_" + notificationType));
    }
    return v;
}

export const setBrowserNotification = (chatId, notificationType, v) => {
    localStorage.setItem(KEY_NOTIFICATION_PREFIX + "_" + chatId + "_" + notificationType, JSON.stringify(v));
}

const global = 'global';
export const getGlobalBrowserNotification = (notificationType) => {
    return getBrowserNotification(global, false, notificationType)
}

export const setGlobalBrowserNotification = (notificationType, v) => {
    setBrowserNotification(global, notificationType, v)
}

export const NOTIFICATION_TYPE_MENTIONS = 'mentions';
export const NOTIFICATION_TYPE_MISSED_CALLS = 'missedCalls';
export const NOTIFICATION_TYPE_ANSWERS = 'answers';
export const NOTIFICATION_TYPE_REACTIONS = 'reactions';
export const NOTIFICATION_TYPE_NEW_MESSAGES = 'newMessages';
export const NOTIFICATION_TYPE_CALL = 'call';


export const KEY_RECORDING_VIDEO_DEVICE_ID = 'recordingVideoDeviceId';
export const getStoredRecordingVideoDeviceId = () => {
    return localStorage.getItem(KEY_RECORDING_VIDEO_DEVICE_ID);
}
export const setStoredRecordingVideoDeviceId = (v) => {
    localStorage.setItem(KEY_RECORDING_VIDEO_DEVICE_ID, v);
}

export const KEY_RECORDING_AUDIO_DEVICE_ID = 'recordingAudioDeviceId';
export const getStoredRecordingAudioDeviceId = () => {
    return localStorage.getItem(KEY_RECORDING_AUDIO_DEVICE_ID);
}
export const setStoredRecordingAudioDeviceId = (v) => {
    localStorage.setItem(KEY_RECORDING_AUDIO_DEVICE_ID, v);
}


export const KEY_CALL_VIDEO_DEVICE_ID = 'callVideoDeviceId';
export const getStoredCallVideoDeviceId = () => {
    return localStorage.getItem(KEY_CALL_VIDEO_DEVICE_ID);
}
export const setStoredCallVideoDeviceId = (v) => {
    localStorage.setItem(KEY_CALL_VIDEO_DEVICE_ID, v);
}

export const KEY_CALL_AUDIO_DEVICE_ID = 'callAudioDeviceId';
export const getStoredCallAudioDeviceId = () => {
    return localStorage.getItem(KEY_CALL_AUDIO_DEVICE_ID);
}
export const setStoredCallAudioDeviceId = (v) => {
    localStorage.setItem(KEY_CALL_AUDIO_DEVICE_ID, v);
}
