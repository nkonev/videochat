import { format, parseISO, differenceInDays } from 'date-fns';
import {blog_post, chat, prefix, video_suffix} from "@/router/routes";

export const isMobileBrowser = () => {
    return navigator.userAgent.indexOf('Mobile') !== -1
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

export const offerToJoinToPublicChatStatus = 428

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
    link.href = `${prefix}/favicon_new2.svg`;
  } else {
    link.href = `${prefix}/favicon2.svg`;
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

export const getAnswerPreviewFields = (dto) => {
  return dto;
}

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
  dto.embedPreviewText = messageText;
  dto.embedPreviewOwner = ownerLogin;
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

export const publicallyAvailableForSearchChatsQuery = "__AVAILABLE_FOR_SEARCH";

export const isArrEqual = (a, b) => {
    if (a == null && b == null) {
        return true
    }
    if (a == null && b != null) {
        return false
    }
    if (a != null && b == null) {
        return false
    }
    if (a != null && b != null) {
        return JSON.stringify(a.sort()) === JSON.stringify(b.sort());
    }
    console.error("Unexpected branch", a, b);
    return true
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
