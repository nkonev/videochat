<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="600px" scrollable>
            <v-card :title="getTitle()">
                <v-expansion-panels v-model="chosenPanel" v-if="showSettings" class="pt-2" @update:modelValue="generateGroupSmileys">
                  <v-expansion-panel
                    v-for="i in groups"
                    :key="i.id"
                    :title="i.title"
                    :value="i.id"
                  >
                    <template v-slot:text>
                      <div class="smiley-buttons">
                        <v-btn @click="onSmileySettingClick(smiley)" v-for="smiley in groupSmileys" :variant="getVariant(smiley)" class="smiley" height="42px" width="42px" min-width="unset">{{smiley}}</v-btn>
                      </div>
                    </template>
                  </v-expansion-panel>
                </v-expansion-panels>

                <v-card-text class="py-0 pt-2 px-4 smiley-buttons" v-else>
                    <template v-if="!loading">
                      <v-btn @click="onSmileyClick(smiley)" v-for="smiley in userSmileys" variant="flat" class="smiley" height="42px" width="42px" min-width="unset">{{smiley}}</v-btn>
                    </template>

                    <v-progress-circular
                      class="ma-4"
                      v-else
                      indeterminate
                      color="primary"
                    ></v-progress-circular>

                </v-card-text>

                <v-card-actions>
                    <v-spacer/>
                    <v-btn v-if="showSettings" color="primary" variant="flat" @click="closeSettings()" :title="$vuetify.locale.t('$vuetify.ok')">{{$vuetify.locale.t('$vuetify.ok')}}</v-btn>
                    <v-btn v-if="!showSettings" variant="outlined" @click="openSettings()" min-width="0" :title="$vuetify.locale.t('$vuetify.settings')"><v-icon size="large">mdi-cog</v-icon></v-btn>
                    <v-btn v-if="!showSettings" color="red" variant="flat" @click="closeModal()">{{ $vuetify.locale.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
import bus, {LOGGED_OUT, OPEN_MESSAGE_EDIT_SMILEY} from "./bus/bus";
    import axios from "axios";

    const GROUP_SMILEYS = "smileys";
    const GROUP_EMOJIS = "emojis";
    const GROUP_ADDITIONAL_3 = "additional_3";
    const GROUP_ADDITIONAL_4 = "additional_4";
    const GROUP_ADDITIONAL_5 = "additional_5";
    const GROUP_ADDITIONAL_6 = "additional_6";
    const GROUP_ADDITIONAL_7 = "additional_7";
    const GROUP_ADDITIONAL_8 = "additional_8";
    const GROUP_ADDITIONAL_9 = "additional_9";
    const GROUP_ADDITIONAL_10 = "additional_10";
    const GROUP_ADDITIONAL_11 = "additional_11";

    export default {
        data () {
            return {
                show: false,
                userSmileys: new Set([]),
                addSmileyCallback: null,
                aTitle: null,
                groups: [
                  { id: GROUP_SMILEYS, title: "Smileys" },
                  { id: GROUP_EMOJIS, title: "Emojis" },
                  { id: GROUP_ADDITIONAL_3, title: "Additional 3" },
                  { id: GROUP_ADDITIONAL_4, title: "Additional 4" },
                  { id: GROUP_ADDITIONAL_5, title: "Additional 5" },
                  { id: GROUP_ADDITIONAL_6, title: "Additional 6" },
                  { id: GROUP_ADDITIONAL_7, title: "Additional 7" },
                  { id: GROUP_ADDITIONAL_8, title: "Additional 8" },
                  { id: GROUP_ADDITIONAL_9, title: "Additional 9" },
                  { id: GROUP_ADDITIONAL_10, title: "Additional 10" },
                  { id: GROUP_ADDITIONAL_11, title: "Additional 11" },
                ],
                chosenPanel: null,
                showSettings: false,
                groupSmileys: [],
                loading: false,
            }
        },
        watch: {
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            },
        },
        methods: {
            showModal({addSmileyCallback, title}) {
                this.$data.show = true;
                this.addSmileyCallback = addSmileyCallback;
                this.aTitle = title;

                if (this.userSmileys.size == 0) {
                  this.loading = true;
                  axios.get('/api/aaa/settings/smileys').then((response) => {
                    this.userSmileys = new Set(response.data);
                  }).finally(()=>{
                    this.loading = false;
                  })
                }
            },
            closeModal() {
                this.show = false;
                this.addSmileyCallback = null;
                this.aTitle = null;
                this.showSettings = false;
                this.groupSmileys = [];
                this.chosenPanel = null;
                this.loading = false;
            },
            onLogout() {
                this.closeModal();
                this.userSmileys = new Set([]);
            },
            onSmileyClick(smiley) {
                if (this.addSmileyCallback) {
                    this.addSmileyCallback(smiley);
                }
            },
            openSettings() {
              this.showSettings = true;
            },
            closeSettings() {
              this.showSettings = false;
            },
            onSmileySettingClick(smiley) {
              if (this.userSmileys.has(smiley)) {
                this.userSmileys.delete((smiley));
              } else {
                this.userSmileys.add(smiley)
              }
              axios.put('/api/aaa/settings/smileys', Array.from(this.userSmileys)).then((response)=>{
                this.userSmileys = new Set(response.data);
              })
            },
            getVariant(smiley) {
              if (this.userSmileys.has(smiley)) {
                return 'tonal'
              } else {
                return 'flat'
              }
            },
            getTitle() {
                if (!this.showSettings) {
                  return this.aTitle
                } else {
                  return this.$vuetify.locale.t('$vuetify.configuring_smileys')
                }
            },
            // https://stackoverflow.com/a/73993544
            // https://stackoverflow.com/questions/30470079/emoji-value-range
            generateEmoji(ch) {
              let hex = ch.toString(16)
              let emo = String.fromCodePoint("0x"+hex);
              return emo
            },
            generateEmojis(from, to) {
              const emojis = [];
              for (var i = from; i <= to; i++) {
                let emo = this.generateEmoji(i);
                emojis.push(emo);
              }
              return emojis
            },
            generateGroupSmileys(group) {
              switch (group) {
                case GROUP_SMILEYS: {
                  this.groupSmileys = this.generateEmojis(0x1F600, 0x1F64F);
                  break
                }
                case GROUP_EMOJIS: {
                  this.groupSmileys = this.generateEmojis(0x1F980, 0x1F9E0);
                  break
                }
                case GROUP_ADDITIONAL_3: {
                  this.groupSmileys = this.generateEmojis(0x1F910, 0x1F96B);
                  break
                }
                case GROUP_ADDITIONAL_4: {
                  this.groupSmileys = this.generateEmojis(0x23E9, 0x23F3);
                  break
                }
                case GROUP_ADDITIONAL_5: {
                  this.groupSmileys = this.generateEmojis(0x23F8, 0x23FA);
                  break
                }
                case GROUP_ADDITIONAL_6: {
                  this.groupSmileys = this.generateEmojis(0x25FB, 0x25FE);
                  break
                }
                case GROUP_ADDITIONAL_7: {
                  this.groupSmileys = this.generateEmojis(0x1F100, 0x1F64F);
                  break
                }
                case GROUP_ADDITIONAL_8: {
                  this.groupSmileys = this.generateEmojis(0x1F680, 0x1F6FF);
                  break
                }
                case GROUP_ADDITIONAL_9: {
                  this.groupSmileys = this.generateEmojis(0x2600, 0x27EF);
                  break
                }
                case GROUP_ADDITIONAL_10: {
                  this.groupSmileys = this.generateEmojis(0x2B00, 0x2BFF);
                  break
                }
                case GROUP_ADDITIONAL_11: {
                  this.groupSmileys = [
                    '💡','☎️', '🧲',
                    '#️⃣', '*️⃣',
                    '0️⃣', '1️⃣', '2️⃣', '3️⃣', '4️⃣', '5️⃣', '6️⃣', '7️⃣', '8️⃣', '9️⃣', '🔟',
                    this.generateEmoji(0x231A), // watches
                    this.generateEmoji(0x231B), // sand watches

                    this.generateEmoji(0x00A9), // (c)
                    this.generateEmoji(0x00AE), // (r)
                    this.generateEmoji(0x2122), // (tm)
                  ];
                  break
                }
              }
            }
        },
        mounted() {
            bus.on(OPEN_MESSAGE_EDIT_SMILEY, this.showModal);
            bus.on(LOGGED_OUT, this.onLogout);
        },
        beforeUnmount() {
            bus.off(OPEN_MESSAGE_EDIT_SMILEY, this.showModal);
            bus.off(LOGGED_OUT, this.onLogout);
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
