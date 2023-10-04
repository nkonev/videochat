<template>
    <span @click="onClick()">Blog list</span>
</template>

<script>
    import {blog_post_name} from "@/router/blogRoutes";

    export default {
        methods: {
            onClick() {
                this.$router.push({name: blog_post_name})
            }
        }
    }
</script>
