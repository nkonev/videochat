// Utilities
import { defineStore } from 'pinia'
import {SEARCH_MODE_POSTS} from "@/mixins/searchString";

export const useBlogStore = defineStore('blog', {
  state: () => {
    return {
        lastError: "",
        errorColor: "",
        isShowSearch: false,
        searchType: SEARCH_MODE_POSTS,
        title: "",
        progressCount: 0,
    }
  },
  actions: {
    incrementProgressCount() {
      this.progressCount++
    },
    decrementProgressCount() {
      if (this.progressCount > 0) {
        this.progressCount--
      } else {
        const err = new Error();
        console.warn("Attempt to decrement progressCount lower than 0", err.stack)
      }
    },
  },

})
