import mitt from 'mitt'

const emitter = mitt()

export default emitter


export const LOGGED_OUT = "loggedOut";
export const PROFILE_SET = "profileSetNotNull";
export const LOGGED_IN = "loggedIn";

export const SEARCH_STRING_CHANGED = "searchStringChanged";
