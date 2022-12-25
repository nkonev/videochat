<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card>
                <v-card-title>{{ title() }}</v-card-title>

                <v-card-text>
                    <v-row dense>
                        <v-col
                            v-for="card in cards"
                            :key="card.title"
                            :cols="card.flex"
                        >
                            <v-hover>
                                <template v-slot:default="{ hover }">
                                    <v-card>
                                        <v-img
                                            :src="card.src"
                                            class="white--text align-end"
                                            gradient="to bottom, rgba(0,0,0,.1), rgba(0,0,0,.5)"
                                            height="200px"
                                        >
                                            <v-card-title v-text="card.title"></v-card-title>
                                        </v-img>

                                        <v-fade-transition>
                                            <v-overlay
                                                v-if="hover"
                                                absolute
                                                @click="accept()"
                                                style="cursor: pointer"
                                            >
                                            </v-overlay>
                                        </v-fade-transition>

                                    </v-card>
                                </template>
                            </v-hover>
                        </v-col>
                    </v-row>
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-spacer/>
                    <v-btn color="primary" class="mr-2" @click="fromDisk()"><v-icon color="white">mdi-file-upload</v-icon>{{ $vuetify.lang.t('$vuetify.choose_file_from_disk') }}</v-btn>
                    <v-btn color="error" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_MESSAGE_EDIT_MEDIA} from "./bus";
    import {media_image, media_video} from "@/utils";

    export default {
        data () {
            return {
                show: false,
                type: '',
                fromDiskCallback: null,
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
            showModal(type, fromDiskCallback) {
                this.$data.show = true;
                this.type = type;
                this.fromDiskCallback = fromDiskCallback;
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
                this.type = '';
                this.fromDiskCallback = null;
            },
            title() {
                switch (this.type) {
                    case media_video:
                        return this.$vuetify.lang.t('$vuetify.message_edit_video')
                    case media_image:
                        return this.$vuetify.lang.t('$vuetify.message_edit_image')
                }
            },
            fromDisk() {
                if (this.fromDiskCallback) {
                    this.fromDiskCallback();
                }
                this.closeModal();
            }
        },
        created() {
            bus.$on(OPEN_MESSAGE_EDIT_MEDIA, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_MESSAGE_EDIT_MEDIA, this.showModal);
        },
    }
</script>
