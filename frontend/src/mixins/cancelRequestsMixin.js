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
            uninstallCancelRequests() {
                this.requestAbortController.abort(); // abort requests
            },
        }
    }
}
