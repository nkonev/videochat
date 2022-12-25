<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.message_edit_video') }}</v-card-title>

                <v-card-text>
                    <v-row dense>
                        <v-col
                            v-for="card in cards"
                            :key="card.title"
                            :cols="card.flex"
                        >
                            <v-card>
                                <v-img
                                    :src="card.src"
                                    class="white--text align-end"
                                    gradient="to bottom, rgba(0,0,0,.1), rgba(0,0,0,.5)"
                                    height="200px"
                                >
                                    <v-card-title v-text="card.title"></v-card-title>
                                </v-img>

                                <v-card-actions>
                                    <v-spacer></v-spacer>
                                    <v-btn @click="accept()">
                                        {{ $vuetify.lang.t('$vuetify.choose') }}
                                    </v-btn>
                                </v-card-actions>
                            </v-card>
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-spacer/>
                    <v-btn color="primary" class="mr-2" @click="accept()"><v-icon color="white">mdi-file-upload</v-icon>{{ $vuetify.lang.t('$vuetify.choose_file_from_disk') }}</v-btn>
                    <v-btn color="error" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_MESSAGE_EDIT_VIDEO} from "./bus";

    export default {
        data () {
            return {
                show: false,
                cards: [
                    { title: 'Pre-fab homes lorem ipsum dolor lorem ipsum dolor lorem ipsum dolor lorem ipsum dolor lorem ipsum.mp4', src: 'https://cdn.vuetifyjs.com/images/cards/house.jpg', flex: 6 },
                    { title: 'Favorite road trips', src: 'https://cdn.vuetifyjs.com/images/cards/road.jpg', flex: 6 },
                    { title: 'Best airlines', src: 'https://cdn.vuetifyjs.com/images/cards/plane.jpg', flex: 6 },
                ],
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
            accept() {
                // TODO
                this.closeModal();
            },
            clear() {
                this.closeModal();
            },
            closeModal() {
                this.show = false;
            },
        },
        created() {
            bus.$on(OPEN_MESSAGE_EDIT_VIDEO, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_MESSAGE_EDIT_VIDEO, this.showModal);
        },
    }
</script>
