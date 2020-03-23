import React from 'react';
import ReactDOM from 'react-dom';
import './index.css';
import App from './App';
import 'typeface-roboto';
import axios from 'axios'

axios.interceptors.response.use((response) => {
    return response
}, (error) => {
    // https://github.com/axios/axios/issues/932#issuecomment-307390761
    if (error && error.response && error.response.status == 401) {
        console.log("Error http", error, error.response);
        window.location = "/login";
    } else {
        return Promise.reject(error)
    }
});

ReactDOM.render(<App />, document.getElementById('root'));
