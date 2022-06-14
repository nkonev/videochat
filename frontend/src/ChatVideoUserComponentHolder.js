export class ChatVideoUserComponentHolder {
    #userVideoComponents = new Map();

    addComponentForUser(userId, component) {
        let existingList = this.#userVideoComponents[userId];
        if (!existingList) {
            existingList = this.#userVideoComponents[userId] = [];
        }
        existingList.push(component);
    }

    removeComponentForUser(userId, component) {
        let existingList = this.#userVideoComponents[userId];
        if (existingList) {
            for(let i = 0; i < existingList.length; i++){
                if (existingList[i].getId() == component.getId()) {
                    existingList.splice(i, 1);
                }
            }
        }
    }

    getByUser(userId) {
        let existingList = this.#userVideoComponents[userId];
        if (!existingList) {
            existingList = this.#userVideoComponents[userId] = [];
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