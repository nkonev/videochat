import {getBrowserNotification, getGlobalBrowserNotification} from "@/store/localStore.js";
import {chat_name, messageIdHashPrefix} from "@/router/routes.js";
import {hasLength} from "@/utils.js";
import {isInteger, isString} from "lodash";

export const createNotification = (title, body, type) => {
    new Notification(title, { body: body, icon: "/favicon_new.svg", tag: type });
}

const notifications = {}

export const createNotificationIfPermitted = (router, chatId, chatName, chatAvatar, messageId, messageText, type) => {
    const shouldGlobalBrowserNotification = getGlobalBrowserNotification(type);
    const shouldChatBrowserNotification = getBrowserNotification(chatId, null, type);
    let decision = shouldGlobalBrowserNotification;
    if (shouldChatBrowserNotification !== null) {
        decision = shouldChatBrowserNotification;
    }

    if (Notification?.permission === "granted" && decision) {
        const notificationObject = { icon: hasLength(chatAvatar) ? chatAvatar : "/favicon_new.svg", tag: type };
        if (hasLength(chatName)) {
            notificationObject.body = chatName
        }
        const notification = new Notification(
            messageText,
            notificationObject,
        );

        const shouldAddMessageId = hasLength(`${messageId}`);
        let hash = undefined;
        if (shouldAddMessageId) {
            hash = messageIdHashPrefix + messageId;
        }

        notification.onclick = () => {
            const routeObj = {
                name: chat_name,
                params: {
                    id: chatId
                },
                hash: hash,
            };

            router.push(routeObj);
        }
        notifications[type] = notification;
    }
}

export const removeNotification = (type) => {
    notifications[type]?.close()
    notifications[type] = null;
}
