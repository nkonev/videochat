import Vue from 'vue'
import App from './App.vue'
import vuetify from '@/plugins/vuetify'
import axios from "axios";
import bus, {UNAUTHORIZED} from './bus';
import store, {SET_PREVIOUS_URL, UNSET_USER, FETCH_USER_PROFILE} from './store'

const vm = new Vue({
  vuetify,
  store,
  // https://ru.vuejs.org/v2/guide/render-function.html
  render: h => h(App, {ref: 'appRef'})
}).$mount('#root');

axios.interceptors.response.use((response) => {
  return response
}, (error) => {
  // https://github.com/axios/axios/issues/932#issuecomment-307390761
  // console.log("Catch error", error, error.request, error.response, error.config);
  if (error && error.response && error.response.status == 401 && error.config.url != '/api/profile') {
    console.log("Catch 401 Unauthorized, saving url", window.location.pathname);
    store.commit(UNSET_USER);
    store.commit(SET_PREVIOUS_URL, window.location.pathname);
    bus.$emit(UNAUTHORIZED, null);
    return Promise.reject(error)
  } else {
    console.log(error.response);
    vm.$refs.appRef.onError(error.response);
    return Promise.reject(error)
  }
});
