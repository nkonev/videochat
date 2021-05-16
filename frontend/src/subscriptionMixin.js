const checkTimeoutStep = 5000;

export default () => {
    return  {
        data () {
            return {
                ttl: 0,
                startTime: 0,
                timerId: null
            }
        },
        methods:{
            getCurrentTimeInSeconds() {
                return new Date().getTime() / 1000;
            },
            processSubscriptionResponse(value) {
                this.ttl = value.data.ttl;
                this.startTime = this.getCurrentTimeInSeconds();
            },
            initSubscription(resubscribeCallback) {
                this.timerId = setInterval(()=>{
                    if (this.startTime + this.ttl - this.getCurrentTimeInSeconds() < checkTimeoutStep / 1000) {
                        resubscribeCallback()
                    }
                }, checkTimeoutStep);

            },
            closeSubscription() {
                clearInterval(this.timerId);
            }
        },

    }
}