import { createClient } from 'graphql-ws';
import {getWebsocketUrlPrefix} from "@/utils";
import bus, {
    LOGGED_OUT,
    WEBSOCKET_CONNECTED, WEBSOCKET_CONNECTING,
    WEBSOCKET_INITIALIZED,
    WEBSOCKET_LOST,
    WEBSOCKET_RESTORED, WEBSOCKET_UNINITIALIZED
} from "@/bus/bus";

// This is an adaptation of "https://the-guild.dev/graphql/ws/recipes#client-usage-with-abrupt-termination-on-pong-timeout" recipe
// see also https://github.com/enisdenjo/graphql-ws/discussions/290

// Testcase:
// 1. Pause event app
// 2. wait roughly 20 sec
// 3. restart event app
// 4. it should reconnect all the subscriptions

let pingTimedOut;

const connectionAckWaitTimeout = 5_000;
const pingSendInterval = 5_000;
const pingReceiveTimeout = 10_000;
const retryDelay = 1000;
const maxAttempts = Number.MAX_SAFE_INTEGER;

let graphQlClient;
export const createGraphQlClient = () => {
    let initialized = false;

    graphQlClient = createClient({
        url: getWebsocketUrlPrefix() + '/api/event/graphql',
        shouldRetry: () => true,
        connectionAckWaitTimeout: connectionAckWaitTimeout,
        keepAlive: pingSendInterval, // ping server every N seconds
        retryAttempts: maxAttempts,
        retryWait: async function randomised(retries) {
            console.log("Attempt to connect to graphql websocket", retries, "of", maxAttempts);
            await new Promise((resolve) =>
                setTimeout(
                    resolve,
                    retryDelay,
                ),
            );
        },
        on: {
            ping: (received) => {
                if (!received /* sent */) {
                    pingTimedOut = setTimeout(() => {
                        // a close event `4499: Terminated` is issued to the current WebSocket and an
                        // artificial `{ code: 4499, reason: 'Terminated', wasClean: false }` close-event-like
                        // object is immediately emitted without waiting for the one coming from `WebSocket.onclose`
                        //
                        // calling terminate is not considered fatal and a connection retry will occur as expected
                        //
                        // see: https://github.com/enisdenjo/graphql-ws/discussions/290
                        graphQlClient.terminate();
                    }, pingReceiveTimeout); // wait 2 * N seconds for the pong and then close the connection
                }
            },
            pong: (received) => {
                if (received) {
                    clearTimeout(pingTimedOut); // pong is received, clear connection close timeout
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
            bus.emit(WEBSOCKET_INITIALIZED);
        }
        initialized = true;
        bus.emit(WEBSOCKET_CONNECTED);
    });
    graphQlClient.on('connecting', (err) => {
        console.info("Connecting to websocket graphql");
        bus.emit(WEBSOCKET_CONNECTING);
    });
    graphQlClient.on('error', (err) => {
        console.info("Error in GraphQL client", err);
        bus.emit(WEBSOCKET_LOST);
    });
    graphQlClient.on('closed', (ev) => {
        console.info("Close GraphQL", ev);
    });
    bus.on(LOGGED_OUT, () => {
        bus.emit(WEBSOCKET_UNINITIALIZED);
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
