import { test, expect } from '@playwright/test';
import {defaultVkontakteUser, defaultGoogleUser, recreateAaaOauth2MocksUrl} from "../constants.mjs";
import Login from "../models/Login.mjs";
import ChatList from "../models/ChatList.mjs";
import axios from "axios";

// https://playwright.dev/docs/intro
test('login vkontakte and google and create chat', async ({ browser }) => {
    await axios.post(recreateAaaOauth2MocksUrl);

    const vkContext = await browser.newContext();
    const vkPage = await vkContext.newPage();
    const vkLoginPage = new Login(vkPage);
    await vkLoginPage.navigate();
    await vkLoginPage.submitVkontakte();
    await vkLoginPage.assertNickname(defaultVkontakteUser.user);

    const googleContext = await browser.newContext();
    const googlePage = await googleContext.newPage();
    const googleLoginPage = new Login(googlePage);
    await googleLoginPage.navigate();
    await googleLoginPage.submitGoogle();
    await googleLoginPage.assertNickname(defaultGoogleUser.user);

    const chatName = "test chatto";
    const vkChatList = new ChatList(vkPage);
    await vkChatList.openNewChatDialog();
    await vkChatList.createAndSubmit(chatName);

    await vkChatList.openNewChatDialog();
    await vkChatList.createAndSubmit(chatName+"_trash");

    // await expect(vkPage.locator('#chat-list-items')).toBeVisible();
    // const locator = vkPage.locator('#chat-list-items .v-list-item');
    // await expect(locator.count()).toBe(2);
    // const firstRow = await locator.first().textContent();
    // expect(firstRow).toHaveValue(chatName);

    const vkRows = vkPage.locator('#chat-list-items .v-list-item .v-list-item__title');
    // https://playwright.dev/docs/locators
    const vkCount = await vkRows.count()
    for (let i = 0; i < vkCount; ++i) {
        console.log(await vkRows.nth(i).textContent());
    }
    expect(vkCount).toBe(2);
    const vkSecondRow = (await vkRows.nth(1).textContent()).trim();
    expect(vkSecondRow).toBe(chatName);
});

