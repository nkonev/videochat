import {differenceInDays, format, parseISO} from "date-fns";
import {enUS} from "date-fns/locale";

export const getHumanReadableDate = (timestamp) => {
    const parsedDate = parseISO(timestamp);
    let formatString = 'HH:mm:ss';
    if (differenceInDays(new Date(), parsedDate) >= 1) {
        formatString = formatString + ', d MMM yyyy';
    }

    const localeObj = {};
    localeObj.locale = enUS;

    return `${format(parsedDate, formatString, localeObj)}`
}
