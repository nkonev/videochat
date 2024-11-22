import debounce from "lodash/debounce.js";

export default () => {
    return {
        data() {
            return {
                requestAbortController: new AbortController(),
            }
        },
        methods: {
            doOnFocus() {
                this.$nextTick(() => {
                    if (!!this.$el) {
                        if (this.onFocus) {
                            this.onFocus();
                        }
                    }
                })
            },
            installOnFocus() {
                this.doOnFocus = debounce(this.doOnFocus, 200, {leading: false, trailing: true});
                window.addEventListener("focus", this.doOnFocus);
            },
            uninstallOnFocus() {
                this.requestAbortController.abort(); // abort requests

                this.doOnFocus.cancel(); // cancel the debounced function in order tot to execute it with the disposed resources

                window.removeEventListener("focus", this.doOnFocus);
            }
        }
    }
}