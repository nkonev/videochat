import axios from "axios";
import { PAGE_SIZE } from "#root/common/utils";
import { getApiHost } from "#root/common/config";

export { data };

async function data(pageContext) {
    const apiHost = getApiHost();

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
}
