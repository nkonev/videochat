import debounce from "lodash/debounce.js";

export default () => {
    return {
        data() {
            return {
                requestAbortController: new AbortController(),
                freshAbortController: new AbortController(),
                lastUpdateDateTime: +new Date(),
            }
        },
        methods: {
            updateLastUpdateDateTime() {
                this.lastUpdateDateTime = +new Date();
            },
            doOnFocus() {
                this.$nextTick(() => {
                    if (!!this.$el && ((+new Date()) - this.lastUpdateDateTime) > (5 * 1000)) {
                        if (this.onFocus) {
                            this.updateLastUpdateDateTime();
                            this.onFocus();
                        }
                    }
                })
            },
            cancelFreshDebounced() {
                this.freshAbortController.abort();
            },
            installOnFocus() {
                this.doOnFocus = debounce(this.doOnFocus, 300, {leading: false, trailing: true});
                window.addEventListener("focus", this.doOnFocus);

                this.cancelFreshDebounced = debounce(this.cancelFreshDebounced, 700, {leading:true, trailing:false})
            },
            uninstallOnFocus() {
                this.requestAbortController.abort(); // abort requests

                this.doOnFocus.cancel(); // cancel the debounced function in order tot to execute it with the disposed resources

                window.removeEventListener("focus", this.doOnFocus);
            },
        }
    }
}
