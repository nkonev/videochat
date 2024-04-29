import axios from "axios";
import { PAGE_SIZE, getApiHost, SEARCH_MODE_POSTS } from "#root/renderer/utils";
import {directionBottom} from "../../renderer/mixins/infiniteScrollMixin.js";

export { data };

function getMaximumItemId(items) {
    return items.length ? Math.max(...items.map(it => it.id)) : null
}
function getMinimumItemId(items) {
    return items.length ? Math.min(...items.map(it => it.id)) : null
}

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

    // updateTopAndBottomIds()
    const startingFromItemIdTop = getMaximumItemId(items);
    const startingFromItemIdBottom = getMinimumItemId(items);

    return {
        items,
        markInstance: null,
        startingFromItemIdTop,
        startingFromItemIdBottom,

        // an effect like one after resetInfiniteScrollVars(), load()
        isFirstLoad: false,
        loadedBottom: true,
        loadedTop: true,
        aDirection: directionBottom,
        scrollerProbePrevious: 0,
        scrollerProbeCurrent: 0,
        preservedScroll: null,
    }
}
