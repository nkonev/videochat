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
                      :variant="provider.textFieldVariant"
                      :autofocus="isMobile()"
                      hide-details
                      single-line
                      :model-value="provider.getModelValue()"
                      @update:model-value="provider.setModelValue"
                      clearable clear-icon="mdi-close-circle"
                      @keyup.esc="resetInput()" @blur="provider.setShowSearchButton(true)" :label="provider.searchName()">
            <template v-slot:append-inner v-if="provider.switchSearchType">
                <v-btn icon density="compact" @click.prevent="provider.switchSearchType()" :disabled="!provider.canSwitchSearchType()"><v-icon class="search-icon">{{ provider.searchIcon() }}</v-icon></v-btn>
            </template>
        </v-text-field>
    </v-card>

</template>


<script>
import {hasLength} from "./utils";
import {isMobileBrowser} from "#root/renderer/utils.js";

const VIEWPORT_VS_CLIENT_HEIGHT_RATIO = 0.75;

export default {
    props: [
        'provider', // .getModelValue, .setModelValue, .getShowSearchButton, .setShowSearchButton, .searchName, .switchSearchType, .canSwitchSearchType, .searchIcon, .textFieldVariant
    ],
    methods: {
        isMobile() {
            return isMobileBrowser()
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
        reactOnKeyboardChange(event) {
            if (
                (event.target.height * event.target.scale) / window.screen.height <
                VIEWPORT_VS_CLIENT_HEIGHT_RATIO
            ) {
                console.log('keyboard is shown');
            } else {
                console.log('keyboard is hidden');
                // close search line when user on mobile presses Back button
                this.provider.setShowSearchButton(true);
            }
        },
    },
    mounted() {
        if ('visualViewport' in window && this.isMobile()) {
            window.visualViewport.addEventListener('resize', this.reactOnKeyboardChange);
        }
    },
    beforeUnmount() {
        if ('visualViewport' in window && this.isMobile()) {
            window.visualViewport.removeEventListener('resize', this.reactOnKeyboardChange);
        }
    },
}
</script>

<style lang="stylus">
.search-card {
    min-width: 330px;
    margin-left: 1.2em;
    margin-right: 2px;
}
.search-card-mobile {
    width: 100%;
    margin-left: 1.2em;
    margin-right: 0.4em;
}
</style>
