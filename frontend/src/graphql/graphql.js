import { createClient } from 'graphql-ws';
import {getWebsocketUrlPrefix} from "@/utils";
import bus, {LOGGED_OUT, WEBSOCKET_CONNECTED, WEBSOCKET_LOST, WEBSOCKET_RESTORED} from "@/bus/bus";

function createRestartableClient(options) {
    let restartRequested = false;
    let restart = () => {
        restartRequested = true;
    };

    const client = createClient({
        ...options,
        on: {
            ...options.on,
            opened: (socket) => {
                options.on?.opened?.(socket);

                restart = () => {
                    if (socket.readyState === WebSocket.OPEN) {
                        // if the socket is still open for the restart, do the restart
                        socket.close(4205, 'Client Restart');
                    } else {
                        // otherwise the socket might've closed, indicate that you want
                        // a restart on the next opened event
                        restartRequested = true;
                    }
                };

                // just in case you were eager to restart
                if (restartRequested) {
                    restartRequested = false;
                    restart();
                }
            },
        },
    });

    return {
        ...client,
        restart: () => restart(),
    };
}

let timedOut;
let graphQlClient;
export const createGraphQlClient = () => {
    let initialized = false;

    // url: getWebsocketUrlPrefix() + '/api/event/graphql',
    graphQlClient = createRestartableClient({
        url: getWebsocketUrlPrefix() + '/api/event/graphql',
        shouldRetry: () => true,
        keepAlive: 10_000, // ping server every 10 seconds
        on: {
            ping: (received) => {
                if (!received /* sent */) {
                    timedOut = setTimeout(() => {
                        // a close event `4499: Terminated` is issued to the current WebSocket and an
                        // artificial `{ code: 4499, reason: 'Terminated', wasClean: false }` close-event-like
                        // object is immediately emitted without waiting for the one coming from `WebSocket.onclose`
                        //
                        // calling terminate is not considered fatal and a connection retry will occur as expected
                        //
                        // see: https://github.com/enisdenjo/graphql-ws/discussions/290
                        graphQlClient.terminate();
                    }, 5_000);
                }
            },
            pong: (received) => {
                if (received) {
                    clearTimeout(timedOut);
                }
            },
        },
    })

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
    bus.on(LOGGED_OUT, () => {
        initialized = false;
        graphQlClient.terminate();
    });
}

export const destroyGraphqlClient = () => {
  bus.off(LOGGED_OUT);
  graphQlClient.terminate();
  graphQlClient = null;
}

export {graphQlClient};
