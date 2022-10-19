import { createClient } from 'graphql-ws';
import {getWebsocketUrlPrefix} from "@/utils";

// https://github.com/enisdenjo/graphql-ws#use-the-client
const graphQlClient = createClient({
    url: getWebsocketUrlPrefix() + '/event/graphql',
});
export default graphQlClient;
