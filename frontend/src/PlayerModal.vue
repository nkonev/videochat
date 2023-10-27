<template>
    <v-row justify="center">
        <v-dialog v-model="show" :persistent="dto?.canPlayAsVideo || dto?.canPlayAsAudio" width="fit-content" max-width="100%" scrollable>
            <v-card :title="getTitle()">
                <v-card-text class="d-flex justify-center">
                    <video class="video-custom-class" v-if="dto?.canPlayAsVideo" :src="dto.url" :poster="dto.previewUrl" playsInline controls/>
                    <img class="image-custom-class" v-if="dto?.canShowAsImage" :src="dto.url"/>
                    <audio class="audio-custom-class" v-if="dto?.canPlayAsAudio" :src="dto.url" controls/>
                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-spacer></v-spacer>
                    <v-btn color="red" variant="flat" @click="hideModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {
    PLAYER_MODAL,
} from "./bus/bus";

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
                return this.$vuetify.locale.t('$vuetify.play')
            } else if (this.$data.dto?.canPlayAsAudio) {
                return this.$vuetify.locale.t('$vuetify.play')
            } else if (this.$data.dto?.canShowAsImage) {
                return this.$vuetify.locale.t('$vuetify.view')
            } else {
                return ""
            }
        },
    },
    mounted() {
        bus.on(PLAYER_MODAL, this.showModal);
    },
    beforeUnmount() {
        bus.off(PLAYER_MODAL, this.showModal);
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

<style lang="stylus" scoped>
@import "messageBody.styl"
</style>
