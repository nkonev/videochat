<template>
    <v-overlay v-model="show" width="100%" height="100%" opacity="0.7" class="player-modal">
        <span class="d-flex flex-column justify-center align-center" style="width: 100%; height: 100%">
            <div class="d-flex justify-center align-center flex-shrink-0 player-media-wrapper">
                <template v-if="isCorrectStatus()">
                    <video class="video-custom-class-view" v-if="dto?.canPlayAsVideo" :src="dto.url" :poster="dto.previewUrl" playsInline controls/>
                    <img class="image-custom-class-view" v-if="dto?.canShowAsImage" :src="dto.url"/>
                    <audio class="audio-custom-class-view" v-if="dto?.canPlayAsAudio" :src="dto.url" controls/>
                </template>
                <template v-else>
                    <img class="image-custom-class-view" :src="statusImage"/>
                </template>
            </div>
            <div v-if="filename" class="player-caption-placeholder flex-shrink-0 d-flex">
              <span class="player-caption-text">{{filename}}</span>
            </div>
        </span>
        <v-btn class="close-button" @click="hideModal()" icon="mdi-close" rounded="0" title="Close"></v-btn>
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
            status: null,
            statusImage: null,
            filename: null,
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
            this.fetchStatus(dto.url).then(()=>{
                this.$data.dto = dto;
                this.fetchMediaListView();
            })
            window.addEventListener("keydown", this.onKeyPress);
        },
        hideModal() {
            this.$data.show = false;
            this.$data.dto = null;
            this.$data.viewList = [];
            this.$data.thisIdx = 0;
            window.removeEventListener("keydown", this.onKeyPress);
            this.$data.status = null;
            this.$data.statusImage = null;
            this.$data.filename = null;
        },
        fetchStatus(url) {
            return axios.post(`/api/storage/public/view/status`, {
                url: url
            }).then((res)=>{
                this.$data.status = res.data.status;
                this.$data.filename = res.data.filename;
                this.$data.statusImage = res.data.statusImage;
            })
        },
        isCorrectStatus() {
            return this.$data.status == "ok"
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
                this.filename = el.filename;
            })
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

.audio-custom-class-view {
    min-width: 640px
    max-width: 100% !important
}
@media screen and (max-width: $mobileWidth) {
  .audio-custom-class-view {
    min-width: 100%
  }
}

.player-caption-placeholder {
  background black
  width 100%
  height 1.4em
}

.player-media-wrapper {
  width: 100%;
  height: calc(100% - 1.4em)
}

.player-caption-text {
  color white
  // ellipsisis start
  white-space: nowrap;
  text-overflow: ellipsis;
  overflow: hidden;
  // ellipsisis end
}

</style>
