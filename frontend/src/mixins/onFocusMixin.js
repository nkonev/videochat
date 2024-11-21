import debounce from "lodash/debounce.js";

export default () => {
    return {
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
                this.doOnFocus.cancel(); // cancel the debounced function in order tot to execute it with the disposed resources

                window.removeEventListener("focus", this.doOnFocus);
            }
        }
    }
}