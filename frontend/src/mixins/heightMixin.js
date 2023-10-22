
export default () => {
    return {
        computed: {
            heightWithoutAppBar() {
                if (this.isMobile()) {
                    return 'height: calc(100dvh - 56px)'
                } else {
                    return 'height: calc(100dvh - 48px)'
                }
            },
        }
    }
}
