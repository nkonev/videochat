<template>
    <v-row justify="center">
        <v-dialog v-model="show" width="fit-content">
            <v-card :title="$vuetify.locale.t('$vuetify.message_edit_smiley')">
                <v-card-text class="py-0 pt-2 px-4 smiley-buttons">
                    <v-row :key="sli" v-for="(smileyLine, sli) in smileys" no-gutters>
                        <span :key="si" @click="onSmileyClick(smiley)" v-for="(smiley, si) in smileyLine" class="smiley">{{smiley}}</span>
                    </v-row>
                </v-card-text>

                <v-card-actions>
                    <v-spacer/>
                    <v-btn color="red" variant="flat" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_MESSAGE_EDIT_SMILEY} from "./bus/bus";

    export default {
        data () {
            return {
                show: false,
                smileys: [
                    ['😀', '😂', '🤔', '🥰', '💋', '❤️', '❤️‍🔥', '😍'],
                    ['😐', '🤒', '🤮', '🥴',  '😎', '😨', '👀', '🌚'],
                    ['😡', '👿', '💩', '😇',  '🤐', '🤪', '💣', '💧'],
                    ['👍',  '👎', '🤟', '🙏',  '💪', '👏', '🔥', '❄️'],
                ],
                addSmileyCallback: null,
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
            showModal(addSmileyCallback) {
                this.$data.show = true;
                this.addSmileyCallback = addSmileyCallback;
            },
            closeModal() {
                this.show = false;
                this.addSmileyCallback = null;
            },
            onSmileyClick(smiley) {
                if (this.addSmileyCallback) {
                    this.addSmileyCallback(smiley);
                }
            },
        },
        created() {
            bus.on(OPEN_MESSAGE_EDIT_SMILEY, this.showModal);
        },
        destroyed() {
            bus.off(OPEN_MESSAGE_EDIT_SMILEY, this.showModal);
        },
    }
</script>

<style lang="stylus" scoped>
    .smiley-buttons {
        button {
            color: rgba(0, 0, 0, 1) !important
        }
    }
    .smiley {
      margin-left 4px
      margin-right 4px

      margin-top 4px
      margin-bottom 2px

      cursor: pointer

      font-size: 2.125rem !important;
      font-weight: 400;
      line-height: 2.5rem;
      letter-spacing: 0.0073529412em !important;
    }

    .smiley:hover {
      background #0d47a1
    }
</style>
