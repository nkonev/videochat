<template>
    <v-btn v-if="showSearchButton && isMobile()" icon :title="searchName" @click="onOpenSearch()">
        <v-icon>{{ hasSearchString ? 'mdi-magnify-close' : 'mdi-magnify'}}</v-icon>
    </v-btn>

    <v-card v-if="!showSearchButton || !isMobile()" variant="plain" :class="isMobile() ? 'search-card-mobile' : 'search-card'">
        <v-text-field density="compact" variant="solo" :autofocus="isMobile()" hide-details single-line v-model="searchStringFacade" clearable clear-icon="mdi-close-circle" @keyup.esc="resetInput" @blur="showSearchButton=true" :label="searchName()">
            <template v-slot:append-inner>
                <v-btn icon density="compact" @click.prevent="switchSearchType()" :disabled="!canSwitchSearchType()"><v-icon class="search-icon">{{ searchIcon }}</v-icon></v-btn>
            </template>
        </v-text-field>
    </v-card>

</template>


<script>
export default {
    data() {
        return {
            showSearchButton: true,
        }
    },
    methods: {
        onOpenSearch() {
            this.showSearchButton = false;
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
