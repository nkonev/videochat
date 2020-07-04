<template>
    <div>It's a chat #{{id}}</div>
</template>

<script>
    export default {
        computed: {
            id() {
                return this.$route.params.id
            }
        }
    }
</script>