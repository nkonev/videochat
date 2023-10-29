import {deepCopy, isMobileBrowser} from "@/utils";

export default (name) => {
  const qName = name + "contextMenu"
  return {
    data(){
      return {
        showContextMenuObj: false,
        menuableItem: null,
        contextMenuX: 0,
        contextMenuY: 0,
      }
    },
    computed: {
      showContextMenu: {
          get() {
              if (isMobileBrowser()) {
                  return !!this.$route.query[qName]
              } else {
                  return this.showContextMenuObj
              }
          },
          set(v) {
              if (isMobileBrowser()) {
                  if (v) {
                      this.$router.push({
                          query: {
                              [qName]: true
                          }
                      }).then(()=>{
                          this.setPosition()
                      })
                  } else {
                      const prev = deepCopy(this.$route.query);
                      delete prev[qName];
                      this.$router.push({query: prev});
                  }
              } else {
                  this.showContextMenuObj = v;
                  if (v) {
                      this.$nextTick(()=>{
                          this.setPosition()
                      })
                  }
              }
          }
      }
    },
    methods: {
      setPosition() {
        const element = document.querySelector("." + this.className() + " .v-overlay__content");
        if (element) {
          element.style.position = "absolute";
          element.style.top = this.contextMenuY + "px";
          element.style.left = this.contextMenuX + "px";

          const bottom = Number(getComputedStyle(element).bottom.replace("px", ''));
          if (bottom < 0) {
            const newTop = this.contextMenuY + bottom - 8;
            element.style.top = newTop + "px";
          }

          const width = Number(getComputedStyle(element).width.replace("px", ''));
          if (width < 260) {
              const delta = Math.abs(260 - width);
              const newLeft = this.contextMenuX - delta - 8;
              element.style.left = newLeft + "px";
          }
        }
      },
      onShowContextMenuBase(e, menuableItem) {
        e.preventDefault();
        this.contextMenuX = e.clientX;
        this.contextMenuY = e.clientY;

        this.menuableItem = menuableItem;

        this.$nextTick(() => {
          this.showContextMenu = true;
        })
      },
      onCloseContextMenuBase() {
        this.showContextMenu = false;
        this.menuableItem = null;
      },
      onUpdate(v) {
        if (!v) {
            this.onCloseContextMenu();
        }
      },
    }
  }
}
