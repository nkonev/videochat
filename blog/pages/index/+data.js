import axios from "axios";
import { PAGE_SIZE, getApiHost, SEARCH_MODE_POSTS } from "#root/common/utils";

export { data };

async function data(pageContext) {
    const apiHost = getApiHost();
    const response = await axios.get(apiHost + '/api/blog', {
        params: {
            size: PAGE_SIZE,
            reverse: false,
            searchString: pageContext.urlParsed.search[SEARCH_MODE_POSTS],
            hasHash: false,
        },
    });

    const items = response.data

    return {
        items,
    }
}
