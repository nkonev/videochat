/**
 * plugins/vuetify.js
 *
 * Framework documentation: https://vuetifyjs.com`
 */

// Styles
import '@mdi/font/css/materialdesignicons.css'
import 'vuetify/styles'

// Composables
import { createVuetify } from 'vuetify';
import {getStoredLanguage} from "@/store/localStore";
import en from "@/locale/en";
import ru from "@/locale/ru";
import {isMobileBrowser} from "@/utils";

const config = {
  theme: {
    themes: {
      light: {
        colors: {
          primary: '#1867C0',
          secondary: '#5CBBF6',
        },
      },
    },
  },
  locale: {
    locale: getStoredLanguage(),
    messages: { en, ru },
  }
}

if (isMobileBrowser()) {
  config.defaults = {
    global: {
      transition: false,
      ripple: false,
      scrim: false,
    },
  }
}

// https://vuetifyjs.com/en/introduction/why-vuetify/#feature-guides
export default createVuetify(config);
