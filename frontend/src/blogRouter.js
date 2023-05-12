import Vue from 'vue'
import Router from 'vue-router'
import Error404 from "./Error404";
import {profile_name} from "@/routes";
import {blog, blog_post_name} from "@/blogRoutes";
const BlogList = () => import("./BlogList");
const BlogPost = () => import("./BlogPost");

// This installs <router-view> and <router-link>,
// and injects $router and $route to all router-enabled child components
// WARNING You shouldn't include it in tests, else avoriaz's globals won't works (https://github.com/eddyerburgh/avoriaz/issues/124)
Vue.use(Router);

const router = new Router({
    mode: 'history',
    // https://router.vuejs.org/en/api/options.html#routes
    routes: [
        { name: "blog", path: blog, component: BlogList},
        { name: blog_post_name, path: '/blog/post/:id', component: BlogPost},
        { name: profile_name, path: '/profile/:id'},
        { path: '*', component: Error404 },
    ]
});

export default router;
