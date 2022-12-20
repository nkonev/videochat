<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="500" persistent>
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.user_profile_choose_avatar') }}</v-card-title>

                <v-container fluid>
                    <v-row justify="center">
                    <croppa :key="croppaKey"
                            v-model="myCroppa"
                            :width="400"
                            :height="400"
                            :remove-button-size="32"

                            :file-size-limit="limit"
                            :show-loading="true"
                            placeholder="Choose avatar image"
                            :initial-image="initialImage()"
                            :placeholder-font-size="32"
                            :disabled="false"
                            :prevent-white-space="true"
                            :show-remove-button="true"
                            accept="image/*"
                            @file-choose="handleCroppaFileChoose"
                            @image-remove="handleCroppaImageRemove"
                            @file-size-exceed="handleCroppaFileSizeExceed"
                            @file-type-mismatch="handleCroppaFileTypeMismatch"

                            @move="handleImageChanged"
                            @zoom="handleImageChanged"
                    >
                    </croppa>
                    </v-row>
                </v-container>

                <v-card-actions class="pa-4">
                    <v-btn color="primary" class="mr-4" @click="saveAvatar()" :loading="uploading" :disabled="uploading">{{ $vuetify.lang.t('$vuetify.ok') }}</v-btn>
                    <v-btn color="error" class="mr-4" @click="closeModal()" :disabled="uploading">{{ $vuetify.lang.t('$vuetify.close') }}</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import bus, {OPEN_CHOOSE_AVATAR} from "./bus";
    import 'vue-croppa/dist/vue-croppa.css'
    import Croppa from 'vue-croppa'

    const UPLOAD_FILE_SIZE_LIMIT = 10000000;

    export default {
        components: {
            'croppa': Croppa.component,
        },
        data() {
            return {
                show: false,

                croppaKey: 1,
                myCroppa: {},
                removeImage: false,
                imageContentType: null,
                imageChanged: false,
                uploading: false,

                initialAvatarCallback: null,
                uploadAvatarFileCallback: null,
                removeAvatarUrlCallback: null,
                storeAvatarUrlCallback: null,
                onSuccessCallback: null,
            }
        },
        computed: {
            limit() {
                return UPLOAD_FILE_SIZE_LIMIT;
            }
        },
        methods: {
            initialImage() {
                if (this.$data.initialAvatarCallback) {
                    return this.$data.initialAvatarCallback();
                } else {
                    return null;
                }
            },
            handleCroppaFileChoose(e){
                this.removeImage = false;
                this.imageContentType = e.type;
                this.handleImageChanged();
                console.debug('image chosen', e);
            },
            handleCroppaImageRemove(){
                console.debug('image removed');
                this.removeImage = true;
                this.imageContentType = null;
            },
            handleImageChanged(){
                this.$data.imageChanged = true;
                console.debug('image changed', this.$data.imageChanged);
            },
            handleCroppaFileSizeExceed(){
                alert(`Image size must be < than ${UPLOAD_FILE_SIZE_LIMIT} bytes`);
            },
            handleCroppaFileTypeMismatch(){
                alert('Image wrong type');
            },

            createBlob(){
                if (this.$data.imageChanged) {
                    console.debug("Invoking next() with blob of type", this.imageContentType);
                    return this.myCroppa.promisedBlob(this.imageContentType);
                } else {
                    console.debug("Invoking next() without blob");
                    return Promise.resolve(false);
                }
            },


            sendAvatar(blob) {
                return this.$data.uploadAvatarFileCallback(blob);
            },
            saveAvatar() {
                this.uploading = true;
                this.createBlob().then(this.sendAvatar).then((res) => {
                    if (!res) { // no res - when createBlob() returned empty when user removed avatar
                        if (this.removeImage) {
                            return this.$data.removeAvatarUrlCallback();
                        } else {
                            return Promise.resolve(false);
                        }
                    } else {
                        return this.$data.storeAvatarUrlCallback(res);
                    }
                }).then(value => {
                    if (value && this.$data.onSuccessCallback) {
                        this.$data.onSuccessCallback();
                    }
                    this.closeModal();
                }).finally(() => {
                    this.uploading = false;
                });
            },

            showModal({initialAvatarCallback, uploadAvatarFileCallback, removeAvatarUrlCallback, storeAvatarUrlCallback, onSuccessCallback}) {
                this.croppaKey++;
                this.$data.show = true;
                this.$data.initialAvatarCallback = initialAvatarCallback;
                this.$data.uploadAvatarFileCallback = uploadAvatarFileCallback;
                this.$data.removeAvatarUrlCallback = removeAvatarUrlCallback;
                this.$data.storeAvatarUrlCallback = storeAvatarUrlCallback;
                this.$data.onSuccessCallback = onSuccessCallback;
            },
            closeModal() {
                this.$data.show=false;
                this.myCroppa = {};
                this.$data.initialAvatarCallback = null;
                this.$data.uploadAvatarFileCallback = null;
                this.$data.removeAvatarUrlCallback = null;
                this.$data.storeAvatarUrlCallback = null;
                this.$data.onSuccessCallback = null;
            }
        },
        created() {
            bus.$on(OPEN_CHOOSE_AVATAR, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_CHOOSE_AVATAR, this.showModal);
        },
    }
</script>