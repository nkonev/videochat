<template>
    <v-btn v-if="provider.getShowSearchButton() && isMobile()" icon :title="provider.searchName()" @click="onOpenSearch()">
        <v-icon>{{ hasSearchString() ? 'mdi-magnify-close' : 'mdi-magnify'}}</v-icon>
    </v-btn>

    <v-card v-if="!provider.getShowSearchButton() || !isMobile()" variant="plain" :class="isMobile() ? 'search-card-mobile' : 'search-card'">
        <v-text-field density="compact" variant="solo" :autofocus="isMobile()" hide-details single-line
                      :model-value="provider.getModelValue()"
                      @update:model-value="provider.setModelValue"
                      clearable clear-icon="mdi-close-circle"
                      @keyup.esc="resetInput()" @blur="provider.setShowSearchButton(true)" :label="provider.searchName()">
            <template v-slot:append-inner>
                <v-btn icon density="compact" @click.prevent="provider.switchSearchType()" :disabled="!provider.canSwitchSearchType()"><v-icon class="search-icon">{{ provider.searchIcon() }}</v-icon></v-btn>
            </template>
        </v-text-field>
    </v-card>

</template>


<script>
import {hasLength} from "@/utils";

export default {
    props: [
        'provider', // .getModelValue, .setModelValue, .getShowSearchButton, .setShowSearchButton, .searchName, .switchSearchType, .canSwitchSearchType, .searchIcon
    ],
    methods: {
        onOpenSearch() {
            this.provider.setShowSearchButton(false);
        },
        resetInput() {
            this.provider.setModelValue(null)
        },
        hasSearchString() {
            return hasLength(this.provider.getModelValue())
        },
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
