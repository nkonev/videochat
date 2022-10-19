import { createClient } from 'graphql-ws';
import {getWebsocketUrlPrefix} from "@/utils";

// https://github.com/enisdenjo/graphql-ws#use-the-client
const graphQlClient = createClient({
    url: getWebsocketUrlPrefix() + '/event/graphql',
});
export default graphQlClient;
/*
// subscription
(async () => {
    const onNext = () => {
        /* handle incoming values /
    };

    let unsubscribe = () => {
        /* complete the subscription /
    };

    await new Promise((resolve, reject) => {
        unsubscribe = graphQlClient.subscribe(
            {
                query: `
                    subscription {
                      globalEvents {
                        eventType
                        chatEvent {
                          id
                          name
                          avatar
                          avatarBig
                          participantIds
                          participants {
                            id
                            login
                            avatar
                            admin
                          }
                        }
                      }
                    }
                `,
            },
            {
                next: onNext,
                error: reject,
                complete: resolve,
            },
        );
    });
})();
*/