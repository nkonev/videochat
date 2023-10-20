import { test, expect } from '@playwright/test';
import {defaultVkontakteUser, defaultGoogleUser, recreateAaaOauth2MocksUrl, removeChatParticipantsUrl} from "../constants.mjs";
import Login from "../models/Login.mjs";
import ChatList from "../models/ChatList.mjs";
import ChatView from "../models/ChatView.mjs"
import axios from "axios";

// https://playwright.dev/docs/intro
test('login vkontakte and google then create chat then write a message', async ({ browser }) => {
    await axios.put(recreateAaaOauth2MocksUrl);
    await axios.delete(removeChatParticipantsUrl);

    const googleContext = await browser.newContext();
    const googlePage = await googleContext.newPage();
    const googleLoginPage = new Login(googlePage);
    await googleLoginPage.navigate();
    await googleLoginPage.submitGoogle();
    await googleLoginPage.assertNickname(defaultGoogleUser.user);

    const vkContext = await browser.newContext();
    const vkPage = await vkContext.newPage();
    const vkLoginPage = new Login(vkPage);
    await vkLoginPage.navigate();
    await vkLoginPage.submitVkontakte();
    await vkLoginPage.assertNickname(defaultVkontakteUser.user);

    const chatName = "test chatto";
    const vkChatList = new ChatList(vkPage);
    await vkChatList.openNewChatDialog();
    await vkChatList.createAndSubmit(chatName, [defaultGoogleUser.user]);
    const vkChatViewPage = new ChatView(vkPage);
    const helloMike = "Hello, Mike!";
    await vkChatViewPage.sendMessage(helloMike);

    const googleChatList = new ChatList(googlePage);
    googleChatList.navigate();
    await expect(googleChatList.getRowsLocator().nth(0)).toHaveText(chatName);
    const googleChatsCount = await (googleChatList.getRowsLocator().count());
    console.log("count behalf google is", googleChatsCount);
    expect(googleChatsCount).toBe(1);
    await googleChatList.openChat(0);

    const googleChatViewPage = new ChatView(googlePage);
    const receivedMikeMessage = await googleChatViewPage.getMessage(0);
    expect(receivedMikeMessage).toBe(helloMike);
    const helloJoe = "Hello, Joe!";
    await googleChatViewPage.sendMessage(helloJoe);

    const receivedMikeMessageVk = await vkChatViewPage.getMessage(1);
    expect(receivedMikeMessageVk).toBe(helloMike);

    const receivedJoeMessage = await vkChatViewPage.getMessage(0);
    expect(receivedJoeMessage).toBe(helloJoe);
});

