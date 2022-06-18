export class ChatVideoUserComponentHolder {
    #userVideoComponents = new Map();

    addComponentForUser(userIdentity, component) {
        let existingList = this.#userVideoComponents[userIdentity];
        if (!existingList) {
            existingList = this.#userVideoComponents[userIdentity] = [];
        }
        existingList.push(component);
    }

    removeComponentForUser(userIdentity, component) {
        let existingList = this.#userVideoComponents[userIdentity];
        if (existingList) {
            for(let i = 0; i < existingList.length; i++){
                if (existingList[i].getId() == component.getId()) {
                    existingList.splice(i, 1);
                }
            }
        }
        if (existingList.length == 0) {
            this.#userVideoComponents.delete(userIdentity)
        }
    }

    isEmpty() {
        return this.#userVideoComponents.size == 0
    }

    getByUser(userIdentity) {
        let existingList = this.#userVideoComponents[userIdentity];
        if (!existingList) {
            existingList = this.#userVideoComponents[userIdentity] = [];
        }
        return existingList;
    }

    /**
     * Very heavy complexity
     * @param callback
     */
    invokeOnAllComponents(callback) {
        for (const userId in this.#userVideoComponents) {
            const existingList = this.#userVideoComponents[userId];
            if (existingList) {
                for(const component of existingList){
                    callback(component);
                }
            }
        }
    }
}