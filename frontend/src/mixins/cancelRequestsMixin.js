export default () => {
    return {
        data() {
            return {
                requestAbortController: new AbortController(),
            }
        },
        methods: {
            installCancelRequests() {
            },
            cancelPreviousRequests() {
                this.requestAbortController.abort();

                this.requestAbortController = new AbortController()
            },
            uninstallCancelRequests() {
                this.requestAbortController.abort(); // abort requests
            },
        }
    }
}
