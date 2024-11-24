import axios from "axios";
import { getChatApiUrl } from "#root/common/config";
import { getMessageLink, unescapeHtml } from "#root/common/utils.js";

export { data };

async function data(pageContext) {
    const apiHost = getChatApiUrl();

    const chatId = pageContext.routeParams?.id;
    const messageId = pageContext.routeParams?.messageId;

    const publishedMessageResponse = await axios.get(apiHost + `/api/chat/public/${chatId}/message/${messageId}`);

    if (publishedMessageResponse.status == 204) {
        pageContext.httpStatus = 404;
        return {
            messageDto: {
                is404: true,
                messageItem: { },
            },
            title: "Page not found",
            chatMessageHref: null,
        }
    }

    const chatMessageHref = getMessageLink(chatId, messageId);

    const title = unescapeHtml(publishedMessageResponse.data.title);
    return {
        messageDto: {
            messageItem: publishedMessageResponse.data.message,
            is404: false,
        },
        chatTitle: title,
        title: title,
        description: publishedMessageResponse.data.preview,
        showSearchButton: true,
        chatMessageHref,
    }

}
