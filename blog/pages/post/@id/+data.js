import axios from "axios";
import { PAGE_SIZE, getApiHost, SEARCH_MODE_POSTS } from "#root/common/utils";

export { data };

async function data(pageContext) {
    const apiHost = getApiHost();

    const blogResponse = await axios.get(apiHost + `/api/blog/${pageContext.routeParams.id}`);

    const commentResponse = await axios.get(apiHost + `/api/blog/${pageContext.routeParams.id}/comment`, {
        params: {
            size: PAGE_SIZE,
            reverse: false,
            searchString: pageContext.urlParsed.search[SEARCH_MODE_POSTS],
            hasHash: false,
        },
    });

    return {
        blogDto: blogResponse.data,
        items: commentResponse.data,
    }
}
