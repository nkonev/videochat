import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import App from './App';
import 'typeface-roboto';
import axios from 'axios'
import { createStore } from 'redux'
import { Provider } from 'react-redux'
import { goLogin, savePreviousUrl } from "./actions"

function storeFunction(state = "", action) {
    switch (action.type) {
        case 'go':
            return {...state, redirectUrl: action.redirectUrl};
        case 'savePrevious':
            return {...state, previousUrl: action.previousUrl};
        case 'restorePrevious':
            const pr = state.previousUrl;
            return {...state, previousUrl: null, redirectUrl: pr};
        case 'clearRedirect':
            return {...state, redirectUrl: null};
        default:
            return state
    }
}

const store = createStore(storeFunction);
store.subscribe(() => console.log("state changed", store.getState()));

axios.interceptors.response.use((response) => {
    return response
}, (error) => {
    // https://github.com/axios/axios/issues/932#issuecomment-307390761
    console.log("Catch error", error.request);
    if (error && error.response && error.response.status == 401) {
        console.log("Catch 401 Unauthorized, saving url", window.location.pathname);
        store.dispatch(savePreviousUrl(window.location.pathname));
        store.dispatch(goLogin());
        return Promise.reject(error)
    } else {
        return Promise.reject(error)
    }
});

ReactDOM.render(
    <Provider store={store}>
        <App />
    </Provider>,
    document.getElementById('root')
);
