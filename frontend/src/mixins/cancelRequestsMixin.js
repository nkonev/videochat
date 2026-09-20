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
            restartCancelRequests() {
                this.requestAbortController = new AbortController()
            },
            uninstallCancelRequests() {
                this.requestAbortController.abort(); // abort requests
            },
        }
    }
}
