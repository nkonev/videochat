import {webUiUrl} from "../constants.mjs";
import {expect} from "@playwright/test";

// https://playwright.dev/docs/pom
export default class Login {
    constructor(page, user, password) {
        this.page = page;
        this.user = user;
        this.password = password;
    }
    async navigate() {
        await this.page.goto(webUiUrl);
    }
    async submitLogin() {
        const submit = this.page.locator('.v-dialog .v-form #button-login');
        await this.page.fill('.v-dialog .v-form #text-login', this.user);
        await this.page.fill('.v-dialog .v-form #text-password', this.password);
        await expect(submit).toBeVisible();
        await submit.click();
    }
}
