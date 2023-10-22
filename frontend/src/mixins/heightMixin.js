
export default () => {
    return {
        computed: {
            heightWithoutAppBar() {
                if (this.isMobile()) {
                    return 'height: calc(100dvh - 56px)' // see also $mobileAppBarHeight in constants.styl
                } else {
                    return 'height: calc(100dvh - 48px)'
                }
            },
        }
    }
}
