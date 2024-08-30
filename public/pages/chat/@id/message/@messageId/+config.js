export default {
    passToClient: [ // props in pageContext
        'isMobile',
        'urlParsed' // because clientRouting is't set and PageShell.vue requires it
    ]
}
