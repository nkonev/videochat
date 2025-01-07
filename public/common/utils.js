import { format, parseISO, differenceInDays } from 'date-fns';
import {chat, messageIdHashPrefix} from "./router/routes.js";
import bus, {PLAYER_MODAL} from "./bus.js";
import axios from "axios";
import he from "he";

export const getHumanReadableDate = (timestamp) => {
    const parsedDate = parseISO(timestamp);
    let formatString = 'HH:mm:ss';
    if (differenceInDays(new Date(), parsedDate) >= 1) {
        formatString = formatString + ', d MMM yyyy';
    }
    return `${format(parsedDate, formatString)}`
}

export const hasLength = (str) => {
    if (!str) {
        return false
    } else {
        return !!str.length
    }
}

export const replaceOrAppend = (array, newArray) => {
    newArray.forEach((element, index) => {
        const replaced = replaceInArray(array, element);
        if (!replaced) {
            array.push(element);
        }
    });
};

export const replaceOrPrepend = (array, newArray) => {
    newArray.forEach((element, index) => {
        const replaced = replaceInArray(array, element);
        if (!replaced) {
            array.unshift(element);
        }
    });
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

export const findIndex = (array, element) => {
    return array.findIndex(value => value.id === element.id);
};

export const findIndexNonStrictly = (array, element) => {
    return array.findIndex(value => value.id == element.id);
};

export const PAGE_SIZE = 40;

export const SEARCH_MODE_POSTS = "qp"

export const PAGE_PARAM = "page"

export const embed_message_reply = "reply";
export const embed_message_resend = "resend";

export const linkColor = '#1976D2' // see also in App.vue

export const getLoginColoredStyle = (item, defaultLinkColor) => {
    const color = item?.loginColor;
    const defaultColor = defaultLinkColor ? linkColor : null;
    return color ? {'color': color} : {'color': defaultColor}
}

export const getMessageLink = (chatId, messageId) => {
    return chat + "/" + chatId + messageIdHashPrefix + messageId
}

export const checkUpByTreeObj = (el, maxLevels, condition) => {
    let level = 0;
    let underCheck = el;
    do {
        if (condition(underCheck)) {
            return {
                found: true,
                el: underCheck
            }
        }
        underCheck = underCheck.parentElement;
        level++;
    } while (level <= maxLevels);
    return {
        found: false
    };
}

const createVideoReplacementElement = (src, poster) => {
    const replacement = document.createElement("VIDEO");
    replacement.src = src;
    replacement.poster = poster;
    replacement.playsinline = true;
    replacement.controls = true;
    replacement.className = "video-custom-class";
    return replacement
}

const createAudioReplacementElement = (src) => {
    const replacement = document.createElement("AUDIO");
    replacement.src = src;
    replacement.controls = true;
    replacement.className = "audio-custom-class";
    return replacement
}

const videoConvertingClass = "video-converting";

export const onClickTrap = (e) => {
    const foundElements = [
        checkUpByTreeObj(e?.target, 0, (el) => el?.tagName?.toLowerCase() == "img" && !el?.parentElement.classList?.contains("media-in-message-wrapper")),
        checkUpByTreeObj(e?.target, 0, (el) => el?.tagName?.toLowerCase() == "span" && el?.classList?.contains("media-in-message-button-open")),
        checkUpByTreeObj(e?.target, 0, (el) => el?.tagName?.toLowerCase() == "span" && el?.classList?.contains("media-in-message-button-replace")),
    ].filter(r => r.found);
    if (foundElements.length) {
        e.preventDefault();
        const found = foundElements[foundElements.length - 1].el;
        switch (found?.tagName?.toLowerCase()) {
            case "img": {
                const src = hasLength(found.getAttribute('data-original')) ? found.getAttribute('data-original') : found.src; // found.src is legacy
                bus.emit(PLAYER_MODAL, {canShowAsImage: true, url: src, canSwitch: true})
                break;
            }
            case "span": { // span of any of "show in player" or "replace" button
                const spanContainer = found.parentElement;
                if (spanContainer.classList.contains("media-in-message-wrapper")) {
                    if (found.classList?.contains("media-in-message-button-open")) { // "show in player" button
                        const theHolder = Array.from(spanContainer?.children).find(ch => ch?.tagName?.toLowerCase() == "img");
                        if (theHolder) {
                            if (!theHolder.classList.contains(videoConvertingClass)) {
                                const playerReq = {
                                    canSwitch: true,
                                    url: theHolder.getAttribute('data-original'),
                                    previewUrl: theHolder.src,
                                }
                                if (spanContainer.classList.contains("media-in-message-wrapper-video")) {
                                    playerReq.canPlayAsVideo = true
                                } else if (spanContainer.classList.contains("media-in-message-wrapper-audio")) {
                                    playerReq.canPlayAsAudio = true
                                }
                                bus.emit(PLAYER_MODAL, playerReq);
                            }
                        }

                    } else if (found.classList?.contains("media-in-message-button-replace")) { // "replace" button
                        const theHolder = Array.from(spanContainer?.children).find(ch => ch?.tagName?.toLowerCase() == "img");
                        if (theHolder) {
                            const src = theHolder.src;
                            const original = theHolder.getAttribute('data-original');

                            if (spanContainer.classList.contains("media-in-message-wrapper-video")) {
                                spanContainer.removeChild(theHolder);
                                spanContainer.removeChild(found);

                                const videoReplacement = createVideoReplacementElement(original, src);
                                spanContainer.appendChild(videoReplacement);

                                axios.post(`/api/storage/public/view/status`, {
                                    url: original
                                }).then(res => {
                                    if (res.data.status == "converting") {
                                        spanContainer.removeChild(videoReplacement);

                                        const imgReplacement = document.createElement("IMG");
                                        imgReplacement.src = res.data.statusImage;
                                        imgReplacement.className = "video-custom-class " + videoConvertingClass;
                                        spanContainer.appendChild(imgReplacement);
                                    }
                                })
                            } else if (spanContainer.classList.contains("media-in-message-wrapper-audio")) {
                                spanContainer.removeChild(theHolder);
                                spanContainer.removeChild(found);

                                const openButton = Array.from(spanContainer?.children).find(ch => ch?.classList?.contains("media-in-message-button-open"));
                                spanContainer.removeChild(openButton);

                                const audioReplacement = createAudioReplacementElement(original);
                                spanContainer.appendChild(audioReplacement);

                                axios.post(`/api/storage/public/view/status`, {
                                    url: original
                                }).then((res) => {
                                    const p = document.createElement("P");
                                    p.textContent=res.data?.filename;
                                    spanContainer.prepend(p);
                                })
                            } else {
                                console.info("no case for it")
                            }
                        } else {
                            console.info("holder is not found")
                        }
                    }
                }
                break;
            }
        }
    }
}

export const getUrlPrefix = () => {
    return window.location.protocol + "//" + window.location.host
}

export const unescapeHtml = (text) => {
    if (!text) {
        return text
    }
    return he.decode(text);
}
