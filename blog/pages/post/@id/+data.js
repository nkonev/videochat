import axios from "axios";
import { PAGE_SIZE, getApiHost } from "#root/common/utils";

export { data };

async function data(pageContext) {
    const apiHost = getApiHost();

    const blogResponse = await axios.get(apiHost + `/api/blog/${pageContext.routeParams.id}`);

    const startingFromItemId = blogResponse.data.messageId;
    const commentResponse = await axios.get(apiHost + `/api/blog/${pageContext.routeParams.id}/comment`, {
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
        title: blogResponse.data.title
    }
}
