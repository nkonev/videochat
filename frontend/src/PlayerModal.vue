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
            <v-btn v-if="canShowLeftArrow" :loading="loading" class="arrow-left-button" variant="text" color="white" icon @click="arrowLeft"><v-icon size="x-large">mdi-arrow-left-bold</v-icon></v-btn>
            <v-btn v-if="canShowRightArrow" :loading="loading" class="arrow-right-button" variant="text" color="white" icon @click="arrowRight"><v-icon size="x-large">mdi-arrow-right-bold</v-icon></v-btn>
        </template>
    </v-overlay>
</template>

<script>
import bus, {
    FILE_CREATED, FILE_REMOVED,
    PLAYER_MODAL,
} from "./bus/bus";
import axios from "axios";

const defaultReverse = false
const PAGE_SIZE = 20
const setForward = 'setForward';
const setBackward = 'setBackward';

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
            createDateTime: null,
            filename: null,
            loading: false,
            noElementsAtLeft: false,
            noElementsAtRight: false,
        }
    },
    computed: {
        chatId() {
            return this.$route.params.id
        },
        showArrows() {
            return this.itemsList.length > 1
        },
        canShowLeftArrow() {
            return this.currentItemIdx > 0 || (!this.noElementsAtLeft)
        },
        canShowRightArrow() {
            return (this.currentItemIdx < this.itemsList.length - 1) || (!this.noElementsAtRight)
        },
        isLeftBound() {
          return this.currentItemIdx == 0
        },
        isRightBound() {
          return this.currentItemIdx == this.itemsList.length - 1
        },
    },
    methods: {
        setDto(d) {
          this.dto = {}
          this.dto.url = d.url;
          this.dto.previewUrl = d.previewUrl;
          this.dto.canPlayAsVideo = d.canPlayAsVideo;
          this.dto.canShowAsImage = d.canShowAsImage;
          this.dto.canPlayAsAudio = d.canPlayAsAudio;
        },
        setDtoExtended(d) {
          this.setDto(d);
          this.dto.canSwitch = d.canSwitch;
        },
        async showModal(dto) {
            this.$data.show = true;
            this.setDtoExtended(dto);
            await this.fetchCurrentItemStatus(dto.url)
            if (this.$data.dto?.canSwitch) {
                const startFrom = this.getStartFromItemId();
                const items = await this.fetchBothDirectionItems(startFrom, defaultReverse)
                this.setItems(items);

                window.addEventListener("keydown", this.onKeyPress);
            }
        },
        async fetchBothDirectionItems(startFrom, reverse) {
          this.loading = true;
          let nextItems = await this.fetchItems(startFrom, reverse, false);
          let prevItems = await this.fetchItems(startFrom, !reverse, true);
          let items;
          if (!reverse) {
            items = prevItems.reverse().concat(nextItems);
          } else {
            items = nextItems.reverse().concat(prevItems);
          }
          this.loading = false;
          console.info("items", items, "noElementsAtLeft", this.noElementsAtLeft, "noElementsAtRight", this.noElementsAtRight)
          return items;
        },
        setItems(items, next) {
          this.itemsList = items;
          let shouldLoop = true;
          for (let i = 0; i < this.itemsList.length && shouldLoop; ++i) {
            const el = this.itemsList[i];
            if (el.this) {
              if (next) {
                switch (next) {
                 case setForward:
                   const nextIdx1 = i + 1;
                   if (nextIdx1 < this.itemsList.length) {
                     this.currentItemIdx = nextIdx1;
                     shouldLoop = false;
                     console.debug("setForward currentItemIdx", this.currentItemIdx);
                   } else {
                     console.warn("skipping setting forward idx")
                   }
                   break
                 case setBackward:
                   const nextIdx2 = i - 1;
                   if (nextIdx2 >= 0) {
                     this.currentItemIdx = nextIdx2;
                     shouldLoop = false;
                     console.debug("setBackward currentItemIdx", this.currentItemIdx);
                   } else {
                     console.warn("skipping setting backward idx")
                   }
                   break
                }
              } else {
                console.debug("regular set currentItemIdx", this.currentItemIdx);
                this.currentItemIdx = i;
              }
              break
            }
          }
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
            this.$data.createDateTime = null;
            this.$data.filename = null;
            this.$data.loading = false;
            this.$data.noElementsAtLeft = false;
            this.$data.noElementsAtRight = false;
        },
        fetchCurrentItemStatus(url) {
            return axios.post(`/api/storage/view/status`, {
                url: url
            }).then((res)=>{
                this.$data.status = res.data.status;
                this.$data.filename = res.data.filename;
                this.$data.statusImage = res.data.statusImage;
                this.$data.fileItemUuid = res.data.fileItemUuid;
                this.$data.createDateTime = res.data.createDateTime;
            })
        },
        isCorrectStatus() {
            return this.$data.status == "ok"
        },
        fetchItems(startFrom, reverse, includeStartingFrom) {
            return axios.post(`/api/storage/view/list`, {
                size: PAGE_SIZE,
                url: this.$data.dto.url,
                startingFromCreateDateTime: startFrom?.createDateTime,
                startingFromFilename: startFrom?.filename,
                reverse: reverse,
                includeStartingFrom: includeStartingFrom,
            }).then((res) =>{
                if (reverse) { // left
                  this.noElementsAtLeft = (res.data.items.length < PAGE_SIZE)
                } else {
                  this.noElementsAtRight = (res.data.items.length < PAGE_SIZE)
                }
                return res.data.items
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
        async arrowLeft() {
            if (this.canShowLeftArrow) {
                this.currentItemIdx--;
                await this.loadOnBound();
                await this.setEl();
            }
        },
        async arrowRight() {
            if (this.canShowRightArrow) {
                this.currentItemIdx++;
                await this.loadOnBound();
                await this.setEl();
            }
        },
        getStartFromItemId() {
          return {createDateTime: this.createDateTime, filename: this.filename};
        },
        async loadOnBound() {
          if (!this.$data.show) { // guard in case closed modal
            return;
          }

          if (this.isLeftBound) {
            console.info("isLeftBound")
            if (!this.noElementsAtLeft) {
              const reverse = true;
              const startFrom = this.getStartFromItemId();
              let items = await this.fetchBothDirectionItems(startFrom, reverse)
              this.setItems(items, setBackward);
            }
          } else if (this.isRightBound) {
            console.info("isRightBound")
            if (!this.noElementsAtRight) {
              const reverse = false;
              const startFrom = this.getStartFromItemId();
              let items = await this.fetchBothDirectionItems(startFrom, reverse)
              this.setItems(items, setForward);
            }
          }
        },
        async setEl() {
            if (!this.$data.show) { // guard in case closed modal
              return;
            }

            const el = this.itemsList[this.currentItemIdx];
            this.$data.dto = {};
            this.$nextTick(()=>{
                this.setDto(el);
                this.fetchCurrentItemStatus(el.url);
            })
        },
        onFileCreatedEvent(dto) {
            if (this.show && this.dto?.url == dto.fileInfoDto.url) {
                this.fetchCurrentItemStatus(dto.fileInfoDto.url).then(()=>{
                  // this is update current page
                  const startFrom = this.getStartFromItemId();
                  const items = this.fetchBothDirectionItems(startFrom, defaultReverse);
                  this.setItems(items);
                })
            }
        },
        onFileDeletedEvent(dto) {
            if (this.show && this.fileItemUuid == dto.fileInfoDto.fileItemUuid) {
                const startFrom = this.getStartFromItemId();
                const items = this.fetchBothDirectionItems(startFrom, defaultReverse);
                this.setItems(items);
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
        currentItemIdx(){
          console.debug("currentItemIdx is", this.currentItemIdx);
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
