import { test, expect } from '@playwright/test';
import {defaultAdminUser, defaultVkontakteUser, defaultWrongUser, recreateAaaOauth2MocksUrl} from "../constants.mjs";
import Login from "../models/Login.mjs";
import axios from "axios";

test('login successful', async ({ page }) => {
    const loginPage = new Login(page, defaultAdminUser.user, defaultAdminUser.password);
    await loginPage.navigate();
    await loginPage.submitLogin();

    await loginPage.assertNickname(defaultAdminUser.user)
});

test('login unsuccessful', async ({ page }) => {
    const loginPage = new Login(page, defaultWrongUser.user, defaultWrongUser.password);
    await loginPage.navigate();
    await loginPage.submitLogin();

    await loginPage.assertWrongLogin();
});

test('login vkontakte', async ({ page }) => {
    await axios.post(recreateAaaOauth2MocksUrl)

    const loginPage = new Login(page);
    await loginPage.navigate();
    await loginPage.submitVkontakte();

    await loginPage.assertNickname(defaultVkontakteUser.user)
});


