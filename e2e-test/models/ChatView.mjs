export default class ChatView {
    constructor(page) {
        this.page = page;
    }

    async sendMessage(message) {
        await this.page.fill('#sendButtonContainer .editorContent .ProseMirror', message);
        const sendButton = this.page.locator('#sendButtonContainer button.send');
        await sendButton.click();
    }

    async getMessage(index) {
        return (await this.page.locator('#messagesScroller .message-item-text').nth(index).textContent()).trim()
    }

}
