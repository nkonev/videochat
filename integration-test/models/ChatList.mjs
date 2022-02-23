import {webUiUrl} from "../constants.mjs";
import {expect} from "@playwright/test";

// https://playwright.dev/docs/pom
export default class ChatList {
    constructor(page, ) {
        this.page = page;
    }
    async navigate() {
        await this.page.goto(webUiUrl);
    }
    async openNewChatDialog() {
        const dialog = this.page.locator('#new-chat-dialog-button');
        await dialog.click();
        const form = this.page.locator('.v-dialog .v-form');
        await expect(form).toBeVisible();
    }

    async createAndSubmit(chatName) {
        await this.page.fill('.v-dialog .v-form #new-chat-text', chatName);
        const submit = this.page.locator('.v-dialog #chat-save-button');
        await submit.click();
        const form = this.page.locator('.v-dialog .v-form');
        await expect(form).not.toBeVisible();
    }


}
