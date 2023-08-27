import { format, parseISO, differenceInDays } from 'date-fns';
import {blog} from "@/router/routes";

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

export const setIcon = (newMessages) => {
  var link = document.querySelector("link[rel~='icon']");
  if (!link) {
    link = document.createElement('link');
    link.rel = 'icon';
    document.getElementsByTagName('head')[0].appendChild(link);
  }
  if (newMessages) {
    link.href = '/front2/favicon_new2.svg';
  } else {
    link.href = '/front2/favicon2.svg';
  }
}

export const deepCopy = (aVal) => {
    return JSON.parse(JSON.stringify(aVal))
}

export const embed_message_reply = "reply";
export const embed_message_resend = "resend";

export const getBlogLink = (chatId) => {
    return blog + '/post/' + chatId;
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
