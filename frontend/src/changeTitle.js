export const titleFactory = (title, isShowSearch, isShowChatEditButton, chatEditId, chatId, chatUsersCount) => {
    return {
        title, isShowSearch, isShowChatEditButton, chatEditId, chatId, chatUsersCount
    }
}

export const phoneFactory = (show, call) => {
    return {
        show, call
    }
}