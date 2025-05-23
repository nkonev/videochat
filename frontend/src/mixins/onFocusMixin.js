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
                window.addEventListener("focus", this.doOnFocus);
            },
            uninstallOnFocus() {
                this.requestAbortController.abort(); // abort requests

                window.removeEventListener("focus", this.doOnFocus);
            }
        }
    }
}
