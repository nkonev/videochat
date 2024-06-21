import axios from "axios";
import { PAGE_SIZE, SEARCH_MODE_POSTS } from "#root/common/utils";
import { getChatApiUrl } from "#root/common/config";

export { data };

async function data(pageContext) {
    const apiHost = getChatApiUrl();
    const response = await axios.get(apiHost + '/blog', {
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
        title: "Blog",
        description: "Various tech blog"
    }
}
