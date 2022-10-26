import graphQlClient from "@/graphql";

// expects methods setError, onNextSubscriptionElement, getGraphQlSubscriptionQuery, and additionalActionAfterGraphQlSubscription
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
                        this.setError(null, `Error in ${nameForLog} subscription`);
                        return
                    }
                    this.onNextSubscriptionElement(e);
                }
                const onError = (e) => {
                    console.error(`Got err in ${nameForLog} subscription, reconnecting`, e);
                    this.subscriptionTimeoutId = setTimeout(this.graphQlSubscribe, 2000);
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

                if (this.additionalActionAfterGraphQlSubscription) {
                    this.additionalActionAfterGraphQlSubscription();
                }
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