<template>
    <div>
        <h1 v-html="blogDto.title"></h1>

        <div class="pr-1 mr-1 pl-1 mt-0 message-item-root" >
            <div class="message-item-with-buttons-wrapper">
                <v-list-item class="grow" v-if="blogDto?.owner">
                    <a @click.prevent="onParticipantClick(blogDto.owner)" :href="getProfileLink(blogDto.owner)">
                        <v-list-item-avatar>
                            <v-img
                                class="elevation-6"
                                alt=""
                                :src="blogDto.owner.avatar"
                            ></v-img>
                        </v-list-item-avatar>
                    </a>

                    <div class="ma-0 pa-0 d-flex top-panel">
                        <v-list-item-content>
                            <v-list-item-title><a @click.prevent="onParticipantClick(blogDto.owner)" :href="getProfileLink(blogDto.owner)">{{blogDto.owner.login}}</a></v-list-item-title>
                            <v-list-item-subtitle>{{getDate(blogDto.createDateTime)}}</v-list-item-subtitle>
                        </v-list-item-content>
                        <div class="ma-0 pa-0 go-to-chat">
                            <v-btn icon :href="getChatLink()" @click="toChat()" :title="$vuetify.lang.t('$vuetify.go_to_chat')"><v-icon dark>mdi-forum</v-icon></v-btn>
                        </div>
                    </div>
                </v-list-item>


                <div class="pa-0 ma-0 mt-1 message-item-wrapper post-content">
                    <v-container v-html="blogDto.text" class="message-item-text ml-0"></v-container>
                </div>
            </div>
        </div>

        <template v-if="blogDto.messageId">
            <v-list id="comment-list">
                <template v-for="(item, index) in items">
                    <MessageItem
                        :key="item.id"
                        :item="item"
                        :chatId="item.chatId"
                        :isInBlog="true"
                    ></MessageItem>
                </template>
            </v-list>

            <infinite-loading @infinite="infiniteHandler" force-use-infinite-wrapper="#comment-list">
                <template slot="no-more"><span/></template>
                <template slot="no-results"><span/></template>
            </infinite-loading>
        </template>
    </div>
</template>

<script>
    import axios from "axios";
    import MessageItem from "@/MessageItem";
    import {SET_SHOW_SEARCH} from "@/blogStore";
    import {getHumanReadableDate, hasLength, replaceOrAppend} from "@/utils";
    import {chat, messageIdHashPrefix, profile, profile_name} from "@/routes";
    import InfiniteLoading from "@/lib/vue-infinite-loading/src/components/InfiniteLoading";

    const pageSize = 40;

    const blogDtoFactory = () => {
        return {
            chatId: 0
        }
    }

    export default {
        data() {
            return {
                blogDto: blogDtoFactory(),
                startingFromItemId: null,
                page: 0,
                items: [ ]
            }
        },
        methods: {
            onParticipantClick(user) {
                const routeDto = { name: profile_name, params: { id: user.id }};
                this.$router.push(routeDto);
            },
            getProfileLink(user) {
                let url = profile + "/" + user.id;
                return url;
            },
            getChatLink() {
                return chat + '/' + this.blogDto.chatId + messageIdHashPrefix + this.blogDto.messageId;
            },
            toChat() {
                window.location.href = this.getChatLink();
            },
            getBlog(id) {
                axios.get('/api/blog/'+id).then(({data}) => {
                    this.blogDto = data;
                    this.startingFromItemId = data.messageId;
                });
            },
            getDate(date) {
                if (hasLength(date)) {
                    return getHumanReadableDate(date)
                } else {
                    return null
                }
            },
            infiniteHandler($state) {
                if (this.items.length) {
                    this.startingFromItemId = Math.max(...this.items.map(it => it.id));
                    console.log("this.startingFromItemId set to", this.startingFromItemId);
                }

                axios.get(`/api/blog/${this.blogDto.chatId}/message`, {
                    params: {
                        startingFromItemId: this.startingFromItemId,
                        page: this.page,
                        size: pageSize,
                    },
                }).then(({ data }) => {
                    const list = data;
                    if (list.length) {
                        this.page += 1;
                        replaceOrAppend(this.items, list);
                        $state.loaded();
                    } else {
                        $state.complete();
                    }
                });
            },
        },
        components: {
            MessageItem,
            InfiniteLoading,
        },
        mounted() {
            this.$store.commit(SET_SHOW_SEARCH, false);
            this.getBlog(this.$route.params.id);
        },
        beforeDestroy() {
            this.blogDto = blogDtoFactory();
            this.startingFromItemId = null;
            this.page = 0;
            this.items = [];
        },
    }
</script>

<style lang="stylus">
    @import "common.styl"

    .top-panel {
        width 100%
    }

    .go-to-chat {
        align-self center
    }

    .post-content {
        background white
        border-color $borderColor
        border-style: dashed;
        border-width 1px
        box-shadow:
            0 0 0 1px rgba(0, 0, 0, 0.05),
            0px 10px 20px rgba(0, 0, 0, 0.1)
    }
</style>
