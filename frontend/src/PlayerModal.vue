<template>
    <v-overlay v-model="show" width="100%" height="100%" opacity="0.7">
        <span class="d-flex justify-center align-center" style="width: 100%; height: 100%">
            <template v-if="isCorrectStatus()">
                <video class="video-custom-class-view" v-if="dto?.canPlayAsVideo" :src="dto.url" :poster="dto.previewUrl" playsInline controls/>
                <img class="image-custom-class-view" v-if="dto?.canShowAsImage" :src="dto.url"/>
                <audio class="audio-custom-class-view" v-if="dto?.canPlayAsAudio" :src="dto.url" controls/>
            </template>
            <template v-else>
                <img class="image-custom-class-view" :src="statusImage"/>
            </template>
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
    FILE_CREATED, FILE_REMOVED,
    PLAYER_MODAL,
} from "./bus/bus";
import axios from "axios";

export default {
    data () {
        return {
            show: false,
            dto: null,
            viewList: [],
            thisIdx: 0,
            status: null,
            statusImage: null,
            fileItemUuid: null,
        }
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
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
            this.fetchStatus(dto.url).then(()=>{
                this.$data.dto = dto;
                if (this.$data.dto?.canSwitch) {
                    this.fetchMediaListView();
                    window.addEventListener("keydown", this.onKeyPress);
                }
            })
        },
        hideModal() {
            this.$data.show = false;
            if (this.$data.dto?.canSwitch) {
                window.removeEventListener("keydown", this.onKeyPress);
            }
            this.$data.dto = null;
            this.$data.viewList = [];
            this.$data.thisIdx = 0;
            this.$data.status = null;
            this.$data.statusImage = null;
            this.$data.fileItemUuid = null;
        },
        fetchStatus(url) {
            return axios.post(`/api/storage/view/status`, {
                url: url
            }).then((res)=>{
                this.$data.status = res.data.status
                this.$data.statusImage = res.data.statusImage;
                this.$data.fileItemUuid = res.data.fileItemUuid;
            })
        },
        isCorrectStatus() {
            return this.$data.status == "ok"
        },
        fetchMediaListView() {
            axios.post(`/api/storage/view/list`, {
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
                this.fetchStatus(el.url).then(()=>{
                    this.dto.url = el.url;
                    this.dto.previewUrl = el.previewUrl;
                    this.dto.canPlayAsVideo = el.canPlayAsVideo;
                    this.dto.canShowAsImage = el.canShowAsImage;
                })
            })
        },
        onFileCreatedEvent(dto) {
            if (this.show && this.dto?.url == dto.fileInfoDto.url) {
                this.fetchStatus(dto.fileInfoDto.url).then(()=>{
                    this.fetchMediaListView();
                })
            }
        },
        onFileDeletedEvent(dto) {
            if (this.show && this.fileItemUuid == dto.fileInfoDto.fileItemUuid) {
                this.fetchMediaListView();
            }
        }
    },
    mounted() {
        bus.on(PLAYER_MODAL, this.showModal);
        bus.on(FILE_CREATED, this.onFileCreatedEvent);
        bus.on(FILE_REMOVED, this.onFileDeletedEvent);
    },
    beforeUnmount() {
        bus.off(PLAYER_MODAL, this.showModal);
        bus.off(FILE_CREATED, this.onFileCreatedEvent);
        bus.off(FILE_REMOVED, this.onFileDeletedEvent);
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
@import "constants.styl"

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
