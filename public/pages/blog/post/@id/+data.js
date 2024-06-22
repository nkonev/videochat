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

    let page = pageContext.urlParsed.search.page;

    let actualPage = undefined;
    if (page) {
        page = parseInt(page);
        actualPage = page - 1;
    }

    const commentResponse = await axios.get(apiHost + `/blog/${pageContext.routeParams.id}/comment`, {
        params: {
            page: actualPage,
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

    const pagesCount = commentResponse.data.pagesCount;
    const count = commentResponse.data.count;

    return {
        page,
        pagesCount,
        count,
        blogDto: blogResponse.data,
        items: commentResponse.data.items,
        // see getPageTitle.js
        title: blogResponse.data.title,
        description: blogResponse.data.preview,
    }

}
