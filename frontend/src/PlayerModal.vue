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
        <v-btn class="close-button" @click="hideModal()" icon="mdi-close" rounded="0" :title="$vuetify.locale.t('$vuetify.close')"></v-btn>
        <template v-if="showArrows">
            <v-btn v-if="canShowLeftArrow" :disabled="loading" :loading="loading" class="arrow-left-button" variant="text" color="white" icon @click="arrowLeft"><v-icon size="x-large">mdi-arrow-left-bold</v-icon></v-btn>
            <v-btn v-if="canShowRightArrow" :disabled="loading" :loading="loading" class="arrow-right-button" variant="text" color="white" icon @click="arrowRight"><v-icon size="x-large">mdi-arrow-right-bold</v-icon></v-btn>
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
            itemsList: [],
            currentItemIdx: 0,
            status: null,
            statusImage: null,
            fileItemUuid: null,
            filename: null,
            loading: false,
        }
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
        showArrows() {
            return this.itemsList.length > 1
        },
        canShowLeftArrow() { // TODO учесть "мы попробовали загрузить и это реально всё"
            return this.currentItemIdx > 0
        },
        canShowRightArrow() {
            return this.currentItemIdx < this.itemsList.length - 1
        },
        isLeftBound() {
          return this.currentItemIdx == 0
        },
        isRightBound() {
          return this.currentItemIdx == this.itemsList.length - 1
        },
    },
    methods: {
        showModal(dto) {
            this.$data.show = true;
            this.$data.dto = dto;
            this.fetchCurrentItemStatus(dto.url).then(()=>{
                if (this.$data.dto?.canSwitch) {
                    const startFromItemId = this.getStartFromItemId();
                    this.fetchMediaListView(startFromItemId);
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
            this.$data.itemsList = [];
            this.$data.currentItemIdx = 0;
            this.$data.status = null;
            this.$data.statusImage = null;
            this.$data.fileItemUuid = null;
            this.$data.filename = null;
            this.$data.loading = false;
        },
        fetchCurrentItemStatus(url) {
            return axios.post(`/api/storage/view/status`, {
                url: url
            }).then((res)=>{
                this.$data.status = res.data.status;
                this.$data.filename = res.data.filename;
                this.$data.statusImage = res.data.statusImage;
                this.$data.fileItemUuid = res.data.fileItemUuid;
            })
        },
        isCorrectStatus() {
            return this.$data.status == "ok"
        },
        fetchMediaListView(startFromItemId, reverse) {
            this.loading = true;
            return axios.post(`/api/storage/view/list`, {
                url: this.$data.dto.url,
                startFromItemId: startFromItemId,
                reverse: reverse,
            }).then((res) => {
                this.itemsList = res.data.items;
                for (let i = 0; i < this.itemsList.length; ++i) {
                    const el = this.itemsList[i];
                    if (el.this) {
                        this.currentItemIdx = i;
                        // console.debug("Setting currentItemIdx", this.currentItemIdx);
                        break
                    }
                }
            }).finally(()=>{
                this.loading = false;
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
                this.currentItemIdx--;
            }
        },
        arrowRight() {
            if (this.canShowRightArrow) {
                this.currentItemIdx++;
            }
        },
        getStartFromItemId(reverse) { // reverse == left
          if (!this.itemsList.length) {
            return 0;
          }

          if (!reverse) {
            return this.itemsList[0].id;
          } else {
            return this.itemsList[this.itemsList.length - 1].id;
          }
        },
        async setEl() {
            if (!this.$data.show) { // guard in case closed modal
                return;
            }

            if (this.isLeftBound) {
              const reverse = true;
              const startFromItemId = this.getStartFromItemId(reverse);
              await this.fetchMediaListView(startFromItemId, reverse)
            }

            if (this.isRightBound) {
              const reverse = false;
              const startFromItemId = this.getStartFromItemId(reverse);
              await this.fetchMediaListView(startFromItemId, reverse)
            }

            const el = this.itemsList[this.currentItemIdx];
            this.$data.dto = {};
            this.$nextTick(()=>{
                this.dto.url = el.url;
                this.dto.previewUrl = el.previewUrl;
                this.dto.canPlayAsVideo = el.canPlayAsVideo;
                this.dto.canShowAsImage = el.canShowAsImage;
                this.filename = el.filename;
                this.fetchCurrentItemStatus(el.url);
            })
        },
        onFileCreatedEvent(dto) {
            if (this.show && this.dto?.url == dto.fileInfoDto.url) {
                this.fetchCurrentItemStatus(dto.fileInfoDto.url).then(()=>{
                  // this is update current page
                  const startFromItemId = this.getStartFromItemId();
                  this.fetchMediaListView(startFromItemId);
                })
            }
        },
        onFileDeletedEvent(dto) {
            if (this.show && this.fileItemUuid == dto.fileInfoDto.fileItemUuid) {
                const startFromItemId = this.getStartFromItemId();
                this.fetchMediaListView(startFromItemId);
            }
        },
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
        },
        currentItemIdx(newValue) {
            this.setEl();
        },
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
