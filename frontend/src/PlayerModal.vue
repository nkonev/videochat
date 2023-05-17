<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" :persistent="true">
            <v-card>
                <v-card-title>{{ getTitle() }}</v-card-title>

                <v-card-text class="py-0 d-flex justify-center">
                        <video class="video-custom-class" v-if="dto?.canPlayAsVideo" :src="dto.url" :poster="dto.previewUrl" playsInline controls></video>
                        <img class="image-custom-class" v-if="dto?.canShowAsImage" :src="dto.url"></img>
                </v-card-text>

                <v-card-actions class="pa-4">
                    <v-btn class="mr-4" @click="hideModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {
    PLAYER_MODAL,
} from "./bus";

export default {
    data () {
        return {
            show: false,
            dto: null,
        }
    },
    methods: {
        showModal(dto) {
            this.$data.show = true;
            this.$data.dto = dto;
        },
        hideModal() {
            this.$data.show = false;
            this.$data.dto = null;
        },
        getTitle() {
            if (this.$data.dto?.canPlayAsVideo) {
                return this.$vuetify.lang.t('$vuetify.play')
            } else if (this.$data.dto?.canShowAsImage) {
                return this.$vuetify.lang.t('$vuetify.view')
            } else {
                return ""
            }
        },
    },
    created() {
        bus.$on(PLAYER_MODAL, this.showModal);
    },
    destroyed() {
        bus.$off(PLAYER_MODAL, this.showModal);
    },
    watch: {
        show(newValue) {
            if (!newValue) {
                this.hideModal();
            }
        }
    }
}
</script>

<style lang="stylus">
    @import "message.styl"
</style>
