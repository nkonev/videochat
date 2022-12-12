import { createClient } from 'graphql-ws';
import {getWebsocketUrlPrefix} from "@/utils";
import {WEBSOCKET_RESTORED} from "@/bus";

let graphQlClient;
export const createGraphQlClient = (bus) => {
    let initialized = false;
    // https://github.com/enisdenjo/graphql-ws#use-the-client
    graphQlClient = createClient({
        url: getWebsocketUrlPrefix() + '/api/event/graphql',
    });
    graphQlClient.on('connected', () => {
        if (initialized) {
            console.log("ReConnected to websocket graphql");
            bus.$emit(WEBSOCKET_RESTORED);
        }
        initialized = true;
    })
}
export {graphQlClient};
