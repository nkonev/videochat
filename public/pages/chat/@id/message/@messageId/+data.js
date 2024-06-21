import axios from "axios";
import { getChatApiUrl } from "#root/common/config";
import {getMessageLink} from "#root/common/utils.js";
import {videochat} from "#root/common/router/routes.js";

export { data };

async function data(pageContext) {
    const apiHost = getChatApiUrl();

    const chatId = pageContext.routeParams?.id;
    const messageId = pageContext.routeParams?.messageId;

    const publishedMessageResponse = await axios.get(apiHost + `/chat/public/${chatId}/message/${messageId}`);

    if (publishedMessageResponse.status == 204) {
        pageContext.httpStatus = 404;
        return {
            loaded: false,
            messageItemDto: { },
            is404: true,
            title: "Page not found",
            chatMessageHref: videochat,
        }
    }

    const chatMessageHref = getMessageLink(chatId, messageId);

    return {
        loaded: true,
        messageItemDto: publishedMessageResponse.data.message,
        is404: false,
        title: publishedMessageResponse.data.title,
        chatMessageHref,
    }

}
