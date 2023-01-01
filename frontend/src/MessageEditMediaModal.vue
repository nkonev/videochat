<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card>
                <v-card-title>{{ title() }}</v-card-title>

                <v-card-text>
                    <v-row dense v-if="!loading">
                        <template v-if="dto.count > 0">
                            <v-col
                                v-for="mediaFile in dto.files"
                                :key="mediaFile.id"
                                :cols="6"
                            >
                                <v-hover>
                                    <template v-slot:default="{ hover }">
                                        <v-card>
                                            <v-img
                                                :src="mediaFile.previewUrl"
                                                class="white--text align-end"
                                                gradient="to bottom, rgba(0,0,0,.1), rgba(0,0,0,.5)"
                                                height="200px"
                                            >
                                                <v-card-title v-text="mediaFile.filename"></v-card-title>
                                            </v-img>

                                            <v-fade-transition>
                                                <v-overlay
                                                    v-if="hover"
                                                    absolute
                                                    @click="accept(mediaFile)"
                                                    style="cursor: pointer"
                                                >
                                                    {{ $vuetify.lang.t('$vuetify.click_to_choose') }}
                                                </v-overlay>
                                            </v-fade-transition>

                                        </v-card>
                                    </template>
                                </v-hover>
                            </v-col>
                        </template>
                        <template v-else>
                            <v-card-text>{{ $vuetify.lang.t('$vuetify.no_files') }}</v-card-text>
                        </template>
                    </v-row>

                    <v-progress-circular
                        class="ma-4 pa-4"
                        v-else
                        indeterminate
                        color="primary"
                    ></v-progress-circular>
                </v-card-text>

                <v-card-actions class="d-flex flex-wrap flex-row">
                    <v-pagination
                        v-if="shouldShowPagination"
                        v-model="filePage"
                        :length="filePagesCount"
                    ></v-pagination>
                    <v-spacer></v-spacer>
                    <v-btn color="primary" class="mr-2" @click="fromDisk()"><v-icon color="white">mdi-file-upload</v-icon>{{ $vuetify.lang.t('$vuetify.choose_file_from_disk') }}</v-btn>
                    <v-btn color="error" @click="closeModal()">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";

    import bus, {OPEN_MESSAGE_EDIT_MEDIA} from "./bus";
    import {media_image, media_video} from "@/utils";

    const firstPage = 1;
    const pageSize = 20;

    const dtoFactory = () => {return {files: []} };

    export default {
        data () {
            return {
                show: false,
                type: '',
                fromDiskCallback: null,
                setExistingMediaCallback: null,
                loading: false,
                dto: dtoFactory(),
                filePage: firstPage,
            }
        },
        watch: {
            show(newValue) {
                if (!newValue) {
                    this.closeModal();
                }
            },
            filePage(newValue) {
                if (this.show) {
                    console.debug("SettingNewPage", newValue);
                    this.dto = dtoFactory();
                    this.updateFiles();
                }
            },
        },
        computed: {
            filePagesCount() {
                const count = Math.ceil(this.dto.count / pageSize);
                console.debug("Calc pages count", count);
                return count;
            },
            shouldShowPagination() {
                return this.dto != null && this.dto.files && this.dto.count > pageSize
            },
            chatId() {
                return this.$route.params.id
            },
        },
        methods: {
            showModal(type, fromDiskCallback, setExistingMediaCallback) {
                this.$data.show = true;
                this.type = type;
                this.fromDiskCallback = fromDiskCallback;
                this.setExistingMediaCallback = setExistingMediaCallback;
                this.updateFiles();
            },
            accept(item) {
                if (this.setExistingMediaCallback) {
                    this.setExistingMediaCallback(item.url, item.previewUrl)
                }
                this.closeModal();
            },
            clear() {
                this.closeModal();
            },
            closeModal() {
                this.show = false;
                this.type = '';
                this.fromDiskCallback = null;
                this.setExistingMediaCallback = null;
                this.loading = false;
                this.dto = dtoFactory();
                this.filePage = firstPage;
            },
            title() {
                switch (this.type) {
                    case media_video:
                        return this.$vuetify.lang.t('$vuetify.message_edit_video')
                    case media_image:
                        return this.$vuetify.lang.t('$vuetify.message_edit_image')
                }
            },
            fromDisk() {
                if (this.fromDiskCallback) {
                    this.fromDiskCallback();
                }
                this.closeModal();
            },
            translatePage() {
                return this.filePage - 1;
            },
            updateFiles() {
                if (!this.show) {
                    return
                }
                this.loading = true;
                axios.get(`/api/storage/${this.chatId}/embed/list`, {
                    params: {
                        page: this.translatePage(),
                        size: pageSize,
                        type: this.type
                    },
                })
                    .then((response) => {
                        this.dto = response.data;
                    })
                    .finally(() => {
                        this.loading = false;
                    })
            },
        },
        created() {
            bus.$on(OPEN_MESSAGE_EDIT_MEDIA, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_MESSAGE_EDIT_MEDIA, this.showModal);
        },
    }
</script>
