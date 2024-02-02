<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="600px">
            <v-card :title="aTitle">
                <v-card-text class="py-0 pt-2 px-4 smiley-buttons">

                    <v-btn @click="onSmileyClick(smiley)" v-for="smiley in smileys" variant="flat" class="smiley" height="42px" width="42px" min-width="unset">{{smiley}}</v-btn>

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
                    '😀', '😂', '🤔', '🥰', '💋', '❤️', '❤️‍🔥', '😍',
                    '😐', '🤒', '🤮', '🥴',  '😎', '😨', '👀', '🌚',
                    '😡', '👿', '💩', '😇',  '🤐', '🤪', '💣', '💧',
                    '👍',  '👎', '🤟', '🙏', '💪', '👏', '🔥', '❄️',
                    '🍎',  '🍅', '🍊', '🍒', '🤑', '🤗', '🤠', '💔',
                    '💯',  '💥', '💤', '👋', '🖖', '🫴', '🫸', '🫷',
                    '👌', '🤌',  '🤞', '🫵', '🪲', '🐞', '🌹', '🌎',
                    '🎯', '♟️', '♠️', '♥️', '♦️', '♣️', '💊', '🧲',
                    '🩹', '🗿', '⚠️',  '⛔', '☢️', '☣️', '♻️', '✅',
                    '❌', '⚡', '🚀', '#️⃣', '*️⃣', '0️⃣', '1️⃣',
                    '2️⃣', '3️⃣', '4️⃣', '5️⃣', '6️⃣', '7️⃣',
                    '8️⃣', '9️⃣', '🔟', '©', '™', '®'
                ],
                addSmileyCallback: null,
                aTitle: null,
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
            showModal({addSmileyCallback, title}) {
                this.$data.show = true;
                this.addSmileyCallback = addSmileyCallback;
                this.aTitle = title;
            },
            closeModal() {
                this.show = false;
                this.addSmileyCallback = null;
                this.aTitle = null;
            },
            onSmileyClick(smiley) {
                if (this.addSmileyCallback) {
                    this.addSmileyCallback(smiley);
                }
            },
        },
        mounted() {
            bus.on(OPEN_MESSAGE_EDIT_SMILEY, this.showModal);
        },
        beforeUnmount() {
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
      font-size: 2.125rem !important;
    }

    .smiley:hover {
      background #0d47a1
    }
</style>
