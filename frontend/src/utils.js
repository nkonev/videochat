import { format, parseISO, differenceInDays } from 'date-fns';
import {blog_post, chat, chat_name, prefix, video_suffix, videochat_name} from "@/router/routes";

export const isMobileBrowser = () => {
    return navigator.userAgent.indexOf('Mobile') !== -1
}

export const isMobileFireFox = () => {
  return navigator.userAgent.indexOf('Firefox') !== -1 && isMobileBrowser()
}

export const isFireFox = () => {
  return navigator.userAgent.indexOf('Firefox') !== -1
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
  document.title = newTitle;
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

export const getHumanReadableDate = (timestamp) => {
    const parsedDate = parseISO(timestamp);
    let formatString = 'HH:mm:ss';
    if (differenceInDays(new Date(), parsedDate) >= 1) {
        formatString = formatString + ', d MMM yyyy';
    }
    return `${format(parsedDate, formatString)}`
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

export const defaultAudioMute = true;

export const renameFilePart = (file, newFileName) => {
  const formData = new FormData();
  const partName = "File";
  formData.append(partName, file, newFileName);
  const renamedFile = formData.get(partName);
  return renamedFile
}

export const isCalling = (status) => {
  return status == "inviting"
}

export const setTimeoutAsync = (cb, delay) =>
  new Promise((resolve) => {
    setTimeout(() => {
      resolve(cb());
    }, delay);
  });
