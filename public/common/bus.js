import mitt from 'mitt'

const emitter = mitt()

export default emitter

export const SEARCH_STRING_CHANGED = "searchStringChanged";

export const PLAYER_MODAL = "playerModal";
