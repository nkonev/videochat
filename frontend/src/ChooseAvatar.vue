<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="500" persistent>
            <v-card>
                <v-card-title>Choose avatar</v-card-title>

                <v-container fluid>
                    <v-row justify="center">
                    <croppa v-model="myCroppa"
                            :width="400"
                            :height="400"
                            :remove-button-size="32"

                            :file-size-limit="limit"
                            :show-loading="true"
                            placeholder="Choose avatar image"
                            :initial-image="initialImage"
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
                    <v-btn color="primary" class="mr-4" @click="saveAvatar()">Choose</v-btn>
                    <v-btn color="error" class="mr-4" @click="show=false">Close</v-btn>
                    <v-spacer/>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>
    import axios from "axios";
    import bus, {OPEN_CHOOSE_AVATAR} from "./bus";
    import {FETCH_USER_PROFILE, GET_USER} from "./store";
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

                myCroppa: {},
                removeImage: false,
                imageContentType: null,
                imageChanged: false,
            }
        },
        computed: {
            initialImage() {
                const user = this.$store.getters[GET_USER];
                if (user) {
                    return user.avatar
                } else {
                    return null
                }
            },
            limit() {
                return UPLOAD_FILE_SIZE_LIMIT;
            }
        },
        methods: {
            initialImageInternal(){
                let url;
                if (!this.initialImage) {
                    url = this.initialImage;
                } else if (typeof this.initialImage === "function") {
                    url = this.initialImage();
                } else if (typeof this.initialImage === "string") {
                    url = this.initialImage;
                } else {
                    throw "Allowed string or function for prop initialImage"
                }
                console.log("Initial image", url);
                if (url) {
                    axios.head(url).then(resp => {
                        // once set imageContentType for able to change image
                        let raw = resp.headers.get("content-type");
                        if (raw) {
                            let arr = raw.split(";");
                            raw = arr[0];
                        }
                        this.imageContentType = raw;
                        console.log("Initial image content-type", this.imageContentType);
                    });
                }
                return url;
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
                if (!blob) {
                    return Promise.resolve(false);
                }

                const config = {
                    headers: { 'content-type': 'multipart/form-data' }
                }
                console.log("Sending avatar to storage");
                const formData = new FormData();
                formData.append('data', blob);
                return axios.post('/api/storage/avatar', formData, config)
            },
            saveAvatar() {
                this.createBlob().then(this.sendAvatar).then((res) => {
                    if (!res) {
                        return axios.patch(`/api/profile`, {removeAvatar: true})
                    } else {
                        return axios.patch(`/api/profile`, {avatar: res.data.relativeUrl})
                    }
                }).then(value => {
                    console.log("PATCH result", value);
                    this.$store.dispatch(FETCH_USER_PROFILE);
                    this.show = false;
                });
            },

            showModal() {
                console.log("Reseiving open avatar");
                this.$data.show = true;
            },
        },
        created() {
            bus.$on(OPEN_CHOOSE_AVATAR, this.showModal);
        },
        destroyed() {
            bus.$off(OPEN_CHOOSE_AVATAR, this.showModal);
        },
    }
</script>