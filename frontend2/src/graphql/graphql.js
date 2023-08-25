import { createClient } from 'graphql-ws';
import {getWebsocketUrlPrefix} from "@/utils";
import bus, {LOGGED_OUT, WEBSOCKET_RESTORED} from "@/bus/bus";

let graphQlClient;
export const createGraphQlClient = () => {
    let initialized = false;
    // https://github.com/enisdenjo/graphql-ws#use-the-client
    graphQlClient = createClient({
        url: getWebsocketUrlPrefix() + '/api/event/graphql',
    });
    graphQlClient.on('connected', () => {
        if (initialized) {
            console.log("ReConnected to websocket graphql");
            bus.emit(WEBSOCKET_RESTORED);
        }
        initialized = true;
    });
    bus.on(LOGGED_OUT, () => {initialized = false});
}

export const destroyGraphqlClient = () => {
  bus.off(LOGGED_OUT);
  graphQlClient.terminate();
  graphQlClient = null;
}

export {graphQlClient};
