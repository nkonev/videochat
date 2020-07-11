import Vue from 'vue'
import App from './App.vue'
import vuetify from '@/plugins/vuetify'
import {setupCentrifuge} from "@/centrifugeConnection"
import axios from "axios";
import bus, {CHAT_ADD, CHAT_DELETED, CHAT_EDITED, UNAUTHORIZED} from './bus';
import store, {UNSET_USER} from './store'
import router from './router.js'
import {getData, getProperData} from "./centrifugeConnection";

const vm = new Vue({
  vuetify,
  store,
  router,
  created(){
    const setCetrifugeSession = (cs) => {
      Vue.prototype.centrifugeSessionId = cs;
    };
    Vue.prototype.centrifuge = setupCentrifuge(setCetrifugeSession);
  },
  destroyed() {
    Vue.prototype.centrifuge.disconnect();
  },
  mounted(){
    this.centrifuge.on('publish', (ctx)=>{
      console.log("Got personal message", ctx);
      if (getData(ctx).type === 'chat_created') {
        const d = getProperData(ctx);
        bus.$emit(CHAT_ADD, d);
      }
      if (getData(ctx).type === 'chat_edited') {
        const d = getProperData(ctx);
        bus.$emit(CHAT_EDITED, d);
      }
      if (getData(ctx).type === 'chat_deleted') {
        const d = getProperData(ctx);
        bus.$emit(CHAT_DELETED, d);
      }
    });
  },
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
    bus.$emit(UNAUTHORIZED, null);
    return Promise.reject(error)
  } else {
    console.log(error.response);
    vm.$refs.appRef.onError(error.response);
    return Promise.reject(error)
  }
});
