export const getChatApiUrl = () => {
    return process.env.CHAT_API_URL || 'http://localhost:1235'
}

export const getFrontendUrl = () => {
    return process.env.FRONTEND_URL || 'http://localhost:8081'
}

// in milliseconds, export const getFrontendUrl = () => {
//     return process.env.FRONTEND_URL || 'http://localhost:8081'
// }
export const getHttpClientTimeout = () => {
    return parseInt(process.env.CLIENT_TIMEOUT_MS || '4000')
}
