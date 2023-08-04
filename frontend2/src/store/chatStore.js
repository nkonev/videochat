// Utilities
import { defineStore } from 'pinia'
import axios from "axios";
import {isMobileBrowser, setIcon} from "@/utils";

export const useChatStore = defineStore('chat', {
  state: () => {
    return {
        currentUser: null,
        notifications: [],
        notificationsSettings: {},
        showCallButton: false,
        showHangButton: false,
        isShowSearch: true,
        videoChatUsersCount: 0,
        shouldPhoneBlink: false,
        tetATet: false,
        availableOAuth2Providers: [],
        showAlert: false,
        lastError: "",
        errorColor: "",
        showDrawer: isMobileBrowser()
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
    fetchNotifications() {
      axios.get(`/api/notification/notification`).then(( {data} ) => {
        console.debug("fetched notifications =", data);
        this.notifications = data;
        setIcon(data != null && data.length > 0);
      });
      axios.get(`/api/notification/settings`).then(( {data} ) => {
        console.debug("fetched notifications settings =", data);
        this.notificationsSettings = data;
      });
    },
    unsetNotifications() {
      this.notifications = [];
      setIcon(false);
    }
  },

})
