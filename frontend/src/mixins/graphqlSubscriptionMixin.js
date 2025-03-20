import {graphQlClient} from "@/graphql/graphql";

// expects methods setError, onNextSubscriptionElement, getGraphQlSubscriptionQuery
export default (nameForLog, getGraphQlSubscriptionQuery, setError, onNextSubscriptionElement) => {
    const state = {};

    return {
        graphQlSubscribe() {
            // unsubscribe from the previous
            this.graphQlUnsubscribe();

            const onNext_ = (e) => {
                console.debug(`Got ${nameForLog} event`, e);
                if (e.errors != null && e.errors.length) {
                    console.log("Subscription errors", e.errors);
                    setError(null, `Error in onNext ${nameForLog} subscription`);
                    return
                }
                onNextSubscriptionElement(e);
            }
            const onError = (e) => {
                console.error(`Got err in ${nameForLog} subscription`, e);
                if (Array.isArray(e)) {
                    setError(null, `Error in onError ${nameForLog} subscription`);
                }
            }
            const onComplete = () => {
                console.log(`Got complete in ${nameForLog} subscription`);
            }

            console.log(`Subscribing to ${nameForLog}`);
            state.unsubscribe = graphQlClient.subscribe(
                {
                    query: getGraphQlSubscriptionQuery(),
                },
                {
                    next: onNext_,
                    error: onError,
                    complete: onComplete,
                },
            );
        },
        graphQlUnsubscribe() {
            console.log(`Unsubscribing from ${nameForLog}`);
            if (state.unsubscribe) {
                state.unsubscribe();
            }
            state.unsubscribe = null;
        },
    };
}
