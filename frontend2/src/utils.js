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
