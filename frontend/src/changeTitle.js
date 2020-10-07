export const titleFactory = (title, isShowSearch, isShowChatEditButton, chatEditId, isShowChatInfoButton) => {
    return {
        title, isShowSearch, isShowChatEditButton, chatEditId, isShowChatInfoButton
    }
}

export const phoneFactory = (show, call) => {
    return {
        show, call
    }
}