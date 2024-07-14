import { format, parseISO, differenceInDays } from 'date-fns';
import {chat, messageIdHashPrefix} from "./router/routes.js";

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

export const setTitle = (newTitle) => {
    document.title = newTitle;
}

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
