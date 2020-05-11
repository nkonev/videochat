export const restorePreviousUrl = () => ({
    type: 'restorePrevious'
});

export const goLogin = () => ({
    type: 'go',
    redirectUrl: "/login"
});

export const savePreviousUrl = (url) => ({
    type: 'savePrevious',
    previousUrl: url
});

export const clearRedirect = () => ({
    type: 'clearRedirect'
});