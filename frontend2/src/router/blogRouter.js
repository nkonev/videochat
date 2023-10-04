import {createRouter, createWebHistory} from "vue-router";
import {blog, blog_name, blog_post, blog_post_name} from "@/router/blogRoutes";

const routes = [
    {
        name: blog_name,
        path: blog,
        component: () => import('@/BlogList.vue'),
    },
    {
        name: blog_post_name,
        path: blog_post,
        component: () => import('@/BlogPost.vue'),
    },

]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
})

export default router
