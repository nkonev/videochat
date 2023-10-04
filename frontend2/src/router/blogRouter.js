import {createRouter, createWebHistory} from "vue-router";
import {blog, blog_name} from "@/router/blogRoutes";

const routes = [
  {
    name: blog_name,
    path: blog,
    component: () => import('@/BlogList.vue'),
  },

]

const router = createRouter({
  history: createWebHistory(process.env.BASE_URL),
  routes,
})

export default router
