export const titleFactory = (title, isShowSearch, isShowChatEditButton, chatEditId, isShowChatInfoButton, chatId) => {
    return {
        title, isShowSearch, isShowChatEditButton, chatEditId, isShowChatInfoButton, chatId
    }
}

export const phoneFactory = (show, call) => {
    return {
        show, call
    }
}