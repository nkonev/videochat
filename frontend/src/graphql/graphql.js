import { createClient } from 'graphql-ws';
import {getWebsocketUrlPrefix} from "@/utils";
import bus, {LOGGED_OUT, WEBSOCKET_CONNECTED, WEBSOCKET_LOST, WEBSOCKET_RESTORED} from "@/bus/bus";

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
        } else {
            console.info("Connected to websocket graphql");
        }
        initialized = true;
        bus.emit(WEBSOCKET_CONNECTED);
    });
    graphQlClient.on('error', (err) => {
        console.info("Error in GraphQL client", err);
    });
    graphQlClient.on('closed', (ev) => {
        console.info("Close GraphQL", ev);
        bus.emit(WEBSOCKET_LOST);
    });
    bus.on(LOGGED_OUT, () => {initialized = false});
}

export const destroyGraphqlClient = () => {
  bus.off(LOGGED_OUT);
  graphQlClient.terminate();
  graphQlClient = null;
}

export {graphQlClient};
