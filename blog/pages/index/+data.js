import axios from "axios";
import { PAGE_SIZE, getApiHost, SEARCH_MODE_POSTS } from "#root/renderer/utils";

export { data };

async function data(pageContext) {
    const apiHost = getApiHost();
    const response = await axios.get(apiHost + '/api/blog', {
        params: {
            size: PAGE_SIZE,
            reverse: false,
            // TODO if set pageContext.urlParsed.search[SEARCH_MODE_POSTS] - then it leads us to parasite download
            searchString: "",
            hasHash: false,
        },
    });

    return {
        items: response.data,
        markInstance: null,
    }
}
