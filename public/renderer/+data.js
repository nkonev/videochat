import {getMessageLink} from "../common/utils.js";

export { data };

async function data(pageContext) {

    const chatId = pageContext.routeParams?.id;
    const messageId = pageContext.routeParams?.messageId;

    const goToChatMessageHref = getMessageLink(chatId, messageId);

    console.warn('>>>>', goToChatMessageHref)

    return {
        goToChatMessageHref
    }
}
