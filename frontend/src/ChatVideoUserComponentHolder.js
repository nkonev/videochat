export class ChatVideoUserComponentHolder {
    #userVideoComponents = new Map();

    addComponentForUser(userIdentity, component) {
        let existingList = this.#userVideoComponents.get(userIdentity);
        if (!existingList) {
            this.#userVideoComponents.set(userIdentity, []);
            existingList = this.#userVideoComponents.get(userIdentity);
        }
        existingList.push(component);
    }

    removeComponentForUser(userIdentity, component) {
        let existingList = this.#userVideoComponents.get(userIdentity);
        if (existingList) {
            for(let i = 0; i < existingList.length; i++){
                if (existingList[i].getId() == component.getId()) {
                    existingList.splice(i, 1);
                }
            }
        }
        if (existingList.length == 0) {
            this.#userVideoComponents.delete(userIdentity);
        }
    }

    isEmpty() {
        return this.#userVideoComponents.size == 0
    }

    getByUser(userIdentity) {
        let existingList = this.#userVideoComponents.get(userIdentity);
        if (!existingList) {
            this.#userVideoComponents.set(userIdentity, []);
            existingList = this.#userVideoComponents.get(userIdentity);
        }
        return existingList;
    }

}