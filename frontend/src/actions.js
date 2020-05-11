export const goLogin = () => ({
    type: 'go',
    redirectUrl: "/login"
});

export const savePreviousUrl = (url) => ({
    type: 'savePrevious',
    previousUrl: url
});