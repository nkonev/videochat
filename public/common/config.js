export const getChatApiUrl = () => {
    return process.env.CHAT_API_URL || 'http://localhost:1235'
}

export const getFrontendUrl = () => {
    return process.env.FRONTEND_URL || 'http://localhost:8081'
}

// in millisecond
export const getHttpClientTimeout = () => {
    return parseInt(process.env.CLIENT_TIMEOUT_MS || '4000')
}

export const getPort = () => {
    return process.env.PORT || '3100'
}
