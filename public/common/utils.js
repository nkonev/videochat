import {
    userP,
    chatP,
    chat,
    messageIdHashPrefix,
    profile,
    publicP,
    blogP,
    postP,
    path_prefix,
    blog_post
} from "./router/routes.js";
import bus, {PLAYER_MODAL} from "./bus.js";
import axios from "axios";
import he from "he";
import { navigate } from 'vike/client/router';

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

export const FIRST_PAGE = 1;
export const PAGE_SIZE = 40;
export const PAGE_SIZE_SMALL = 20;

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

const createIframeReplacementElement = (src, width, height, allowfullscreen) => {
    const replacement = document.createElement("IFRAME");
    replacement.src = src;
    replacement.setAttribute('width', width);
    replacement.setAttribute('height', height);
    if (allowfullscreen) {
        replacement.setAttribute('allowFullScreen', '')
    }
    replacement.className = "iframe-custom-class";
    return replacement
}

const videoConvertingClass = "video-converting";

export const onClickTrap = (e) => {
    const foundElements = [
        checkUpByTreeObj(e?.target, 0, (el) => el?.tagName?.toLowerCase() == "img" && !el?.parentElement.classList?.contains("media-in-message-wrapper")),
        checkUpByTreeObj(e?.target, 0, (el) => el?.tagName?.toLowerCase() == "span" && el?.classList?.contains("media-in-message-button-open")),
        checkUpByTreeObj(e?.target, 0, (el) => el?.tagName?.toLowerCase() == "span" && el?.classList?.contains("media-in-message-button-replace")),
        checkUpByTreeObj(e?.target, 1, (el) => el?.tagName?.toLowerCase() == "a"), // 1 is to handle struck links
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

                                const openButton = Array.from(spanContainer.children).find(ch => ch?.classList?.contains("media-in-message-button-open"));
                                if (openButton) {
                                    spanContainer.removeChild(openButton);
                                }

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
                            } else if (spanContainer.classList.contains("media-in-message-wrapper-iframe")) {
                                const width = theHolder.getAttribute('data-width');
                                const height = theHolder.getAttribute('data-height');
                                const allowfullscreen = theHolder.getAttribute('data-allowfullscreen');

                                spanContainer.removeChild(theHolder);
                                spanContainer.removeChild(found);

                                const iframeReplacement = createIframeReplacementElement(original, width, height, allowfullscreen);
                                spanContainer.appendChild(iframeReplacement);
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
            case "a": {
                const href = found.getAttribute("href");
                if (found.classList?.contains("mention")) {
                    const userId = found.getAttribute('data-id');
                    if (hasLength(userId)) {
                        navigate(profile + "/" + userId);
                    }
                    break;
                } else if (href.startsWith("/")) {
                    // try to parse message link and go to it - only "/chat/1000#message-1", regardless in video call we are or not
                    console.info("examining internal link", href);

                    const messageObj = parseMessageLink(href);
                    if (messageObj) {
                        console.info("href", href, "is recognized as message", messageObj);
                        navigate(chat + "/" + messageObj.chatId + messageIdHashPrefix + messageObj.id);
                        break;
                    }

                    const chatObj = parseChatLink(href);
                    if (chatObj) {
                        console.info("href", href, "is recognized as chat", chatObj);
                        navigate(chat + "/" + chatObj.chatId);
                        break;
                    }

                    const userObj = parseUserLink(href);
                    if (userObj) {
                        console.info("href", href, "is recognized as user", userObj);
                        navigate(profile + "/" + userObj.userId);
                        break;
                    }

                    const blogObj = parseBlogLink(href);
                    if (blogObj) {
                        console.info("href", href, "is recognized as blog", blogObj);
                        navigate(path_prefix + blog_post + "/" + blogObj.postId);
                        break;
                    }
                }
                window.open(href, '_blank').focus();
            }
        }
    }
}

export const getIdFromRouteHash = (hash) => {
    if (!hash) {
        return null;
    }
    const str = hash.replace(/\D/g, '');
    return hasLength(str) ? str : null;
};

// "/chat/1000#message-1"
export const parseMessageLink = (href) => {
    try {
        const url = new URL(getUrlPrefix() + href);
        const pathArray = url.pathname.split('/');
        if (pathArray.length) {
            if (pathArray[1] == chatP) {
                const chatId = parseInt(pathArray[2]);
                const maybeMessageId = getIdFromRouteHash(url.hash);
                if (maybeMessageId) {
                    const messageId = parseInt(maybeMessageId);
                    return {
                        chatId: chatId,
                        id: messageId
                    }
                }
            }
        }
        return null
    } catch (ignore) {
        return null
    }
}

// /public/blog/post/2
export const parseBlogLink = (href) => {
    try {
        const url = new URL(getUrlPrefix() + href);
        const pathArray = url.pathname.split('/');
        if (pathArray.length > 4) {
            if (pathArray[1] == publicP && pathArray[2] == blogP && pathArray[3] == postP) {
                const postId = parseInt(pathArray[4]);
                return {
                    postId: postId,
                }
            }
        }
        return null
    } catch (ignore) {
        return null
    }
}

// /chat/1
export const parseChatLink = (href) => {
    try {
        const url = new URL(getUrlPrefix() + href);
        const pathArray = url.pathname.split('/');
        if (pathArray.length) {
            if (pathArray[1] == chatP) {
                const chatId = parseInt(pathArray[2]);
                return {
                    chatId: chatId,
                }
            }
        }
        return null
    } catch (ignore) {
        return null
    }
}

// /user/1
export const parseUserLink = (href) => {
    try {
        const url = new URL(getUrlPrefix() + href);
        const pathArray = url.pathname.split('/');
        if (pathArray.length) {
            if (pathArray[1] == userP) {
                const userId = parseInt(pathArray[2]);
                return {
                    userId: userId,
                }
            }
        }
        return null
    } catch (ignore) {
        return null
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

export const formatSize = (size) => {
    const operableSize = Math.abs(size);
    if (operableSize > 1024 * 1024 * 1024 * 1024) {
        return (size / 1024 / 1024 / 1024 / 1024).toFixed(2) + ' TB'
    } else if (operableSize > 1024 * 1024 * 1024) {
        return (size / 1024 / 1024 / 1024).toFixed(2) + ' GB'
    } else if (operableSize > 1024 * 1024) {
        return (size / 1024 / 1024).toFixed(2) + ' MB'
    } else if (operableSize > 1024) {
        return (size / 1024).toFixed(2) + ' KB'
    }
    return size.toString() + ' B'
};

export const deepCopy = (aVal) => {
    return JSON.parse(JSON.stringify(aVal))
}

export const isMobileBrowser = () => {
    return navigator.userAgent.indexOf('Mobile') !== -1
}

export const isStrippedUserLogin = (u) => {
    if (u == null) {
        return false
    }
    return u.additionalData && (!u.additionalData.confirmed || u.additionalData.locked || !u.additionalData.enabled)
}
