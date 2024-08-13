import {
    blog_post,
    chat,
    chat_name, chatIdHashPrefix,
    messageIdHashPrefix,
    prefix,
    public_prefix, userIdHashPrefix,
    video_suffix,
    videochat_name
} from "@/router/routes";
import he from "he";

export const isMobileBrowser = () => {
    return navigator.userAgent.indexOf('Mobile') !== -1
}

export const isMobileFireFox = () => {
  return navigator.userAgent.indexOf('Firefox') !== -1 && isMobileBrowser()
}

export const isFireFox = () => {
  return navigator.userAgent.indexOf('Firefox') !== -1
}

export const isMobileWidth = (width) => {
    return width < 800 // same as $mobileWidth in constants.styl
}

export const hasLength = (str) => {
    if (!str) {
        return false
    } else {
        return !!str.length
    }
}

export const isSet = (str) => {
    return str != null
}

export const setTitle = (newTitle) => {
    if (hasLength(newTitle)) {
        document.title = unescapeHtml(newTitle);
    } else {
        document.title = "VideoChat"
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
    link.href = `${prefix}/favicon_new.svg`;
  } else {
    link.href = `${prefix}/favicon.svg`;
  }
}

export const deepCopy = (aVal) => {
    return JSON.parse(JSON.stringify(aVal))
}

export const embed_message_reply = "reply";
export const embed_message_resend = "resend";

export const getBlogLink = (chatId) => {
    return blog_post + "/" + chatId;
}

export const getPublicMessageLink = (chatId, messageId) => {
    // see also public_message in routes.js
    return getUrlPrefix() + public_prefix + chat + '/' + chatId + '/message/' + messageId;
}

export const getMessageLinkRouteObject = (chatId, messageId) => {
    return {
        name: chat_name,
        params: {
            id: chatId
        },
        hash: messageIdHashPrefix + messageId,
    };
}

export const gotoMessageLink = (router, chatId, messageId) => {
    const routeObj = getMessageLinkRouteObject(chatId, messageId);
    router.push(routeObj);
}

export const getMessageLink = (chatId, messageId) => {
    return getUrlPrefix() + chat + "/" + chatId + messageIdHashPrefix + messageId
}

export const media_image = "image";

export const media_video = "video";

export const media_audio = "audio";

export const embed = "embed";


export const link_dialog_type_add_link_to_text = "add_link_to_text";
export const link_dialog_type_add_media_by_link = "add_media_by_link";
export const link_dialog_type_add_media_embed = "add_media_embed";

export const chatEditMessageDtoFactory = () => {
  return {
    id: null,
    text: "",
    fileItemUuid: null,
  }
};


export const colorText = 'colorText';
export const colorBackground = 'colorBackground';

export const colorLogin = 'colorLogin';


export const getUrlPrefix = () => {
  return window.location.protocol + "//" + window.location.host
}

export const getWebsocketUrlPrefix = () => {
  return ((window.location.protocol === "https:") ? "wss://" : "ws://") + window.location.host
}

export const findIndex = (array, element) => {
  return array.findIndex(value => value.id === element.id);
};

