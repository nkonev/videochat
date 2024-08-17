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

    const searchString = pageContext.urlParsed.search[SEARCH_MODE_POSTS];

    const response = await axios.get(apiHost + '/blog', {
        params: {
            page: actualPage,
            size: PAGE_SIZE,
            reverse: false,
            searchString: searchString,
        },
    });

    const pagesCount = response.data.pagesCount;
    const count = response.data.count;

    return {
        page,
        pagesCount,
        count,
        items: response.data.items,
        showSearchButton: true,
        searchStringFacade: searchString,
        markInstance: null,
        title: "Blog",
        description: "Various tech blog",
        loading: false,
    }
}
