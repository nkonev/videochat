import {graphQlClient} from "@/graphql/graphql";
import {hasLength} from "@/utils";

export default (nameForLog) => {
    return {
        data() {
            return {
                subscriptionElements: [],
            }
        },
        methods: {
            getUserName(item) {
                let bldr = item.login;
                if (!hasLength(item.avatar) && item.online) {
                    bldr += " (" + this.$vuetify.locale.t('$vuetify.user_online') + ")";
                }
                return bldr;
            },
            transformItem(item) {
                item.online = false;
                item.isInVideo = false;
            },
            getColor(item) {
                return item.isInVideo ? 'red accent-4' : 'success accent-4'
            },
            getUserOnlineSubscriptionQuery() {
                const userIds = this.getUserIdsSubscribeTo();
                return `
                    subscription {
                        userOnlineEvents(userIds:[${userIds}]) {
                            id
                            online
                        }
                    }`
            },
            getUserVideoStatusSubscriptionQuery() {
                const userIds = this.getUserIdsSubscribeTo();
                return `
                    subscription {
                        userVideoStatusEvents(userIds:[${userIds}]) {
                            userId
                            isInVideo
                        }
                    }`
            },

            graphQlSubscribe() {
                // unsubscribe from the previous for case re-subscribing on user list change
                this.graphQlUnsubscribe();

                const subscriptionElement1 = { name: 'userOnline ' + nameForLog };
                this.performSubscription(subscriptionElement1, this.getUserOnlineSubscriptionQuery, this.onUserOnlineChanged)
                this.subscriptionElements.push(subscriptionElement1);

                const subscriptionElement2 = { name: 'userVideoStatus ' + nameForLog };
                this.performSubscription(subscriptionElement2, this.getUserVideoStatusSubscriptionQuery, this.onUserVideoStatusChanged)
                this.subscriptionElements.push(subscriptionElement2);
            },
            performSubscription(subscriptionElement, getGraphQlSubscriptionQuery, handler) {
                // unsubscribe from the previous for case restart
                this.doUnsubscribe(subscriptionElement);

                const onNext_ = (e) => {
                    console.debug(`Got ${subscriptionElement.name} event`, e);
                    if (e.errors != null && e.errors.length) {
                        console.log("Subscription errors", e.errors);
                        this.setError(null, `Error in onNext ${subscriptionElement.name} subscription`);
                        return
                    }
                    handler(e);
                }
                const onError_ = (e) => {
                    if (Array.isArray(e)) {
                        console.error(`Got err in ${subscriptionElement.name} subscription`, e);
                        this.setError(null, `Error in onError ${subscriptionElement.name} subscription`);
                    } else {
                        console.error(`Got connection err in ${subscriptionElement.name} subscription, reconnecting`, e);
                        subscriptionElement.timeout = setTimeout(() => this.performSubscription(subscriptionElement, getGraphQlSubscriptionQuery, handler), 2000);
                    }
                }
                const onComplete_ = () => {
                    console.log(`Got compete in ${subscriptionElement.name} subscription`);
                }

                console.log(`Subscribing to ${subscriptionElement.name}`);
                subscriptionElement.unsubscribe = graphQlClient.subscribe(
                    {
                        query: getGraphQlSubscriptionQuery(),
                    },
                    {
                        next: onNext_,
                        error: onError_,
                        complete: onComplete_,
                    },
                );
            },
            doUnsubscribe(subscriptionElement) {
                console.log(`Unsubscribing from ${subscriptionElement.name}`);

                if (subscriptionElement.unsubscribe) {
                    subscriptionElement.unsubscribe();
                    subscriptionElement.unsubscribe = null;
                }
                if (subscriptionElement.timeout) {
                    clearInterval(subscriptionElement.timeout);
                    subscriptionElement.timeout = null;
                }
            },
            graphQlUnsubscribe() {
                console.log(`Unsubscribing from all subscriptions`);
                for (const subscriptionElement of this.subscriptionElements) {
                    this.doUnsubscribe(subscriptionElement);
                }
                this.subscriptionElements = [];
            }
        }
    };
}
