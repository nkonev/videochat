// reducer https://css-tricks.com/understanding-how-reducers-are-used-in-redux/
function reducerFunction(state = "", action) {
    switch (action.type) {
        case 'go':
            return {...state, redirectUrl: action.redirectUrl};
        case 'savePrevious':
            if (!state.previousUrl) {
                return {...state, previousUrl: action.previousUrl};
            } else {
                return state;
            }
        case 'restorePrevious':
            const pr = state.previousUrl ? state.previousUrl : "/";
            return {...state, previousUrl: null, redirectUrl: pr};
        case 'clearRedirect':
            return {...state, redirectUrl: null};
        case 'unsetProfile': {
            return {...state, profile: null};
        }
        case 'setProfile': {
            return {...state, profile: action.profile};
        }
        case 'openEditModal': {
            return {...state, editModal: action.editModal};
        }
        default:
            return state
    }
}

export default reducerFunction;
