import axios from "axios";
import { PAGE_SIZE } from "#root/renderer/utils";

export { data };

async function data(pageContext) {
    const response = await axios.get(`http://localhost:8081/api/blog`, { // TODO make host configurable
        params: {
            size: PAGE_SIZE,
            reverse: false,
            searchString: null,
            hasHash: false,
        },
    });

    return {
        items: response.data,
        markInstance: null,
    }
}
