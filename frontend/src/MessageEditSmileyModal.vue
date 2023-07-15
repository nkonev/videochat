<template>
    <v-row justify="center">
        <v-dialog v-model="show" width="fit-content">
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.message_edit_smiley') }}</v-card-title>

                <v-card-text class="py-0 pt-2 px-4 smiley-buttons">
                    <v-row no-gutters>
                        <v-btn tile icon large class="display-1">😀</v-btn>
                        <v-btn tile icon large class="display-1">😂</v-btn>
                        <v-btn tile icon large class="display-1">🥰</v-btn>
                        <v-btn tile icon large class="display-1">💋</v-btn>
                        <v-btn tile icon large class="display-1">❤️</v-btn>
                        <v-btn tile icon large class="display-1">🤔</v-btn>
                    </v-row>
                    <v-row no-gutters>
                        <v-btn tile icon large class="display-1">😐</v-btn>
                        <v-btn tile icon large class="display-1">🤒</v-btn>
                        <v-btn tile icon large class="display-1">🤮</v-btn>
                        <v-btn tile icon large class="display-1">🥴</v-btn>
                        <v-btn tile icon large class="display-1">😎</v-btn>
                        <v-btn tile icon large class="display-1">😨</v-btn>
                    </v-row>
                    <v-row no-gutters>
                        <v-btn tile icon large class="display-1">😡</v-btn>
                        <v-btn tile icon large class="display-1">👿</v-btn>
                        <v-btn tile icon large class="display-1">💩</v-btn>
                        <v-btn tile icon large class="display-1">😇</v-btn>
                        <v-btn tile icon large class="display-1">🤐</v-btn>
                        <v-btn tile icon large class="display-1">💣</v-btn>
                    </v-row>
                    <v-row no-gutters>
                        <v-btn tile icon large class="display-1">👍</v-btn>
                        <v-btn tile icon large class="display-1">👎</v-btn>
                        <v-btn tile icon large class="display-1">🤟</v-btn>
                        <v-btn tile icon large class="display-1">🙏</v-btn>
                        <v-btn tile icon large class="display-1">💪</v-btn>
                        <v-btn tile icon large class="display-1">👏</v-btn>
                    </v-row>
                </v-card-text>

                <v-card-actions>
                    <v-spacer/>
                    <v-btn class="my-1" color="error" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_MESSAGE_EDIT_SMILEY} from "./bus";

    export default {
        data () {
            return {
                show: false,
            }
        },
        watch: {
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            }
        },
        methods: {
            showModal(colorMode, color) {
                this.$data.show = true;
            },
            closeModal() {
                this.show = false;
            },
        },
        created() {
            bus.$on(OPEN_MESSAGE_EDIT_SMILEY, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_MESSAGE_EDIT_SMILEY, this.showModal);
        },
    }
</script>

<style lang="stylus">
    .smiley-buttons {
        button {
            color: rgba(0, 0, 0, 1) !important
        }
    }
</style>