export const findIndexNonStrictly = (array, element) => {
  return array.findIndex(value => value.id == element.id);
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

export const replaceOrPrepend = (array, newArray) => {
  newArray.forEach((element, index) => {
    const replaced = replaceInArray(array, element);
    if (!replaced) {
      array.unshift(element);
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

export const setAnswerPreviewFields = (dto, messageText, ownerLogin) => {
  // used only to show on front, ignored in message create machinery
  dto.embedMessage.embedPreviewText = messageText;
  dto.embedMessage.embedPreviewOwner = ownerLogin;
}

export const getAnswerPreviewFields = (dto) => {
  return dto.embedMessage;
}

export const haveEmbed = (dto) => {
  return !!dto.embedMessage;
}

export const getEmbed = (dto) => {
  return dto.embedMessage;
}

export const setEmbed = (dto, e) => {
  dto.embedMessage = e;
}

export const edit_message = 'editMessage'
export const reply_message = 'replyMessage'

export const new_message = 'newMessage'

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

export const publicallyAvailableForSearchChatsQuery = "__AVAILABLE_FOR_SEARCH";

export const isSetEqual = (a, b) => {
    if (a == null && b == null) {
        return true
    } else if (a == null && b != null) {
        return false
    } else if (a != null && b == null) {
        return false
    } else {
      const first = new Set(a);
      const second = new Set(b);
      return first.size === second.size &&
        [...first].every((x) => second.has(x))
    }
}

export function dynamicSort(property) {
    var sortOrder = 1;
    if(property[0] === "-") {
        sortOrder = -1;
        property = property.substring(1);
    }
    return function (a,b) {
        /* next line works with strings and numbers,
         * and you may want to customize it to your needs
         */
        var result = (a[property] < b[property]) ? -1 : (a[property] > b[property]) ? 1 : 0;
        return result * sortOrder;
    }
}

export function dynamicSortMultiple() {
    /*
     * save the arguments object as it will be overwritten
     * note that arguments object is an array-like object
     * consisting of the names of the properties to sort by
     */
    var props = arguments;
    return function (obj1, obj2) {
        var i = 0, result = 0, numberOfProperties = props.length;
        /* try getting a different result from 0 (equal)
         * as long as we have extra properties to compare
         */
        while(result === 0 && i < numberOfProperties) {
            result = dynamicSort(props[i])(obj1, obj2);
            i++;
        }
        return result;
    }
}

export const getChatLink = (chatId) => {
    const link = getUrlPrefix() + chat + '/' + chatId;
    return link;
}

export const copyChatLink = (chatId) => {
  const link = getUrlPrefix() + chat + '/' + chatId;
  navigator.clipboard.writeText(link);
}

export const copyCallLink = (chatId) => {
  const link = getUrlPrefix() + chat + '/' + chatId + video_suffix;
  navigator.clipboard.writeText(link);
}

export const isChatRoute = (route) => {
  return route.name == chat_name || route.name == videochat_name
}

// #message-1
export const isMessageHash = (hash) => {
    return hash?.startsWith(messageIdHashPrefix)
}

// #chat-1
export const isChatHash = (hash) => {
    hash?.startsWith(chatIdHashPrefix)
}

export const isUserHash = (hash) => {
    hash?.startsWith(userIdHashPrefix)
}

export const getIdFromRouteHash = (hash) => {
    if (!hash) {
        return null;
    }
    const str = hash.replace(/\D/g, '');
    return hasLength(str) ? str : null;
};

export const defaultAudioMute = true;

export const renameFilePart = (file, newFileName) => {
  const formData = new FormData();
  const partName = "File";
  formData.append(partName, file, newFileName);
  const renamedFile = formData.get(partName);
  return renamedFile
}

export const isCalling = (status) => {
  return status == "beingInvited"
}

export const setLanguageToVuetify = (that, newLanguage) => {
    that.$vuetify.locale.current = newLanguage;
}

export const linkColor = '#1976D2' // see also in App.vue

export const getLoginColoredStyle = (item, defaultLinkColor) => {
    const color = item?.loginColor;
    const defaultColor = defaultLinkColor ? linkColor : null;
    return color ? {'color': color} : {'color': defaultColor}
}

export const getNotificationSubtitle = (vuetify, item) => {
    switch (item.notificationType) {
        case "missed_call":
            return vuetify.locale.t('$vuetify.notification_missed_call', item.byLogin)
        case "mention":
            let builder1 = vuetify.locale.t('$vuetify.notification_mention', item.byLogin)
            if (hasLength(item.chatTitle)) {
                builder1 += (vuetify.locale.t('$vuetify.in') + "'" + item.chatTitle + "'");
            }
            return builder1
        case "reply":
            let builder2 = vuetify.locale.t('$vuetify.notification_reply', item.byLogin)
            if (hasLength(item.chatTitle)) {
                builder2 += (vuetify.locale.t('$vuetify.in') + "'" + item.chatTitle + "'")
            }
            return builder2
        case "reaction":
            let builder3 = vuetify.locale.t('$vuetify.notification_reaction', item.byLogin)
            if (hasLength(item.chatTitle)) {
                builder3 += (vuetify.locale.t('$vuetify.in') + "'" + item.chatTitle + "'")
            }
            return builder3
    }
}

export const getNotificationTitle = (item) => {
    return item.description
}

export const checkUpByTree = (el, maxLevels, condition) => {
    return checkUpByTreeObj(el, maxLevels, condition).found
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

export const PURPOSE_CALL = 'call';
export const PURPOSE_RECORDING = 'recording';

export const isStrippedUserLogin = (u) => {
    if (u == null) {
        return false
    }
    return u.additionalData && (!u.additionalData.confirmed || u.additionalData.locked || !u.additionalData.enabled)
}

export const isConverted = (name) => {
    return name != null && name.includes("_converted.")
}

export const unescapeHtml = (text) => {
    if (!text) {
        return text
    }
    return he.decode(text);
}

export const getExtendedUserFragment = (reqEmail) => {
    const email = `
email,
awaitingForConfirmEmailChange,
`;

    return `
... on UserAccountExtendedDto {
  id,
  login,
  ${reqEmail ? email : ""}
  avatar,
  avatarBig,
  shortInfo,
  lastLoginDateTime,
  oauth2Identifiers {
    facebookId,
    vkontakteId,
    googleId,
    keycloakId,
  },
  additionalData, {
    enabled,
    expired,
    locked,
    confirmed,
    roles,
  },
  canLock,
  canEnable,
  canDelete,
  canChangeRole,
  canConfirm,
  loginColor,
  canRemoveSessions,
  ldap,
  canSetPassword
}
`
}

export const isFullscreen = () => {
    return !!(document.fullscreenElement)
}

export const loadingMessage = 'Loading...';

export const goToPreservingQuery = (route, router, to) => {
    const prev = deepCopy(route.query);
    return router.push({ ...to, query: prev })
}

export const stopCall = (chatStore, route, router) => {
    chatStore.leavingVideoAcceptableParam = true;
    const routerNewState = { name: chat_name };
    goToPreservingQuery(route, router, routerNewState);
}