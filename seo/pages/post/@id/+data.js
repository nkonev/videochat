import axios from "axios";
import { PAGE_SIZE } from "#root/common/utils";
import { getChatApiUrl } from "#root/common/config";

export { data };

async function data(pageContext) {
    const apiHost = getChatApiUrl();

    try {
        const blogResponse = await axios.get(apiHost + `/blog/${pageContext.routeParams.id}`);

        const startingFromItemId = blogResponse.data.messageId;
        const commentResponse = await axios.get(apiHost + `/blog/${pageContext.routeParams.id}/comment`, {
            params: {
                startingFromItemId: startingFromItemId,
                size: PAGE_SIZE,
                reverse: false,
            },
        });

        return {
            blogDto: blogResponse.data,
            items: commentResponse.data,
            // see getPageTitle.js
            title: blogResponse.data.title,
            description: blogResponse.data.preview,
        }
    } catch (e) {
        if (JSON.parse(JSON.stringify(e)).status == 404) {
            pageContext.httpStatus = 404;
            return {
                blogDto: {
                    is404: true
                },
                items: [],
                title: "Page not found",
            }
        } else {
            throw e
        }
    }

}
