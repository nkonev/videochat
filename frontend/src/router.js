import Vue from 'vue'
import Router from 'vue-router'
import goTo from 'vuetify/lib/services/goto'
import {chat_list_name, root, chat_name, profile_self_name, videochat_name, profile_name, video_suffix} from "./routes";
import Error404 from "./Error404";
import ChatList from "./ChatList";
import bus, {CLOSE_SIMPLE_MODAL, OPEN_SIMPLE_MODAL} from "@/bus";
import vuetify from "@/plugins/vuetify";
const ChatView = () => import("./ChatView.vue");
const UserSelfProfile = () => import("./UserSelfProfile");
const UserProfile = () => import("./UserProfile");

// This installs <router-view> and <router-link>,
// and injects $router and $route to all router-enabled child components
// WARNING You shouldn't include it in tests, else avoriaz's globals won't works (https://github.com/eddyerburgh/avoriaz/issues/124)
Vue.use(Router);

const router = new Router({
    scrollBehavior: (to, from, savedPosition) => {
        let scrollTo = 0;

        const scrollerDiv = document.getElementById("messagesScroller"); // ChatView.vue specific
        let options = null;
        if (scrollerDiv && to.hash) {
            scrollTo = to.hash;
            options = {container: scrollerDiv, duration: 0};
        } else if (savedPosition) {
            scrollTo = savedPosition.y;
        }

        try {
            return goTo(scrollTo, options)
        } catch (e) {
            console.log("Ignoring missing element", e);
        }
    },
    mode: 'history',
    // https://router.vuejs.org/en/api/options.html#routes
    routes: [
        { name: chat_list_name, path: root, component: ChatList},
        { name: chat_name, path: '/chat/:id', component: ChatView},
        { name: videochat_name, path: '/chat/:id' + video_suffix, component: ChatView},
        { name: profile_self_name, path: '/profile', component: UserSelfProfile},
        { name: profile_name, path: '/profile/:id', component: UserProfile},
        { path: '*', component: Error404 },
    ]
});

router.beforeEach((to, from, next) => {
    if (from.name == videochat_name && to.name != videochat_name && from.params.id && from.params.id != to.params.id && to.params.leavingVideoAcceptableParam != true) {
        bus.$emit(OPEN_SIMPLE_MODAL, {
            buttonName: vuetify.framework.lang.translator('$vuetify.ok'),
            title: vuetify.framework.lang.translator('$vuetify.leave_call'),
            text: vuetify.framework.lang.translator('$vuetify.leave_call_text'),
            actionFunction: ()=> {
                next();
                bus.$emit(CLOSE_SIMPLE_MODAL);
            },
            cancelFunction: ()=>{
                next(false)
            }
        });
    } else {
        next();
    }
});

export default router;
