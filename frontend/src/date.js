import {getStoredLanguage} from "@/store/localStore.js";
import {differenceInDays, format, parseISO} from "date-fns";
import {ru, enUS} from "date-fns/locale";

export const getHumanReadableDate = (timestamp) => {
    const parsedDate = parseISO(timestamp);
    let formatString = 'HH:mm:ss';
    if (differenceInDays(new Date(), parsedDate) >= 1) {
        formatString = formatString + ', d MMM yyyy';
    }

    const lang = getStoredLanguage();
    const localeObj = {};
    switch (lang) {
        case 'ru':
            localeObj.locale = ru;
            break
        case 'en':
        default:
            localeObj.locale = enUS;
            break
    }

    return `${format(parsedDate, formatString, localeObj)}`
}
