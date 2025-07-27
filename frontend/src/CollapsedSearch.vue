<template>
    <v-btn v-if="provider.getShowSearchButton() && isMobile()"
           icon
           :title="provider.searchName()"
           @click="onOpenSearch()"
           variant="text"
    >
        <v-icon>{{ hasSearchString() ? 'mdi-magnify-close' : 'mdi-magnify'}}</v-icon>
    </v-btn>

    <v-card v-if="!provider.getShowSearchButton() || !isMobile()" variant="plain" :class="isMobile() ? 'search-card-mobile' : 'search-card'">
      <div :class="wrapperClass()">
        <v-text-field density="compact"
                      @focusout="onFocusOut"
                      :variant="provider.textFieldVariant"
                      :autofocus="isMobile()"
                      hide-details
                      :model-value="provider.getModelValue()"
                      @update:model-value="provider.setModelValue"
                      clearable clear-icon="mdi-close-circle"
                      @keyup.esc="resetInput()" @blur="provider.setShowSearchButton(true)" :label="provider.searchName()">
            <template v-slot:append-inner v-if="provider.switchSearchType">
                <v-btn icon density="compact" @click.prevent="provider.switchSearchType()" :disabled="!provider.canSwitchSearchType()" :title="$vuetify.locale.t('$vuetify.switch_search_by')"><v-icon class="search-icon">{{ provider.searchIcon() }}</v-icon></v-btn>
            </template>
        </v-text-field>
      </div>
    </v-card>

</template>


<script>
import {hasLength} from "@/utils";

export default {
    props: [
        'provider', // .getModelValue, .setModelValue, .getShowSearchButton, .setShowSearchButton, .searchName, .switchSearchType, .canSwitchSearchType, .searchIcon, .textFieldVariant, .beforeOpenCallback, .afterCloseCallback
        'paddingsY'
    ],
    methods: {
        onOpenSearch() {
            if (this.provider.beforeOpenCallback) {
              this.provider.beforeOpenCallback()
            }
            this.provider.setShowSearchButton(false);
        },
        resetInput() {
            this.provider.setModelValue(null)
        },
        hasSearchString() {
            return hasLength(this.provider.getModelValue())
        },
        onFocusOut() {
          if (this.isMobile()) {
            this.provider.setShowSearchButton(true);
            if (this.provider.afterCloseCallback) {
              this.provider.afterCloseCallback()
            }
          }
        },
        wrapperClass() {
          let cl = [];
          if (this.paddingsY) {
            cl.push('py-2')
          }
          return cl
        },
    },
    mounted() {
    },
    beforeUnmount() {
    },
}
</script>

<style lang="stylus">
.search-card {
    align-self center
    min-width: 330px;
    margin-left: 0.4em;
    margin-right: 2px;
}
.search-card-mobile {
    align-self center
    width: 100%;
    margin-right: 0.4em;
}
</style>
