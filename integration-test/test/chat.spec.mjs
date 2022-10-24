import { test, expect } from '@playwright/test';
import {defaultVkontakteUser, defaultGoogleUser, recreateAaaOauth2MocksUrl, removeChatParticipantsUrl} from "../constants.mjs";
import Login from "../models/Login.mjs";
import ChatList from "../models/ChatList.mjs";
import axios from "axios";

// https://playwright.dev/docs/intro
test('login vkontakte and google and create chat', async ({ browser }) => {
    await axios.post(recreateAaaOauth2MocksUrl);
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

    await vkChatList.openNewChatDialog();
    await vkChatList.createAndSubmit(chatName+" trash", []);

    // https://playwright.dev/docs/locators
    await vkChatList.assertChatItemCount(2);
    await vkChatList.assertChatName(chatName);

    const googleChatList = new ChatList(googlePage);
    await expect(googleChatList.getRowsLocator().nth(0)).toHaveText(chatName);
    const googleChatsCount = await (googleChatList.getRowsLocator().count());
    console.log("count behalf google is", googleChatsCount);
    expect(googleChatsCount).toBe(1);
});

