<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640">
            <v-card>
                <v-card-title>{{ title() }}</v-card-title>

                <v-card-text class="px-4 py-0">
                    <v-text-field autofocus v-model="link" :placeholder="placeHolder()" @keyup.native.enter="accept()"/>
                </v-card-text>

                <v-card-actions>
                    <v-spacer/>
                    <v-btn color="primary" class="mr-2 my-1" @click="accept()">{{ $vuetify.lang.t('$vuetify.ok') }}</v-btn>
                    <v-btn v-if="shouldShowClearButton()" class="mr-2 my-1" @click="clear()">{{ $vuetify.lang.t('$vuetify.clear') }}</v-btn>
                    <v-btn color="error" class="my-1" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {EMBED_LINK_SET, MEDIA_LINK_SET, MESSAGE_EDIT_LINK_SET, OPEN_MESSAGE_EDIT_LINK} from "./bus";
import {
    embed,
    link_dialog_type_add_link_to_text,
    link_dialog_type_add_media_by_link, link_dialog_type_add_media_embed,
    media_image,
    media_video
} from "@/utils";

    export default {
        data () {
            return {
                show: false,
                link: null,
                dialogType: '',
                mediaType: ''
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
            showModal(dto) {
                this.$data.show = true;
                this.link = dto.previousUrl;
                this.dialogType = dto.dialogType;
                this.mediaType = dto.mediaType;
            },
            accept() {
                if (this.dialogType == link_dialog_type_add_link_to_text) {
                    bus.$emit(MESSAGE_EDIT_LINK_SET, this.link);
                } else if (this.dialogType == link_dialog_type_add_media_by_link) {
                    bus.$emit(MEDIA_LINK_SET, this.link, this.mediaType);
                } else if (this.dialogType == link_dialog_type_add_media_embed) {
                    bus.$emit(EMBED_LINK_SET, this.link);
                } else {
                    console.error("Wrong dialogType", this.dialogType)
                }
                this.closeModal();
            },
            clear() {
                bus.$emit(MESSAGE_EDIT_LINK_SET, '');
                this.closeModal();
            },
            title() {
                if (this.mediaType == media_video) {
                    return this.$vuetify.lang.t('$vuetify.add_media_video_by_link')
                } else if (this.mediaType == media_image) {
                    return this.$vuetify.lang.t('$vuetify.add_media_image_by_link')
                } else if (this.mediaType == embed) {
                    return this.$vuetify.lang.t('$vuetify.add_media_embed')
                } else {
                    return this.$vuetify.lang.t('$vuetify.message_edit_link')
                }
            },
            placeHolder() {
                if (this.dialogType == link_dialog_type_add_link_to_text) {
                    return "https://google.com"
                } else if (this.dialogType == link_dialog_type_add_media_by_link) {
                    return "https://example.com/file.mp4"
                } else if (this.dialogType == link_dialog_type_add_media_embed) {
                    return '<iframe ... src="https://www.youtube.com/embed/mt-fRVomKdY"'
                }
            },
            shouldShowClearButton() {
                return this.dialogType == link_dialog_type_add_link_to_text
            },
            closeModal() {
                this.show = false;
                this.link = null;
                this.dialogType = '';
                this.mediaType = null;
            }
        },
        created() {
            bus.$on(OPEN_MESSAGE_EDIT_LINK, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_MESSAGE_EDIT_LINK, this.showModal);
        },
    }
</script>
