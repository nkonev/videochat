export const titleFactory = (title, isShowSearch, isShowChatEditButton, chatEditId) => {
    return {
        title, isShowSearch, isShowChatEditButton, chatEditId
    }
}

export const phoneFactory = (show, call) => {
    return {
        show, call
    }
}