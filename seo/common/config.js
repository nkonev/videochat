export const getApiHost = () => {
    return process.env.API_HOST || 'http://localhost:1235'
}

export const getSeoHost = () => {
    return process.env.SEO_HOST || 'http://localhost:8081'
}
