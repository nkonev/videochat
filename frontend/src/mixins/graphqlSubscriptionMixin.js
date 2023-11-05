import {graphQlClient} from "@/graphql/graphql";

// expects methods setError, onNextSubscriptionElement, getGraphQlSubscriptionQuery
export default (nameForLog) => {
    return {
        data() {
            return {
                unsubscribe: null,
                subscriptionTimeoutId: null
            }
        },
        methods: {
            graphQlSubscribe() {
                // unsubscribe from the previous
                this.graphQlUnsubscribe();

                const onNext_ = (e) => {
                    console.debug(`Got ${nameForLog} event`, e);
                    if (e.errors != null && e.errors.length) {
                        console.log("Subscription errors", e.errors);
                        this.setError(null, `Error in onNext ${nameForLog} subscription`);
                        return
                    }
                    this.onNextSubscriptionElement(e);
                }
                const onError = (e) => {
                    if (Array.isArray(e)) {
                        console.error(`Got err in ${nameForLog} subscription`, e);
                        this.setError(null, `Error in onError ${nameForLog} subscription`);
                    } else {
                        console.error(`Got connection err in ${nameForLog} subscription, reconnecting`, e);
                        this.subscriptionTimeoutId = setTimeout(this.graphQlSubscribe, 2000);
                    }
                }
                const onComplete = () => {
                    console.log(`Got compete in ${nameForLog} subscription`);
                }

                console.log(`Subscribing to ${nameForLog}`);
                this.unsubscribe = graphQlClient.subscribe(
                    {
                        query: this.getGraphQlSubscriptionQuery(),
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
                if (this.subscriptionTimeoutId) {
                    clearInterval(this.subscriptionTimeoutId);
                    this.subscriptionTimeoutId = null;
                }
                if (this.unsubscribe) {
                    this.unsubscribe();
                }
                this.unsubscribe = null;
            },
        }
    };
}
