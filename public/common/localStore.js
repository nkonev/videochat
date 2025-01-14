import { isMobileBrowser } from "./utils.js"

export const KEY_FILE_LIST_MODE = 'fileListMode';

export const getStoredFileListMode = () => {
    let v = JSON.parse(localStorage.getItem(KEY_FILE_LIST_MODE));
    if (v === null) {
        console.log("Resetting fileListMode to default");
        const defaultFileListMode = !isMobileBrowser();
        setStoredFileListMode(defaultFileListMode);
        v = JSON.parse(localStorage.getItem(KEY_FILE_LIST_MODE));
    }
    return v;
}

export const setStoredFileListMode = (v) => {
    localStorage.setItem(KEY_FILE_LIST_MODE, JSON.stringify(v));
}
