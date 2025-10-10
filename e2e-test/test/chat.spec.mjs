import { test, expect } from '@playwright/test';
import {defaultVkontakteUser, defaultGoogleUser, recreateAaaOauth2MocksUrl, truncateChatUrl} from "../constants.mjs";
import Login from "../models/Login.mjs";
import ChatList from "../models/ChatList.mjs";
import ChatView from "../models/ChatView.mjs"
import axios from "axios";
import fs from "fs";

let googleContext;
let vkContext;

let googlePage;
let vkPage;

let tmpVideoGoogle;
let tmpVideoVk;
// https://github.com/microsoft/playwright/issues/14164#issuecomment-1131451544

test.beforeEach(async ({}, testInfo) => {
    tmpVideoGoogle = testInfo.outputPath('video-behalf-google');
    tmpVideoVk = testInfo.outputPath('video-behalf-vk');
})

test.afterEach(async ({}, testInfo) => {
    const videoPathGoogle = testInfo.outputPath('video-google.webm');
    const videoPathVk = testInfo.outputPath('video-vk.webm');
    await Promise.all([
        googlePage.video().saveAs(videoPathGoogle),
        googlePage.close(),

        vkPage.video().saveAs(videoPathVk),
        vkPage.close(),
    ]);
    testInfo.attachments.push({
        name: 'video-google',
        path: videoPathGoogle,
        contentType: 'video/webm'
    });
    testInfo.attachments.push({
        name: 'video-vk',
        path: videoPathVk,
        contentType: 'video/webm'
    });

    fs.rmSync(tmpVideoGoogle, { recursive: true });
    fs.rmSync(tmpVideoVk, { recursive: true });

    // Gracefully close up everything
    await googleContext.close();
    await vkContext.close();
});

// https://playwright.dev/docs/intro
test('login vkontakte and google then create chat then write a message', async ({ browser }) => {
    googleContext = await browser.newContext();
    vkContext = await browser.newContext();

    googlePage = await browser.newPage({
        recordVideo: {
            dir: tmpVideoGoogle,
        }
    });

    vkPage = await browser.newPage({
        recordVideo: {
            dir: tmpVideoVk,
        }
    });

    await axios.put(recreateAaaOauth2MocksUrl);
    await axios.delete(truncateChatUrl);

    const googleLoginPage = new Login(googlePage);
    await googleLoginPage.navigate();
    await googleLoginPage.submitGoogle();
    await googleLoginPage.assertNickname(defaultGoogleUser.user);

    const vkLoginPage = new Login(vkPage);
    await vkLoginPage.navigate();
    await vkLoginPage.submitVkontakte();
    await vkLoginPage.assertNickname(defaultVkontakteUser.user);

    const chatName = "test chatto login and write msgs";
    const vkChatList = new ChatList(vkPage);
    await vkChatList.openNewChatDialog();
    await vkChatList.createAndSubmit(chatName, [defaultGoogleUser.user]);
    const vkChatViewPage = new ChatView(vkPage);
    const helloMike = "Hello, Mike!";
    await vkChatViewPage.sendMessage(helloMike);

    const googleChatList = new ChatList(googlePage);
    await googleChatList.navigate();
    await expect(googleChatList.getRowsLocator().nth(0)).toHaveText(chatName);
    const googleChatsCount = await (googleChatList.getRowsLocator().count());
    console.log("count behalf google is", googleChatsCount);
    expect(googleChatsCount).toBe(1);

    // sometimes browser ignores click, so we repeat it
    // https://stackoverflow.com/questions/69806337/how-to-pause-the-test-script-for-3-seconds-before-continue-running-it-playwrigh/79637287#79637287
    await expect
        .poll(async () => {
            await googleChatList.openChat(0);
            return await googleChatList.getHeaderLocator().textContent();
        })
        .toBe(chatName);

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


test('login vkontakte and google then create chat then join into publicly awailable', async ({ browser }) => {
    googleContext = await browser.newContext();
    vkContext = await browser.newContext();

    googlePage = await browser.newPage({
        recordVideo: {
            dir: tmpVideoGoogle,
        }
    });

    vkPage = await browser.newPage({
        recordVideo: {
            dir: tmpVideoVk,
        }
    });

    await axios.put(recreateAaaOauth2MocksUrl);
    await axios.delete(truncateChatUrl);

    const googleLoginPage = new Login(googlePage);
    await googleLoginPage.navigate();
    await googleLoginPage.submitGoogle();
    await googleLoginPage.assertNickname(defaultGoogleUser.user);

    const vkLoginPage = new Login(vkPage);
    await vkLoginPage.navigate();
    await vkLoginPage.submitVkontakte();
    await vkLoginPage.assertNickname(defaultVkontakteUser.user);

    const chatName = "test chatto join into publiclly available";
    const vkChatList = new ChatList(vkPage);
    await vkChatList.openNewChatDialog();
    await vkChatList.createAndSubmit(chatName, [], true);
    const vkChatViewPage = new ChatView(vkPage);
    const helloMike = "Hello, Mike!";
    await vkChatViewPage.sendMessage(helloMike);

    const googleChatList = new ChatList(googlePage);
    await googleChatList.navigate(true);
    await expect(googleChatList.getRowsLocator().nth(0)).toHaveText(chatName);
    const googleChatsCount = await (googleChatList.getRowsLocator().count());
    console.log("count behalf google is", googleChatsCount);
    expect(googleChatsCount).toBe(1);

    // sometimes browser ignores click, so we repeat it
    // https://stackoverflow.com/questions/69806337/how-to-pause-the-test-script-for-3-seconds-before-continue-running-it-playwrigh/79637287#79637287
    await expect
        .poll(async () => {
            await googleChatList.openChat(0);
            return await googleChatList.getHeaderLocator().textContent();
        })
        .toBe(chatName);

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

