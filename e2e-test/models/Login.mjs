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
        const submit = this.page.locator('.v-dialog .v-form #login-btn');
        await this.page.fill('.v-dialog .v-form #login-text', this.user);
        await this.page.fill('.v-dialog .v-form #password-text', this.password);
        await expect(submit).toBeVisible();
        await submit.click();
    }

    async submitVkontakte() {
        const submit = this.page.locator('.v-dialog .v-form .c-btn-vk');
        await expect(submit).toBeVisible();
        await submit.click();
    }

    async submitGoogle() {
        const submit = this.page.locator('.v-dialog .v-form .c-btn-google');
        await expect(submit).toBeVisible();
        await submit.click();
    }

    assertWrongLogin() {
        const alertLocator = this.page.locator('.v-dialog .v-form .v-alert');
        return expect(alertLocator).toBeVisible().then(() => {
            return expect(alertLocator).toHaveText("Wrong login or password");
        });
    }

    async assertNickname(expected) {
        return expect(this.page.locator('.v-navigation-drawer__content #right-panel-user-login .v-list-item-title')).toHaveText(expected)
    }

}
