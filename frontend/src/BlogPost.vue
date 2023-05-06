<template>
    <div>
        <h1>Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. </h1>
        <div class="message-item-text">
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
            "Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum."
        </div>
        <v-divider/>
        <v-list>
            <!-- TODO MessageList.vue -->
            <template v-for="(item, index) in items">
                <MessageItem
                    :key="item.id"
                    :item="item"
                    :chatId="chatId"
                    :my="true"
                ></MessageItem>
            </template>
        </v-list>
    </div>
</template>

<script>
    import MessageItem from "@/MessageItem";

    export default {
        data() {
            return {
                chatId: 34,
                items: [
                    {
                        "id": 2822,
                        "text": "<p>Testw</p>",
                        "chatId": 34,
                        "ownerId": 1,
                        "createDateTime": "2023-04-13T16:06:21.569617Z",
                        "editDateTime": "2023-05-06T01:02:18.329121Z",
                        "owner": {
                            "id": 1,
                            "login": "admin",
                            "avatar": "/api/storage/public/user/avatar/1_AVATAR_200x200.jpg?time=1676930657",
                            "shortInfo": "Admin account"
                        },
                        "canEdit": true,
                        "canDelete": true,
                        "fileItemUuid": "17b13878-2a0d-4860-a6d6-2ad8545901b0",
                        "embedMessage": null,
                        "pinned": false
                    },
                    {
                        "id": 2823,
                        "text": "<p>Lorem</p>",
                        "chatId": 34,
                        "ownerId": 1,
                        "createDateTime": "2023-04-13T16:06:21.569617Z",
                        "editDateTime": "2023-05-06T01:02:18.329121Z",
                        "owner": {
                            "id": 1,
                            "login": "admin",
                            "avatar": "/api/storage/public/user/avatar/1_AVATAR_200x200.jpg?time=1676930657",
                            "shortInfo": "Admin account"
                        },
                        "canEdit": true,
                        "canDelete": true,
                        "fileItemUuid": "17b13878-2a0d-4860-a6d6-2ad8545901b0",
                        "embedMessage": null,
                        "pinned": false
                    },
                    {
                        "id": 2824,
                        "text": "<b>Ipsum</b>",
                        "chatId": 34,
                        "ownerId": 1,
                        "createDateTime": "2023-04-13T16:06:21.569617Z",
                        "editDateTime": "2023-05-06T01:02:18.329121Z",
                        "owner": {
                            "id": 1,
                            "login": "admin",
                            "avatar": "/api/storage/public/user/avatar/1_AVATAR_200x200.jpg?time=1676930657",
                            "shortInfo": "Admin account"
                        },
                        "canEdit": true,
                        "canDelete": true,
                        "fileItemUuid": "17b13878-2a0d-4860-a6d6-2ad8545901b0",
                        "embedMessage": null,
                        "pinned": false
                    },
                    {
                        "id": 2825,
                        "text": "<p>Dolor</p>",
                        "chatId": 34,
                        "ownerId": 1,
                        "createDateTime": "2023-04-13T16:06:21.569617Z",
                        "editDateTime": "2023-05-06T01:02:18.329121Z",
                        "owner": {
                            "id": 1,
                            "login": "admin",
                            "avatar": "/api/storage/public/user/avatar/1_AVATAR_200x200.jpg?time=1676930657",
                            "shortInfo": "Admin account"
                        },
                        "canEdit": true,
                        "canDelete": true,
                        "fileItemUuid": "17b13878-2a0d-4860-a6d6-2ad8545901b0",
                        "embedMessage": null,
                        "pinned": false
                    },
                ]
            }
        },
        components: {
            MessageItem
        }
    }
</script>
