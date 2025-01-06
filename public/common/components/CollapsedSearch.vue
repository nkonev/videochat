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
        <v-text-field density="compact"
                      @focusout="onFocusOut"
                      :variant="provider.textFieldVariant"
                      :autofocus="isMobile()"
                      hide-details
                      single-line
                      :model-value="provider.getModelValue()"
                      @update:model-value="provider.setModelValue"
                      clearable clear-icon="mdi-close-circle"
                      @keyup.esc="resetInput()" @blur="provider.setShowSearchButton(true)" :label="provider.searchName()">
            <template v-slot:append-inner>
                <v-btn icon density="compact" :disabled="true"><v-icon class="search-icon">{{ provider.searchIcon() }}</v-icon></v-btn>
            </template>
        </v-text-field>
    </v-card>

</template>


<script>
import {usePageContext} from "#root/renderer/usePageContext.js";
import {hasLength} from "#root/common/utils";

export default {
    setup() {
        const pageContext = usePageContext();

        // expose to template and other options API hooks
        return {
            pageContext
        }
    },
    props: [
        'provider', // .getModelValue, .setModelValue, .getShowSearchButton, .setShowSearchButton, .searchName, .switchSearchType, .canSwitchSearchType, .searchIcon, .textFieldVariant
    ],
    methods: {
        isMobile() {
            return this.pageContext.isMobile
        },
        onOpenSearch() {
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
          }
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
    min-width: 330px;
    margin-right: 2px;
}
.search-card-mobile {
    width: 100%;
    margin-right: 0.4em;
}
</style>
