import axios from "axios";
import { getChatApiUrl } from "#root/common/config";
import { PAGE_SIZE, unescapeHtml } from "#root/common/utils.js";

export { data };

async function data(pageContext) {
    const apiHost = getChatApiUrl();

    const blogResponse = await axios.get(apiHost + `/api/blog/${pageContext.routeParams.id}`);

    if (blogResponse.status == 204) {
        pageContext.httpStatus = 404;
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

    const commentResponse = await axios.get(apiHost + `/api/blog/${pageContext.routeParams.id}/comment`, {
        params: {
            page: actualPage,
            size: PAGE_SIZE,
            reverse: false,
        },
    });

    if (commentResponse.status == 204) {
        pageContext.httpStatus = 404;
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
        blogDto: blogResponse.data.post,
        header: blogResponse.data.header,
        items: commentResponse.data.items,
        // see getPageTitle.js
        title: unescapeHtml(blogResponse.data.post.title),
        description: blogResponse.data.post.preview,
        showSearchButton: true,
    }

}
