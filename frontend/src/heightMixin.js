
export default () => {
    return {
        computed: {
            heightWithoutAppBar() {
                if (this.isMobile()) {
                    return 'height: calc(var(--100vvh, 100vh) - 56px)'
                } else {
                    return 'height: calc(var(--100vvh, 100vh) - 48px)'
                }
            },
        }
    }
}
