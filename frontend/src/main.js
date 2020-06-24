import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import App from './App';
import 'typeface-roboto';
import axios from 'axios'
import { createStore } from 'redux'
import { Provider } from 'react-redux'
import {goLogin, savePreviousUrl, unsetProfile} from "./actions"
import {getProfile, setupCentrifuge} from "./utils";
import reducerFunction from "./reducer"

const store = createStore(reducerFunction);
store.subscribe(() => console.log("state changed", store.getState()));

axios.interceptors.response.use((response) => {
    return response
}, (error) => {
    // https://github.com/axios/axios/issues/932#issuecomment-307390761
    // console.log("Catch error", error, error.request, error.response, error.config);
    if (error && error.response && error.response.status == 401 && error.config.url != '/api/profile') {
        console.log("Catch 401 Unauthorized, saving url", window.location.pathname);
        store.dispatch(unsetProfile());
        store.dispatch(savePreviousUrl(window.location.pathname));
        store.dispatch(goLogin());
        return Promise.reject(error)
    } else {
        return Promise.reject(error)
    }
});

const centrifuge = setupCentrifuge();

getProfile(store.dispatch).finally(()=>{
    ReactDOM.render(
        <Provider store={store}>
            <App centrifuge={centrifuge} />
        </Provider>,
        document.getElementById('root')
    );
});

window.onunload = () => {
    centrifuge.disconnect();
};