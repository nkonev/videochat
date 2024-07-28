<template>
    <v-overlay v-model="show" width="100%" height="100%" opacity="0.7">
        <span class="d-flex justify-center align-center" style="width: 100%; height: 100%">
            <video class="video-custom-class-view" v-if="dto?.canPlayAsVideo" :src="dto.url" :poster="dto.previewUrl" playsInline controls/>
            <img class="image-custom-class-view" v-if="dto?.canShowAsImage" :src="dto.url"/>
            <audio class="audio-custom-class-view" v-if="dto?.canPlayAsAudio" :src="dto.url" controls/>
        </span>
        <v-btn class="close-button" @click="hideModal()" icon="mdi-close" rounded="0" :title="$vuetify.locale.t('$vuetify.close')"></v-btn>
        <template v-if="showArrows">
            <v-btn v-if="canShowLeftArrow" class="arrow-left-button" variant="text" color="white" icon @click="arrowLeft"><v-icon size="x-large">mdi-arrow-left-bold</v-icon></v-btn>
            <v-btn v-if="canShowRightArrow" class="arrow-right-button" variant="text" color="white" icon @click="arrowRight"><v-icon size="x-large">mdi-arrow-right-bold</v-icon></v-btn>
        </template>
    </v-overlay>
</template>

<script>
import bus, {
    PLAYER_MODAL,
} from "#root/common/bus";
import axios from "axios";

export default {
    data () {
        return {
            show: false,
            dto: null,
            viewList: [],
            thisIdx: 0,
        }
    },
    computed: {
        showArrows() {
            return this.viewList.length > 1
        },
        canShowLeftArrow() {
            return this.thisIdx > 0
        },
        canShowRightArrow() {
            return this.thisIdx < this.viewList.length - 1
        },
    },
    methods: {
        showModal(dto) {
            this.$data.show = true;
            this.$data.dto = dto;
            this.fetchMediaListView();
            window.addEventListener("keydown", this.onKeyPress);
        },
        hideModal() {
            this.$data.show = false;
            this.$data.dto = null;
            this.$data.viewList = [];
            this.$data.thisIdx = 0;
            window.removeEventListener("keydown", this.onKeyPress);
        },
        fetchMediaListView() {
            axios.post(`/api/storage/public/view/list`, {
                url: this.$data.dto.url
            }).then((res) => {
                this.viewList = res.data.items;
                for (let i = 0; i < this.viewList.length; ++i) {
                    const el = this.viewList[i];
                    if (el.this) {
                        this.thisIdx = i;
                        // console.debug("Setting thisIdx", this.thisIdx);
                        break
                    }
                }
            })
        },
        onKeyPress(event) {
            switch (event.key) {
                case "ArrowLeft":
                    this.arrowLeft()
                    break;
                case "ArrowRight":
                    this.arrowRight()
                    break;
            }
        },
        arrowLeft() {
            if (this.canShowLeftArrow) {
                this.thisIdx--;
                this.setEl();
            }
        },
        arrowRight() {
            if (this.canShowRightArrow) {
                this.thisIdx++;
                this.setEl();
            }
        },
        setEl() {
            const el = this.viewList[this.thisIdx];
            this.$data.dto = null;
            this.$nextTick(()=>{
                this.$data.dto = {};
                this.dto.url = el.url;
                this.dto.previewUrl = el.previewUrl;
                this.dto.canPlayAsVideo = el.canPlayAsVideo;
                this.dto.canShowAsImage = el.canShowAsImage;
            })
        }
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
@import "../styles/constants.styl"

.close-button {
    position absolute
    top 0
    right 0
}

.arrow-left-button {
    position absolute
    left 0.2em
    top: 0;
    bottom: 0;
    margin: auto 0;
    text-shadow: 1px 1px 2px #000;
}

.arrow-right-button {
    position absolute
    right 0.2em
    top: 0;
    bottom: 0;
    margin: auto 0;
    text-shadow: 1px 1px 2px #000;
}

.image-custom-class-view {
    max-width: 100% !important
    max-height: 100% !important
}

.video-custom-class-view {
    max-width: 100% !important
    max-height: 100% !important
}

@media screen and (max-width: $mobileWidth) {
    .image-custom-class-view {
        max-width: 100% !important
        height: 360px !important
    }

    .video-custom-class-view {
        max-width: 100% !important
        height: 360px !important
    }
}

.audio-custom-class-view {
    min-width: 600px
    max-width: 100% !important
}

</style>
