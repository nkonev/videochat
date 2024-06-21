export { onBeforeRender }

async function onBeforeRender(pageContext) {
    return {
        pageContext: {
            isMobile: pageContext.userAgent.indexOf('Mobile') !== -1
        }
    }
}
