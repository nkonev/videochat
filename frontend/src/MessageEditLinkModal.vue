<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="480">
          <v-card :title="title()">
                <v-card-text class="py-0 mb-2">
                    <v-text-field density="comfortable" autofocus hide-details variant="underlined" v-model="link" :placeholder="placeHolder()" @keyup.native.enter="accept()"/>
                </v-card-text>

                <v-card-actions>
                    <v-spacer/>
                    <v-btn color="primary" @click="accept()" variant="flat">{{ $vuetify.locale.t('$vuetify.ok') }}</v-btn>
                    <v-btn v-if="shouldShowClearButton()" variant="outlined" @click="clear()">{{ $vuetify.locale.t('$vuetify.clear') }}</v-btn>
                    <v-btn color="red" variant="flat" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {EMBED_LINK_SET, MEDIA_LINK_SET, MESSAGE_EDIT_LINK_SET, OPEN_MESSAGE_EDIT_LINK} from "./bus/bus";
import {
    embed,
    link_dialog_type_add_link_to_text,
    link_dialog_type_add_media_by_link, link_dialog_type_add_media_embed, media_audio,
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
                    bus.emit(MESSAGE_EDIT_LINK_SET, this.link);
                } else if (this.dialogType == link_dialog_type_add_media_by_link) {
                    bus.emit(MEDIA_LINK_SET, {link: this.link, mediaType: this.mediaType});
                } else if (this.dialogType == link_dialog_type_add_media_embed) {
                    bus.emit(EMBED_LINK_SET, this.link);
                } else {
                    console.error("Wrong dialogType", this.dialogType)
                }
                this.closeModal();
            },
            clear() {
                bus.emit(MESSAGE_EDIT_LINK_SET, '');
                this.closeModal();
            },
            title() {
                if (this.mediaType == media_video) {
                    return this.$vuetify.locale.t('$vuetify.add_media_video_by_link')
                } else if (this.mediaType == media_image) {
                    return this.$vuetify.locale.t('$vuetify.add_media_image_by_link')
                } else if (this.mediaType == media_audio) {
                    return this.$vuetify.locale.t('$vuetify.add_media_audio_by_link')
                } else if (this.mediaType == embed) {
                    return this.$vuetify.locale.t('$vuetify.add_media_embed')
                } else {
                    return this.$vuetify.locale.t('$vuetify.message_edit_link')
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
        mounted() {
            bus.on(OPEN_MESSAGE_EDIT_LINK, this.showModal);
        },
        beforeUnmount() {
            bus.off(OPEN_MESSAGE_EDIT_LINK, this.showModal);
        },
    }
</script>
