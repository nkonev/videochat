import { test, expect } from '@playwright/test';
import {defaultVkontakteUser, defaultGoogleUser, recreateAaaOauth2MocksUrl} from "../constants.mjs";
import Login from "../models/Login.mjs";
import axios from "axios";

// https://playwright.dev/docs/intro
test('login vkontakte and google and create chat', async ({ browser }) => {
    await axios.post(recreateAaaOauth2MocksUrl);

    /*const vkContext = await browser.newContext();
    const vkPage = await vkContext.newPage();
    const vkLoginPage = new Login(vkPage);
    await vkLoginPage.navigate();
    await vkLoginPage.submitVkontakte();
    await vkLoginPage.assertNickname(defaultVkontakteUser.user)*/

    const googleContext = await browser.newContext();
    const googlePage = await googleContext.newPage();
    const googleLoginPage = new Login(googlePage);
    await googleLoginPage.navigate();
    await googleLoginPage.submitGoogle();
    await googleLoginPage.assertNickname(defaultGoogleUser.user)

});

