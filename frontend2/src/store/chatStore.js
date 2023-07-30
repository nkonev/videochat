// Utilities
import {defineStore, setMapStoreSuffix} from 'pinia'
import axios from "axios";

setMapStoreSuffix('');

export const useChatStore = defineStore('chatStore', {
  state: () => {
    return {
        currentUser: null,
        notificationsCount: 0,
        showCallButton: false,
        showHangButton: false,
        videoChatUsersCount: 0,
        shouldPhoneBlink: false,
        tetATet: false,
        availableOAuth2Providers: [],
        showAlert: false,
        lastError: "",
        errorColor: "",
    }
  },
  actions: {
    unsetUser() {
      this.currentUser = null
    },
    fetchUserProfile() {
        axios.get(`/api/profile`).then(( {data} ) => {
            console.debug("fetched profile =", data);
            this.currentUser = data;
        });
    },
    fetchAvailableOauth2Providers() {
          return axios.get(`/api/oauth2/providers`).then(( {data} ) => {
              console.debug("fetched oauth2 providers =", data);
              this.availableOAuth2Providers = data;
          });
      },

  },

})
