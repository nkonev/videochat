export const titleFactory = (title, isShowSearch, isShowChatEditButton, chatEditId, chatId) => {
    return {
        title, isShowSearch, isShowChatEditButton, chatEditId, chatId
    }
}

export const phoneFactory = (show, call) => {
    return {
        show, call
    }
}