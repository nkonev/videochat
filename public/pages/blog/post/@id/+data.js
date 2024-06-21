import axios from "axios";
import { PAGE_SIZE } from "#root/common/utils";
import { getChatApiUrl } from "#root/common/config";

export { data };

async function data(pageContext) {
    const apiHost = getChatApiUrl();

    const blogResponse = await axios.get(apiHost + `/blog/${pageContext.routeParams.id}`);

    if (blogResponse.status == 204) {
        return {
            blogDto: {
                is404: true
            },
            items: [],
            title: "Page not found",
        }
    }

    const startingFromItemId = blogResponse.data.messageId;
    const commentResponse = await axios.get(apiHost + `/blog/${pageContext.routeParams.id}/comment`, {
        params: {
            startingFromItemId: startingFromItemId,
            size: PAGE_SIZE,
            reverse: false,
        },
    });

    if (commentResponse.status == 204) {
        return {
            blogDto: {
                is404: true
            },
            items: [],
            title: "Page not found",
        }
    }

    return {
        blogDto: blogResponse.data,
        items: commentResponse.data,
        // see getPageTitle.js
        title: blogResponse.data.title,
        description: blogResponse.data.preview,
    }

}
