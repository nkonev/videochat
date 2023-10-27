class VVP {
    constructor() {
        this.enabled = typeof (window.visualViewport) === "object";
        if (!this.enabled) {
            console.error("Visual Viewport is not available in this browser.");
        }
        this.vvp = {w: 0, h: 0};
        this.vp = {w: 0, h: 0};
        this.create_style_element();
        this.refresh();

        window.visualViewport.addEventListener('resize', this.refresh.bind(this));
    }

    get style_element() {
        return document.getElementById("viewport_fix_variables");
    }

    get calculate_viewport() {
        return new Promise((resolve, reject) => {
            if (!this.enabled) {
                return reject("Could not calculate window.visualViewport");
            }
            this.vvp.w = window.visualViewport.width;
            this.vvp.h = window.visualViewport.height;

            this.vp.w = Math.max(document.documentElement.clientWidth || 0, window.innerWidth || 0);
            this.vp.h = Math.max(document.documentElement.clientHeight || 0, window.innerHeight || 0);
            return resolve();
        })
    }

    refresh() {
        return this.calculate_viewport.then(this.set_viewport()).catch((e) => console.error(e));
    }

    create_style_element() {
        const style_tag = document.createElement("style");
        style_tag.id = "viewport_fix_variables";
        return document.head.prepend(style_tag);
    }

    set_viewport() {
        return this.style_element.innerHTML = `
      :root {
        --100vvw: ${this.vvp.w}px;
        --100vvh: ${this.vvp.h}px;
        
        --offset-w: ${this.vp.w - this.vvp.w}px;
        --offset-h: ${this.vp.h - this.vvp.h}px;
      }
    `;
    }
}
