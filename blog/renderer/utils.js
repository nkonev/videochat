import { format, parseISO, differenceInDays } from 'date-fns';

export const isMobileBrowser = () => {
    return navigator.userAgent.indexOf('Mobile') !== -1
}


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

export const getApiHost = () => {
    return process.env.API_HOST || 'http://localhost:8081'
}
