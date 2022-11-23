<template>
    <v-row justify="center">
        <v-dialog v-model="show" max-width="640" scrollable>
            <v-card>
                <v-card-title>{{ $vuetify.lang.t('$vuetify.notifications') }}</v-card-title>
                <v-card-text class="ma-0 pa-0">
                    <v-list class="pb-0">
                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-at</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>Sed ut perspiciatis, unde omnis iste natus error sit voluptatem accusantium doloremque laudantium, totam rem aperiam eaque ipsa, quae ab illo inventore veritatis et quasi architecto beatae vitae dicta sunt, explicabo. Nemo enim ipsam voluptatem, quia voluptas sit, aspernatur aut odit aut fugit, sed quia consequuntur magni dolores eos, qui ratione voluptatem sequi nesciunt, neque porro quisquam est, qui dolorem ipsum, quia dolor sit amet consectetur adipisci[ng] velit, sed quia non numquam [do] eius modi tempora inci[di]dunt, ut labore et dolore magnam aliquam quaerat voluptatem. Ut enim ad minima veniam, quis nostrum[d] exercitationem ullam corporis suscipit laboriosam, nisi ut aliquid ex ea commodi consequatur? [D]Quis autem vel eum i[r]ure reprehenderit, qui in ea voluptate velit esse, quam nihil molestiae consequatur, vel illum, qui dolorem eum fugiat, quo voluptas nulla pariatur? [33] At vero eos et accusamus et iusto odio dignissimos ducimus, qui blanditiis praesentium voluptatum deleniti atque corrupti, quos dolores et quas molestias excepturi sint, obcaecati cupiditate non provident, similique sunt in culpa, qui officia deserunt mollitia animi, id est laborum et dolorum fuga. Et harum quidem rerum facilis est et expedita distinctio. Nam libero tempore, cum soluta nobis est eligendi optio, cumque nihil impedit, quo minus id, quod maxime placeat, facere possimus, omnis voluptas assumenda est, omnis dolor repellendus. Temporibus autem quibusdam et aut officiis debitis aut rerum necessitatibus saepe eveniet, ut et voluptates repudiandae sint et molestiae non recusandae. Itaque earum rerum hic tenetur a sapiente delectus, ut aut reiciendis voluptatibus maiores alias consequatur aut perferendis doloribus asperiores repellat.</v-list-item-title>
                                <v-list-item-subtitle>{{ $vuetify.lang.t('$vuetify.notification_mention') }}</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone-missed</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>{{ $vuetify.lang.t('$vuetify.notification_missed_call') }}</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>

                        <v-list-item link>
                            <v-list-item-icon class="mr-4"><v-icon>mdi-phone</v-icon></v-list-item-icon>
                            <v-list-item-content class="py-0">
                                <v-list-item-title>To take a trivial example, which of us ever undertakes laborious physical exercise, except to obtain some advantage from it? But who has any right to find fault with a man who chooses to enjoy a pleasure that has no annoying consequences, or one who avoids a pain that produces no resultant pleasure?</v-list-item-title>
                                <v-list-item-subtitle>Missed call</v-list-item-subtitle>
                            </v-list-item-content>
                        </v-list-item>
                    </v-list>
                </v-card-text>
                <v-divider/>
                <v-card-actions>
                    <v-switch
                        :label="$vuetify.lang.t('$vuetify.notify_about_mentions')"
                        dense
                        hide-details
                        class="ma-0 ml-2 mr-4"
                    ></v-switch>
                    <v-switch
                        :label="$vuetify.lang.t('$vuetify.notify_about_missed_calls')"
                        dense
                        hide-details
                        class="ma-0 ml-2 mr-4"
                    ></v-switch>
                    <v-spacer></v-spacer>

                    <v-btn
                        color="error"
                        @click="closeModal()"
                    >
                        {{ $vuetify.lang.t('$vuetify.close') }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-row>
</template>

<script>

import bus, {
    OPEN_NOTIFICATIONS_DIALOG,
} from "./bus";
import axios from "axios";

export default {
    data () {
        return {
            show: false,
            dto: null,
            loading: false,
        }
    },

    methods: {
        showModal() {
            this.show = true;
            // TODO uncomment
            //  this.fetchNotifications();
        },
        fetchNotifications() {
            if (!this.show) {
                return
            }
            this.loading = true;
            axios.get(`/api/notifications`)
                .then((response) => {
                    this.dto = response.data;
                })
                .finally(() => {
                    this.loading = false;
                })
        },
        closeModal() {
            this.show = false;
        },

    },
    watch: {
        show(newValue) {
            if (!newValue) {
                this.closeModal();
            }
        }
    },
    created() {
        bus.$on(OPEN_NOTIFICATIONS_DIALOG, this.showModal);
    },
    destroyed() {
        bus.$off(OPEN_NOTIFICATIONS_DIALOG, this.showModal);
    },
}
</script>
