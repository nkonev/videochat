import axios from "axios";
import { PAGE_SIZE, SEARCH_MODE_POSTS } from "#root/common/utils";
import { getChatApiUrl } from "#root/common/config";

export { data };

async function data(pageContext) {
    const apiHost = getChatApiUrl();
    let page = pageContext.urlParsed.search.page;

    let actualPage = undefined;
    if (page) {
        page = parseInt(page);
        actualPage = page - 1;
    }
    const response = await axios.get(apiHost + '/blog', {
        params: {
            page: actualPage,
            size: PAGE_SIZE,
            reverse: false,
            searchString: pageContext.urlParsed.search[SEARCH_MODE_POSTS],
        },
    });

    const pagesCount = response.data.count / PAGE_SIZE;

    return {
        page,
        pagesCount,
        items: response.data.items,
        showSearchButton: true,
        title: "Blog",
        description: "Various tech blog"
    }
}
