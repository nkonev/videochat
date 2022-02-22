import { test, expect } from '@playwright/test';
import { defaultAdminUser, defaultWrongUser} from "../constants.mjs";
import Login from "../models/Login.mjs";
import axios from "axios";

test('login successful', async ({ page }) => {
    const loginPage = new Login(page, defaultAdminUser.user, defaultAdminUser.password);
    await loginPage.navigate();
    await loginPage.submitLogin();

    await expect(page.locator('#chat-list-items')).toBeVisible();
    const count = await page.locator('#chat-list-items .v-list-item').count();
    expect(count).toBeGreaterThanOrEqual(1);
});

test('login unsuccessful', async ({ page }) => {
    const loginPage = new Login(page, defaultWrongUser.user, defaultWrongUser.password);
    await loginPage.navigate();
    await loginPage.submitLogin();

    const alertLocator = page.locator('.v-dialog .v-form .v-alert');
    await expect(alertLocator).toBeVisible();
    await expect(alertLocator).toHaveText("Wrong login or password");
});

test('login vkontakte', async ({ page }) => {
    await axios.post('http://localhost:9080/recreate-oauth2-mocks')

    const loginPage = new Login(page);
    await loginPage.navigate();
    await loginPage.submitVkontakte();

    await expect(page.locator('.v-navigation-drawer .user-login')).toHaveText("Никита Конев")
});


