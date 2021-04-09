import '@mdi/font/css/materialdesignicons.css' // Ensure you are using css-loader
import Vue from 'vue'
import Vuetify from 'vuetify/lib/framework' // https://vuetifyjs.com/en/features/sass-variables/#compilation-time

import { library } from '@fortawesome/fontawesome-svg-core'
import { FontAwesomeIcon } from '@fortawesome/vue-fontawesome'
import { faFacebook } from '@fortawesome/free-brands-svg-icons/faFacebook'
import { faVk } from '@fortawesome/free-brands-svg-icons/faVk'
import { faGoogle } from '@fortawesome/free-brands-svg-icons/faGoogle'

library.add(faFacebook, faVk, faGoogle);
Vue.component('font-awesome-icon', FontAwesomeIcon) // Register component globally

Vue.use(Vuetify);

export default new Vuetify({
    icons: {
        iconfont: 'mdi', // default - only for display purposes
    },
})